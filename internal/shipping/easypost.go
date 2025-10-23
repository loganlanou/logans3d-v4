package shipping

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/EasyPost/easypost-go/v5"
)

type EasyPostClient struct {
	client *easypost.Client
}

// Address structures compatible with our existing service layer
// These will be converted to/from EasyPost's native types

type EasyPostAddress struct {
	Name    string
	Street1 string
	Street2 string
	City    string
	State   string
	Zip     string
	Country string
	Phone   string
}

type EasyPostRate struct {
	ID              string
	Carrier         string
	Service         string
	Rate            float64
	Currency        string
	DeliveryDays    int
	DeliveryDate    string
	EstimatedDate   string
	CarrierNickname string
	ServiceType     string
}

type EasyPostLabel struct {
	ID             string
	TrackingCode   string
	LabelURL       string
	Rate           float64
	Currency       string
	CreatedAt      time.Time
	Status         string
	Refunded       bool
	RefundedAmount float64
}

func NewEasyPostClient() *EasyPostClient {
	apiKey := os.Getenv("EASYPOST_API_KEY")
	if apiKey == "" {
		// Return client anyway for mock mode
		return &EasyPostClient{client: nil}
	}

	client := easypost.New(apiKey)
	return &EasyPostClient{
		client: client,
	}
}

func (c *EasyPostClient) IsUsingMockData() bool {
	return c.client == nil
}

// GetRates retrieves shipping rates for a given shipment
func (c *EasyPostClient) GetRates(fromAddr Address, toAddr Address, pkg Package) ([]Rate, error) {
	if c.IsUsingMockData() {
		return c.getMockRates(pkg), nil
	}

	// Convert our Address type to EasyPost Address
	from := &easypost.Address{
		Name:    fromAddr.Name,
		Street1: fromAddr.AddressLine1,
		Street2: fromAddr.AddressLine2,
		City:    fromAddr.CityLocality,
		State:   fromAddr.StateProvince,
		Zip:     fromAddr.PostalCode,
		Country: fromAddr.CountryCode,
		Phone:   fromAddr.Phone,
	}

	to := &easypost.Address{
		Name:    toAddr.Name,
		Street1: toAddr.AddressLine1,
		Street2: toAddr.AddressLine2,
		City:    toAddr.CityLocality,
		State:   toAddr.StateProvince,
		Zip:     toAddr.PostalCode,
		Country: toAddr.CountryCode,
		Phone:   toAddr.Phone,
	}

	// Create parcel - EasyPost uses pounds for weight
	weightLbs := pkg.Weight.Value
	if pkg.Weight.Unit == "ounce" {
		weightLbs = pkg.Weight.Value / 16.0
	}

	parcel := &easypost.Parcel{
		Length: pkg.Dimensions.Length,
		Width:  pkg.Dimensions.Width,
		Height: pkg.Dimensions.Height,
		Weight: weightLbs,
	}

	// Create shipment
	// Note: EasyPost will automatically get rates from ALL configured carrier accounts
	// We don't specify carriers here - that's done in the API settings
	shipment := &easypost.Shipment{
		FromAddress: from,
		ToAddress:   to,
		Parcel:      parcel,
	}

	fmt.Printf("Creating EasyPost shipment: from=%s %s, to=%s %s, weight=%.2f lbs, dims=%.1fx%.1fx%.1f\n",
		from.City, from.Zip, to.City, to.Zip, weightLbs,
		pkg.Dimensions.Length, pkg.Dimensions.Width, pkg.Dimensions.Height)

	createdShipment, err := c.client.CreateShipment(shipment)
	if err != nil {
		fmt.Printf("EasyPost API error creating shipment: %v\n", err)
		return nil, fmt.Errorf("failed to create shipment: %w", err)
	}

	// Log the shipment for debugging
	fmt.Printf("EasyPost shipment created: ID=%s, Rates=%d\n", createdShipment.ID, len(createdShipment.Rates))
	if len(createdShipment.Rates) == 0 {
		fmt.Printf("EasyPost returned 0 rates. Shipment messages: %v\n", createdShipment.Messages)
	} else {
		// Log all carrier codes returned
		fmt.Printf("EasyPost returned rates from carriers: ")
		carriersSeen := make(map[string]bool)
		for _, rate := range createdShipment.Rates {
			carriersSeen[rate.Carrier] = true
		}
		for carrier := range carriersSeen {
			fmt.Printf("%s ", carrier)
		}
		fmt.Printf("\n")
	}

	// Convert EasyPost rates to our Rate type
	var rates []Rate
	for _, epRate := range createdShipment.Rates {
		// Parse rate from string to float64
		rateAmount := 0.0
		if epRate.Rate != "" {
			if parsed, err := strconv.ParseFloat(epRate.Rate, 64); err == nil {
				rateAmount = parsed
			}
		}

		// Convert delivery date to string
		deliveryDateStr := ""
		if epRate.DeliveryDate != nil {
			deliveryDateStr = epRate.DeliveryDate.String()
		}

		rate := Rate{
			RateID:          epRate.ID,
			CarrierID:       epRate.CarrierAccountID,
			CarrierCode:     epRate.Carrier,
			CarrierNickname: epRate.Carrier,
			ServiceCode:     epRate.Service,
			ServiceType:     epRate.Service,
			ShippingAmount: Amount{
				Currency: epRate.Currency,
				Amount:   rateAmount,
			},
			DeliveryDays:  epRate.DeliveryDays,
			EstimatedDate: deliveryDateStr,
		}
		rates = append(rates, rate)
	}

	return rates, nil
}

// CreateLabel purchases a shipping label for the given rate
func (c *EasyPostClient) CreateLabel(rateID string) (*Label, error) {
	if c.IsUsingMockData() {
		return c.getMockLabel(rateID), nil
	}

	// Extract shipment ID from rate ID if needed
	// In EasyPost, we need to buy the shipment with a rate
	// This requires us to have stored the shipment ID earlier
	// For now, we'll return an error indicating the need for a different approach
	return nil, fmt.Errorf("EasyPost requires shipment ID to buy label - refactor needed")
}

// BuyShipment purchases a label for a shipment using the specified rate
func (c *EasyPostClient) BuyShipment(shipmentID, rateID string) (*Label, error) {
	if c.IsUsingMockData() {
		return c.getMockLabel(rateID), nil
	}

	// Buy the shipment with the selected rate
	boughtShipment, err := c.client.BuyShipment(shipmentID, &easypost.Rate{ID: rateID}, "")
	if err != nil {
		return nil, fmt.Errorf("failed to buy shipment: %w", err)
	}

	// Parse rate amount from string
	rateAmount := 0.0
	if boughtShipment.SelectedRate != nil && boughtShipment.SelectedRate.Rate != "" {
		if parsed, err := strconv.ParseFloat(boughtShipment.SelectedRate.Rate, 64); err == nil {
			rateAmount = parsed
		}
	}

	// Convert created_at to time.Time
	createdAt := time.Now()
	if boughtShipment.CreatedAt != nil {
		createdAt = boughtShipment.CreatedAt.AsTime()
	}

	// Get label URLs
	labelPDFURL := ""
	labelURL := ""
	if boughtShipment.PostageLabel != nil {
		labelPDFURL = boughtShipment.PostageLabel.LabelPDFURL
		labelURL = boughtShipment.PostageLabel.LabelURL
	}

	// Convert to our Label type
	label := &Label{
		LabelID:        boughtShipment.ID,
		TrackingNumber: boughtShipment.TrackingCode,
		Status:         boughtShipment.Status,
		ShippingAmount: Amount{
			Currency: "usd",
			Amount:   rateAmount,
		},
		CarrierID:   boughtShipment.SelectedRate.CarrierAccountID,
		ServiceCode: boughtShipment.SelectedRate.Service,
		LabelDownload: LabelDownload{
			Hrefs: LabelHrefs{
				PDF: labelPDFURL,
				PNG: labelURL, // EasyPost uses LabelURL as the primary URL
			},
		},
		CreatedAt: createdAt,
	}

	return label, nil
}

// RefundLabel refunds/voids a shipping label
func (c *EasyPostClient) RefundLabel(shipmentID string) (bool, string, error) {
	if c.IsUsingMockData() {
		return true, "Mock refund approved", nil
	}

	// EasyPost uses Refund API
	refund, err := c.client.RefundShipment(shipmentID)
	if err != nil {
		return false, fmt.Sprintf("Refund failed: %v", err), err
	}

	approved := refund.Status == "submitted"
	message := fmt.Sprintf("Refund status: %s", refund.Status)

	return approved, message, nil
}

// VoidLabel is an alias for RefundLabel to maintain compatibility
func (c *EasyPostClient) VoidLabel(shipmentID string) (*VoidLabelResponse, error) {
	approved, message, err := c.RefundLabel(shipmentID)
	if err != nil {
		return nil, err
	}

	return &VoidLabelResponse{
		Approved: approved,
		Message:  message,
	}, nil
}

// GetCarriers returns available carriers
func (c *EasyPostClient) GetCarriers() (*CarriersResponse, error) {
	if c.IsUsingMockData() {
		return c.getMockCarriers(), nil
	}

	// EasyPost doesn't have a direct "list carriers" endpoint
	// Carriers are configured in the dashboard and available through carrier accounts
	// We'll return a mock list of common carriers
	return &CarriersResponse{
		Carriers: []Carrier{
			{CarrierID: "usps", CarrierCode: "USPS", CarrierNickname: "USPS"},
			{CarrierID: "ups", CarrierCode: "UPS", CarrierNickname: "UPS"},
			{CarrierID: "fedex", CarrierCode: "FedEx", CarrierNickname: "FedEx"},
		},
	}, nil
}

// DownloadLabelPDF downloads the label PDF from the URL
func (c *EasyPostClient) DownloadLabelPDF(label *Label) ([]byte, error) {
	// The label URL is already available, just return it
	// The client can download directly from the URL
	return nil, fmt.Errorf("use LabelDownload.Hrefs.PDF URL directly: %s", label.LabelDownload.Hrefs.PDF)
}

// Mock data functions for development without API key

func (c *EasyPostClient) getMockRates(pkg Package) []Rate {
	basePrice := 5.0 + (pkg.Weight.Value * 0.5) + (pkg.Dimensions.Length * pkg.Dimensions.Width * pkg.Dimensions.Height * 0.01)

	return []Rate{
		{
			RateID:          "mock-rate-usps-ground",
			CarrierID:       "usps",
			CarrierCode:     "USPS",
			CarrierNickname: "USPS",
			ServiceCode:     "GroundAdvantage",
			ServiceType:     "USPS Ground Advantage",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice},
			DeliveryDays:    5,
		},
		{
			RateID:          "mock-rate-usps-priority",
			CarrierID:       "usps",
			CarrierCode:     "USPS",
			CarrierNickname: "USPS",
			ServiceCode:     "Priority",
			ServiceType:     "USPS Priority Mail",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice + 4.50},
			DeliveryDays:    2,
		},
		{
			RateID:          "mock-rate-ups-ground",
			CarrierID:       "ups",
			CarrierCode:     "UPS",
			CarrierNickname: "UPS",
			ServiceCode:     "Ground",
			ServiceType:     "UPS Ground",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice + 2.50},
			DeliveryDays:    4,
		},
	}
}

func (c *EasyPostClient) getMockLabel(rateID string) *Label {
	return &Label{
		LabelID:        "mock-label-" + rateID,
		TrackingNumber: "MOCK1234567890",
		Status:         "created",
		ShippingAmount: Amount{Currency: "usd", Amount: 10.50},
		LabelDownload: LabelDownload{
			Hrefs: LabelHrefs{
				PDF: "https://example.com/mock-label.pdf",
			},
		},
		CreatedAt: time.Now(),
	}
}

func (c *EasyPostClient) getMockCarriers() *CarriersResponse {
	return &CarriersResponse{
		Carriers: []Carrier{
			{CarrierID: "usps", CarrierCode: "USPS", CarrierNickname: "USPS"},
			{CarrierID: "ups", CarrierCode: "UPS", CarrierNickname: "UPS"},
			{CarrierID: "fedex", CarrierCode: "FedEx", CarrierNickname: "FedEx"},
		},
	}
}
