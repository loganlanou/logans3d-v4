package shipping

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/loganlanou/logans3d-v4/storage/db"
)

type ShippingOption struct {
	RateID          string           `json:"rate_id"`
	ShipmentID      string           `json:"shipment_id"`                // EasyPost shipment ID for label purchase
	AllRateIDs      []string         `json:"all_rate_ids,omitempty"`     // All rate IDs for multi-box orders
	AllShipmentIDs  []string         `json:"all_shipment_ids,omitempty"` // All shipment IDs for multi-box orders
	CarrierName     string           `json:"carrier_name"`
	ServiceName     string           `json:"service_name"`
	Price           float64          `json:"price"`
	Currency        string           `json:"currency"`
	DeliveryDays    int              `json:"delivery_days"`
	EstimatedDate   string           `json:"estimated_date,omitempty"`
	BoxSKU          string           `json:"box_sku"`
	BoxCost         float64          `json:"box_cost"`
	HandlingCost    float64          `json:"handling_cost"`
	TotalCost       float64          `json:"total_cost"`
	PackingSolution *PackingSolution `json:"packing_solution,omitempty"`
}

type ShippingQuoteRequest struct {
	ItemCounts ItemCounts `json:"item_counts"`
	ShipTo     Address    `json:"ship_to"`
}

type ShippingQuoteResponse struct {
	Options       []ShippingOption `json:"options"`
	DefaultOption *ShippingOption  `json:"default_option,omitempty"`
	Error         string           `json:"error,omitempty"`
}

type ShippingService struct {
	config                     *ShippingConfig
	client                     *EasyPostClient
	packer                     *Packer
	carrierIDs                 []string
	carrierMap                 map[string]Carrier // Maps carrier ID to carrier info
	carrierAccountsByCadott    []string           // USPS carrier account IDs for Cadott (54727)
	carrierAccountsByEauClaire []string           // UPS/FedEx carrier account IDs for Eau Claire (54701)
}

func NewShippingService(config *ShippingConfig, queries *db.Queries) (*ShippingService, error) {
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
			// Load mock carrier accounts for local development
			service.carrierAccountsByCadott = []string{"ca_mock_usps"}
			service.carrierAccountsByEauClaire = []string{"ca_mock_ups", "ca_mock_fedex"}
			return service, nil
		}
		return nil, fmt.Errorf("failed to load carrier IDs: %w", err)
	}

	// Load carrier accounts from database
	if err := service.loadCarrierAccounts(queries); err != nil {
		return nil, fmt.Errorf("failed to load carrier accounts: %w", err)
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

func (s *ShippingService) loadCarrierAccounts(queries *db.Queries) error {
	ctx := context.Background()

	// Load carrier accounts for Cadott, WI (54727) - USPS
	cadottAccounts, err := queries.GetCarrierAccountsByLocation(ctx, "54727")
	if err != nil {
		return fmt.Errorf("failed to get Cadott carrier accounts: %w", err)
	}
	s.carrierAccountsByCadott = make([]string, 0, len(cadottAccounts))
	for _, account := range cadottAccounts {
		s.carrierAccountsByCadott = append(s.carrierAccountsByCadott, account.EasypostID)
	}
	slog.Debug("loaded Cadott carrier accounts", "count", len(s.carrierAccountsByCadott), "accounts", s.carrierAccountsByCadott)

	// Load carrier accounts for Eau Claire, WI (54701) - UPS/FedEx
	eauClaireAccounts, err := queries.GetCarrierAccountsByLocation(ctx, "54701")
	if err != nil {
		return fmt.Errorf("failed to get Eau Claire carrier accounts: %w", err)
	}
	s.carrierAccountsByEauClaire = make([]string, 0, len(eauClaireAccounts))
	for _, account := range eauClaireAccounts {
		s.carrierAccountsByEauClaire = append(s.carrierAccountsByEauClaire, account.EasypostID)
	}
	slog.Debug("loaded Eau Claire carrier accounts", "count", len(s.carrierAccountsByEauClaire), "accounts", s.carrierAccountsByEauClaire)

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

	// Get rates for each box
	var boxRates []BoxRatesResult
	for boxIdx, boxSelection := range packingSolution.Boxes {
		rates, err := s.getRatesForBox(boxSelection, req.ShipTo)
		if err != nil {
			slog.Debug("GetShippingQuote: Failed to get rates for box",
				"box_index", boxIdx,
				"box_sku", boxSelection.Box.SKU,
				"error", err)
			continue
		}
		boxRates = append(boxRates, BoxRatesResult{
			BoxSelection: boxSelection,
			Rates:        rates,
		})
	}

	// Aggregate rates across all boxes using extracted pure function
	sortedOptions := AggregateRates(boxRates, packingSolution, s.config.Shipping.RatePreferences.Sort)

	if len(sortedOptions) == 0 {
		return &ShippingQuoteResponse{
			Error: "No shipping options available",
		}, nil
	}

	slog.Debug("GetShippingQuote: Rate preferences",
		"sort_order", s.config.Shipping.RatePreferences.Sort,
		"total_options_available", len(sortedOptions))

	response := &ShippingQuoteResponse{
		Options: sortedOptions,
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

	// Get rates for USPS from Cadott, WI (54727) using only USPS carrier accounts
	if len(s.carrierAccountsByCadott) > 0 {
		uspsRates, err := s.getRatesForCarriers(s.carrierAccountsByCadott, boxSelection, shipTo, s.addressFromConfigUSPS())
		if err == nil {
			allRates = append(allRates, uspsRates...)
		}
	}

	// Get rates for UPS/FedEx from Eau Claire, WI (54701) using only UPS/FedEx carrier accounts
	if len(s.carrierAccountsByEauClaire) > 0 {
		otherRates, err := s.getRatesForCarriers(s.carrierAccountsByEauClaire, boxSelection, shipTo, s.addressFromConfigOther())
		if err == nil {
			allRates = append(allRates, otherRates...)
		}
	}

	if len(allRates) == 0 {
		return nil, fmt.Errorf("no rates available from any carrier")
	}

	return allRates, nil
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

	// Get rates from EasyPost with specific carrier accounts
	rates, err := s.client.GetRates(shipFrom, shipTo, pkg, carrierIDs)
	if err != nil {
		slog.Debug("getRatesForCarriers: Failed to get rates", "error", err)
		return nil, fmt.Errorf("failed to get rates: %w", err)
	}

	slog.Debug("getRatesForCarriers: Received rates from EasyPost",
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
	return SortShippingOptions(options, s.config.Shipping.RatePreferences.Sort)
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

// CreateLabelsForMultiBox creates labels for all shipments in a multi-box order.
// Returns all labels created, or an error if any label purchase fails.
// If a failure occurs partway through, already-purchased labels are still returned
// so they can be voided if needed.
func (s *ShippingService) CreateLabelsForMultiBox(shipmentIDs, rateIDs []string) ([]*Label, error) {
	if len(shipmentIDs) != len(rateIDs) {
		return nil, fmt.Errorf("shipment IDs and rate IDs must have same length: got %d and %d",
			len(shipmentIDs), len(rateIDs))
	}

	if len(shipmentIDs) == 0 {
		return nil, fmt.Errorf("no shipments provided")
	}

	var labels []*Label
	for i := range shipmentIDs {
		label, err := s.client.BuyShipment(shipmentIDs[i], rateIDs[i])
		if err != nil {
			slog.Error("failed to buy shipment in multi-box order",
				"error", err,
				"shipment_id", shipmentIDs[i],
				"rate_id", rateIDs[i],
				"box_index", i,
				"total_boxes", len(shipmentIDs),
				"labels_purchased_so_far", len(labels))
			return labels, fmt.Errorf("failed to buy shipment %d of %d: %w", i+1, len(shipmentIDs), err)
		}
		labels = append(labels, label)
	}

	slog.Info("created labels for multi-box order",
		"total_labels", len(labels),
		"shipment_ids", shipmentIDs)

	return labels, nil
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

func (s *ShippingService) IsUsingMockData() bool {
	return s.client.IsUsingMockData()
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

	// Use empty carrier accounts list - we just want to validate the address
	_, err := s.client.GetRates(s.addressFromConfig(), addr, pkg, []string{})
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

// GetShipmentTracking retrieves tracking info for a shipment from EasyPost
func (s *ShippingService) GetShipmentTracking(shipmentID string) (*ShipmentTracking, error) {
	return s.client.GetShipmentTracking(shipmentID)
}

// RefreshShipmentRates gets updated rates for an existing EasyPost shipment
func (s *ShippingService) RefreshShipmentRates(shipmentID string) ([]Rate, error) {
	return s.client.RefreshShipmentRates(shipmentID)
}

// GetDefaultItemWeights returns the configured default weights per category (in oz)
func (s *ShippingService) GetDefaultItemWeights() map[string]float64 {
	weights := make(map[string]float64)
	for category, iw := range s.config.Packing.ItemWeights {
		weights[category] = iw.AvgOz
	}
	return weights
}

// GetDefaultDimensions returns the configured default dimensions per category
func (s *ShippingService) GetDefaultDimensions() map[string]DimensionGuard {
	return s.config.Packing.DimensionGuard
}
