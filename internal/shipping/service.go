package shipping

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

type ShippingOption struct {
	RateID          string  `json:"rate_id"`
	ShipmentID      string  `json:"shipment_id"`      // EasyPost shipment ID for label purchase
	CarrierName     string  `json:"carrier_name"`
	ServiceName     string  `json:"service_name"`
	Price           float64 `json:"price"`
	Currency        string  `json:"currency"`
	DeliveryDays    int     `json:"delivery_days"`
	EstimatedDate   string  `json:"estimated_date,omitempty"`
	BoxSKU          string  `json:"box_sku"`
	BoxCost         float64 `json:"box_cost"`
	TotalCost       float64 `json:"total_cost"`
	PackingSolution *PackingSolution `json:"packing_solution,omitempty"`
}

type ShippingQuoteRequest struct {
	ItemCounts ItemCounts `json:"item_counts"`
	ShipTo     Address    `json:"ship_to"`
}

type ShippingQuoteResponse struct {
	Options      []ShippingOption `json:"options"`
	DefaultOption *ShippingOption `json:"default_option,omitempty"`
	Error        string           `json:"error,omitempty"`
}

type ShippingService struct {
	config       *ShippingConfig
	client       *EasyPostClient
	packer       *Packer
	carrierIDs   []string
	carrierMap   map[string]Carrier // Maps carrier ID to carrier info
}

func NewShippingService(config *ShippingConfig) (*ShippingService, error) {
	client := NewEasyPostClient()
	packer := NewPacker(config)

	service := &ShippingService{
		config: config,
		client: client,
		packer: packer,
	}

	if err := service.loadCarrierIDs(); err != nil {
		// In development mode without API credentials, use mock data
		if client.IsUsingMockData() {
			service.carrierIDs = []string{"usps", "ups", "fedex"}
			return service, nil
		}
		return nil, fmt.Errorf("failed to load carrier IDs: %w", err)
	}

	return service, nil
}

func (s *ShippingService) loadCarrierIDs() error {
	carriersResp, err := s.client.GetCarriers()
	if err != nil {
		return fmt.Errorf("failed to get carriers from ShipStation: %w", err)
	}

	s.carrierIDs = make([]string, 0, len(carriersResp.Carriers))
	s.carrierMap = make(map[string]Carrier)
	for _, carrier := range carriersResp.Carriers {
		s.carrierIDs = append(s.carrierIDs, carrier.CarrierID)
		s.carrierMap[carrier.CarrierID] = carrier
	}

	return nil
}

func (s *ShippingService) GetShippingQuote(req *ShippingQuoteRequest) (*ShippingQuoteResponse, error) {
	slog.Debug("GetShippingQuote: Starting shipping quote calculation",
		"small_count", req.ItemCounts.Small,
		"medium_count", req.ItemCounts.Medium,
		"large_count", req.ItemCounts.Large,
		"xl_count", req.ItemCounts.XL,
		"ship_to_postal_code", req.ShipTo.PostalCode,
		"ship_to_state", req.ShipTo.StateProvince)

	packingSolution := s.packer.Pack(req.ItemCounts)
	if !packingSolution.Valid {
		slog.Debug("GetShippingQuote: Packing failed", "error", packingSolution.Error)
		return &ShippingQuoteResponse{
			Error: fmt.Sprintf("Unable to pack items: %s", packingSolution.Error),
		}, nil
	}

	slog.Debug("GetShippingQuote: Packing successful",
		"total_boxes", packingSolution.TotalBoxes,
		"total_cost", packingSolution.TotalCost)

	var allOptions []ShippingOption

	for _, boxSelection := range packingSolution.Boxes {
		rates, err := s.getRatesForBox(boxSelection, req.ShipTo)
		if err != nil {
			continue
		}

		for _, rate := range rates {
			totalCost := rate.ShippingAmount.Amount + boxSelection.BoxCost

			option := ShippingOption{
				RateID:          rate.RateID,
				ShipmentID:      rate.ShipmentID,
				CarrierName:     rate.CarrierNickname,
				ServiceName:     rate.ServiceType,
				Price:           rate.ShippingAmount.Amount,
				Currency:        rate.ShippingAmount.Currency,
				DeliveryDays:    rate.DeliveryDays,
				EstimatedDate:   rate.EstimatedDate,
				BoxSKU:          boxSelection.Box.SKU,
				BoxCost:         boxSelection.BoxCost,
				TotalCost:       totalCost,
				PackingSolution: packingSolution,
			}

			allOptions = append(allOptions, option)
		}
	}

	if len(allOptions) == 0 {
		return &ShippingQuoteResponse{
			Error: "No shipping options available",
		}, nil
	}

	sortedOptions := s.sortShippingOptions(allOptions)
	topN := s.config.Shipping.RatePreferences.PresentTopN

	slog.Debug("GetShippingQuote: Rate preferences",
		"configured_present_top_n", s.config.Shipping.RatePreferences.PresentTopN,
		"sort_order", s.config.Shipping.RatePreferences.Sort,
		"total_options_available", len(sortedOptions),
		"will_return_top_n", topN)

	if topN > len(sortedOptions) {
		topN = len(sortedOptions)
	}

	response := &ShippingQuoteResponse{
		Options: sortedOptions[:topN],
	}

	if len(response.Options) > 0 {
		response.DefaultOption = &response.Options[0]
	}

	return response, nil
}

func (s *ShippingService) getRatesForBox(boxSelection BoxSelection, shipTo Address) ([]Rate, error) {
	// If using mock data (no API credentials), return mock rates
	if s.client.IsUsingMockData() {
		return s.getMockRates(boxSelection, shipTo), nil
	}

	var allRates []Rate

	// Separate carriers into USPS and non-USPS
	uspsCarriers, otherCarriers := s.separateCarriersByType()

	// Get rates for USPS carriers from Cadott, WI (54727)
	if len(uspsCarriers) > 0 {
		uspsRates, err := s.getRatesForCarriers(uspsCarriers, boxSelection, shipTo, s.addressFromConfigUSPS())
		if err == nil {
			allRates = append(allRates, uspsRates...)
		}
	}

	// Get rates for non-USPS carriers from Eau Claire, WI (54701)
	if len(otherCarriers) > 0 {
		otherRates, err := s.getRatesForCarriers(otherCarriers, boxSelection, shipTo, s.addressFromConfigOther())
		if err == nil {
			allRates = append(allRates, otherRates...)
		}
	}

	if len(allRates) == 0 {
		return nil, fmt.Errorf("no rates available from any carrier")
	}

	return allRates, nil
}

// separateCarriersByType separates carrier IDs into USPS and non-USPS groups
func (s *ShippingService) separateCarriersByType() (usps []string, other []string) {
	for _, carrierID := range s.carrierIDs {
		// Check if this is a USPS carrier by checking if the carrier nickname or code contains "usps" or "stamps"
		// We'll make an API call to get carrier details, but for now use a simple heuristic
		// ShipStation USPS carrier IDs typically contain "stamps_com" or similar
		if s.isUSPSCarrier(carrierID) {
			usps = append(usps, carrierID)
		} else {
			other = append(other, carrierID)
		}
	}
	return usps, other
}

// isUSPSCarrier checks if a carrier ID represents a USPS carrier
func (s *ShippingService) isUSPSCarrier(carrierID string) bool {
	carrier, exists := s.carrierMap[carrierID]
	if !exists {
		// Simple check if not in map
		return carrierID == "usps" || carrierID == "USPS"
	}

	// Check carrier code and nickname for USPS identifiers
	code := carrier.CarrierCode
	nickname := carrier.CarrierNickname

	// EasyPost uses "USPS" for USPS carrier
	return code == "USPS" || code == "usps" ||
		nickname == "USPS"
}

// getRatesForCarriers gets rates for specific carriers from a specific origin
func (s *ShippingService) getRatesForCarriers(carrierIDs []string, boxSelection BoxSelection, shipTo Address, shipFrom Address) ([]Rate, error) {
	slog.Debug("getRatesForCarriers: Requesting rates",
		"box_sku", boxSelection.Box.SKU,
		"box_name", boxSelection.Box.Name,
		"box_dimensions_LxWxH", fmt.Sprintf("%.1fx%.1fx%.1f in", boxSelection.Box.L, boxSelection.Box.W, boxSelection.Box.H),
		"package_weight_oz", boxSelection.Weight,
		"box_weight_oz", boxSelection.Box.BoxWeightOz,
		"small_units", boxSelection.SmallUnits,
		"items_small", boxSelection.ItemCounts.Small,
		"items_medium", boxSelection.ItemCounts.Medium,
		"items_large", boxSelection.ItemCounts.Large,
		"items_xl", boxSelection.ItemCounts.XL,
		"ship_from_postal", shipFrom.PostalCode,
		"ship_to_postal", shipTo.PostalCode,
		"carriers", fmt.Sprintf("%v", carrierIDs))

	// Create package for EasyPost
	pkg := Package{
		PackageCode: "package",
		Weight: Weight{
			Value: boxSelection.Weight,
			Unit:  "ounce",
		},
		Dimensions: Dimensions{
			Length: boxSelection.Box.L,
			Width:  boxSelection.Box.W,
			Height: boxSelection.Box.H,
			Unit:   "inch",
		},
	}

	// Get rates from EasyPost
	rates, err := s.client.GetRates(shipFrom, shipTo, pkg)
	if err != nil {
		slog.Debug("getRatesForCarriers: Failed to get rates", "error", err)
		return nil, fmt.Errorf("failed to get rates: %w", err)
	}

	// Filter rates by requested carriers if not using mock data
	if !s.client.IsUsingMockData() && len(carrierIDs) > 0 {
		var filteredRates []Rate
		carrierSet := make(map[string]bool)
		for _, id := range carrierIDs {
			carrierSet[id] = true
			// Also add uppercase version for case-insensitive matching
			carrierSet[strings.ToUpper(id)] = true
		}

		for _, rate := range rates {
			slog.Debug("getRatesForCarriers: Checking rate for filtering",
				"carrier_code", rate.CarrierCode,
				"carrier_id", rate.CarrierID,
				"carrier_nickname", rate.CarrierNickname,
				"requested_carriers", carrierIDs)

			if carrierSet[rate.CarrierCode] || carrierSet[rate.CarrierID] ||
			   carrierSet[strings.ToUpper(rate.CarrierCode)] || carrierSet[strings.ToUpper(rate.CarrierID)] {
				filteredRates = append(filteredRates, rate)
				slog.Debug("getRatesForCarriers: Rate PASSED filter", "carrier", rate.CarrierCode)
			} else {
				slog.Debug("getRatesForCarriers: Rate FILTERED OUT", "carrier", rate.CarrierCode)
			}
		}
		rates = filteredRates
	}

	slog.Debug("getRatesForCarriers: Received rates",
		"rate_count", len(rates),
		"box_sku", boxSelection.Box.SKU)

	for i, rate := range rates {
		slog.Debug("getRatesForCarriers: Rate option",
			"index", i,
			"carrier", rate.CarrierNickname,
			"service", rate.ServiceType,
			"price", rate.ShippingAmount.Amount,
			"delivery_days", rate.DeliveryDays)
	}

	return rates, nil
}

func (s *ShippingService) getMockRates(boxSelection BoxSelection, shipTo Address) []Rate {
	// Generate realistic mock rates based on box size and weight
	basePrice := 5.0 + (boxSelection.Weight * 0.5) + (boxSelection.Box.L * boxSelection.Box.W * boxSelection.Box.H * 0.01)

	return []Rate{
		{
			RateID:          "mock-rate-usps-ground",
			CarrierID:       "mock-usps",
			CarrierCode:     "stamps_com",
			CarrierNickname: "USPS",
			ServiceCode:     "usps_ground_advantage",
			ServiceType:     "USPS Ground Advantage",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice},
			DeliveryDays:    5,
			EstimatedDate:   "",
		},
		{
			RateID:          "mock-rate-usps-priority",
			CarrierID:       "mock-usps",
			CarrierCode:     "stamps_com",
			CarrierNickname: "USPS",
			ServiceCode:     "usps_priority_mail",
			ServiceType:     "USPS Priority Mail",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice + 4.50},
			DeliveryDays:    2,
			EstimatedDate:   "",
		},
		{
			RateID:          "mock-rate-ups-ground",
			CarrierID:       "mock-ups",
			CarrierCode:     "ups",
			CarrierNickname: "UPS",
			ServiceCode:     "ups_ground",
			ServiceType:     "UPS Ground",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice + 2.50},
			DeliveryDays:    4,
			EstimatedDate:   "",
		},
		{
			RateID:          "mock-rate-ups-3day",
			CarrierID:       "mock-ups",
			CarrierCode:     "ups",
			CarrierNickname: "UPS",
			ServiceCode:     "ups_3_day_select",
			ServiceType:     "UPS 3 Day Select",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice + 6.00},
			DeliveryDays:    3,
			EstimatedDate:   "",
		},
		{
			RateID:          "mock-rate-fedex-ground",
			CarrierID:       "mock-fedex",
			CarrierCode:     "fedex",
			CarrierNickname: "FedEx",
			ServiceCode:     "fedex_ground",
			ServiceType:     "FedEx Ground",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice + 3.00},
			DeliveryDays:    3,
			EstimatedDate:   "",
		},
		{
			RateID:          "mock-rate-fedex-2day",
			CarrierID:       "mock-fedex",
			CarrierCode:     "fedex",
			CarrierNickname: "FedEx",
			ServiceCode:     "fedex_2day",
			ServiceType:     "FedEx 2Day",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice + 8.00},
			DeliveryDays:    2,
			EstimatedDate:   "",
		},
		{
			RateID:          "mock-rate-usps-express",
			CarrierID:       "mock-usps",
			CarrierCode:     "stamps_com",
			CarrierNickname: "USPS",
			ServiceCode:     "usps_priority_mail_express",
			ServiceType:     "USPS Priority Mail Express",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice + 12.00},
			DeliveryDays:    1,
			EstimatedDate:   "",
		},
		{
			RateID:          "mock-rate-fedex-overnight",
			CarrierID:       "mock-fedex",
			CarrierCode:     "fedex",
			CarrierNickname: "FedEx",
			ServiceCode:     "fedex_standard_overnight",
			ServiceType:     "FedEx Standard Overnight",
			ShippingAmount:  Amount{Currency: "usd", Amount: basePrice + 15.00},
			DeliveryDays:    1,
			EstimatedDate:   "",
		},
	}
}

func (s *ShippingService) addressFromConfig() Address {
	cf := s.config.Shipping.ShipFrom
	return Address{
		Name:                        cf.Name,
		Phone:                       cf.Phone,
		AddressLine1:                cf.AddressLine1,
		CityLocality:                cf.CityLocality,
		StateProvince:               cf.StateProvince,
		PostalCode:                  cf.PostalCode,
		CountryCode:                 cf.CountryCode,
		AddressResidentialIndicator: cf.AddressResidentialIndicator,
	}
}

// addressFromConfigUSPS returns the USPS ship-from address (Cadott, WI 54727)
func (s *ShippingService) addressFromConfigUSPS() Address {
	// Use USPS-specific address if configured, fallback to default
	cf := s.config.Shipping.ShipFromUSPS
	if cf.PostalCode == "" {
		cf = s.config.Shipping.ShipFrom
	}
	return Address{
		Name:                        cf.Name,
		Phone:                       cf.Phone,
		AddressLine1:                cf.AddressLine1,
		CityLocality:                cf.CityLocality,
		StateProvince:               cf.StateProvince,
		PostalCode:                  cf.PostalCode,
		CountryCode:                 cf.CountryCode,
		AddressResidentialIndicator: cf.AddressResidentialIndicator,
	}
}

// addressFromConfigOther returns the non-USPS ship-from address (Eau Claire, WI 54701)
func (s *ShippingService) addressFromConfigOther() Address {
	// Use non-USPS-specific address if configured, fallback to default
	cf := s.config.Shipping.ShipFromOther
	if cf.PostalCode == "" {
		cf = s.config.Shipping.ShipFrom
	}
	return Address{
		Name:                        cf.Name,
		Phone:                       cf.Phone,
		AddressLine1:                cf.AddressLine1,
		CityLocality:                cf.CityLocality,
		StateProvince:               cf.StateProvince,
		PostalCode:                  cf.PostalCode,
		CountryCode:                 cf.CountryCode,
		AddressResidentialIndicator: cf.AddressResidentialIndicator,
	}
}

func (s *ShippingService) sortShippingOptions(options []ShippingOption) []ShippingOption {
	sorted := make([]ShippingOption, len(options))
	copy(sorted, options)

	sortBy := s.config.Shipping.RatePreferences.Sort

	switch sortBy {
	case "price_then_days":
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].TotalCost == sorted[j].TotalCost {
				return sorted[i].DeliveryDays < sorted[j].DeliveryDays
			}
			return sorted[i].TotalCost < sorted[j].TotalCost
		})
	case "days_then_price":
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].DeliveryDays == sorted[j].DeliveryDays {
				return sorted[i].TotalCost < sorted[j].TotalCost
			}
			return sorted[i].DeliveryDays < sorted[j].DeliveryDays
		})
	default:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].TotalCost < sorted[j].TotalCost
		})
	}

	return sorted
}

func (s *ShippingService) CreateLabel(rateID string) (*Label, error) {
	// NOTE: EasyPost requires both shipment ID and rate ID
	// For now, we'll try to use the client method, but this may need refactoring
	// to store and pass shipment IDs through the checkout flow
	label, err := s.client.CreateLabel(rateID)
	if err != nil {
		return nil, fmt.Errorf("failed to create label: %w", err)
	}

	return label, nil
}

// CreateLabelFromShipment creates a label using shipment ID and rate ID
func (s *ShippingService) CreateLabelFromShipment(shipmentID, rateID string) (*Label, error) {
	label, err := s.client.BuyShipment(shipmentID, rateID)
	if err != nil {
		return nil, fmt.Errorf("failed to buy shipment: %w", err)
	}

	return label, nil
}

func (s *ShippingService) VoidLabel(shipmentID string) error {
	voidResp, err := s.client.VoidLabel(shipmentID)
	if err != nil {
		return fmt.Errorf("failed to void label: %w", err)
	}

	if !voidResp.Approved {
		return fmt.Errorf("label void not approved: %s", voidResp.Message)
	}

	return nil
}

func (s *ShippingService) DownloadLabelPDF(label *Label) ([]byte, error) {
	return s.client.DownloadLabelPDF(label)
}

func (s *ShippingService) RefreshCarriers() error {
	return s.loadCarrierIDs()
}

func (s *ShippingService) ValidateAddress(addr Address) error {
	// EasyPost validates addresses during rate retrieval
	// We'll do a simple rate check with minimal package to validate
	pkg := Package{
		PackageCode: "package",
		Weight:      Weight{Value: 1, Unit: "ounce"},
		Dimensions:  Dimensions{Length: 1, Width: 1, Height: 1, Unit: "inch"},
	}

	_, err := s.client.GetRates(s.addressFromConfig(), addr, pkg)
	return err
}

// UpdateConfig updates the shipping service configuration and recreates the packer
func (s *ShippingService) UpdateConfig(config *ShippingConfig) {
	slog.Info("ShippingService: Configuration reloaded",
		"present_top_n", config.Shipping.RatePreferences.PresentTopN,
		"sort_order", config.Shipping.RatePreferences.Sort,
		"num_boxes", len(config.Boxes))
	s.config = config
	s.packer = NewPacker(config)
}