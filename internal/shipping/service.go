package shipping

import (
	"fmt"
	"sort"
)

type ShippingOption struct {
	RateID          string  `json:"rate_id"`
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
	config     *ShippingConfig
	client     *ShipStationClient
	packer     *Packer
	carrierIDs []string
}

func NewShippingService(config *ShippingConfig) (*ShippingService, error) {
	client := NewShipStationClient()
	packer := NewPacker(config)

	service := &ShippingService{
		config: config,
		client: client,
		packer: packer,
	}

	if err := service.loadCarrierIDs(); err != nil {
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
	for _, carrier := range carriersResp.Carriers {
		s.carrierIDs = append(s.carrierIDs, carrier.CarrierID)
	}

	return nil
}

func (s *ShippingService) GetShippingQuote(req *ShippingQuoteRequest) (*ShippingQuoteResponse, error) {
	packingSolution := s.packer.Pack(req.ItemCounts)
	if !packingSolution.Valid {
		return &ShippingQuoteResponse{
			Error: fmt.Sprintf("Unable to pack items: %s", packingSolution.Error),
		}, nil
	}

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
	rateReq := &RateRequest{
		RateOptions: RateOptions{
			CarrierIDs: s.carrierIDs,
		},
		Shipment: Shipment{
			ValidateAddress: "no_validation",
			ShipFrom:        s.addressFromConfig(),
			ShipTo:          shipTo,
			Packages: []Package{
				{
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
				},
			},
		},
	}

	rateResp, err := s.client.GetRates(rateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get rates: %w", err)
	}

	return rateResp.Rates, nil
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
	labelResp, err := s.client.CreateLabel(rateID)
	if err != nil {
		return nil, fmt.Errorf("failed to create label: %w", err)
	}

	if len(labelResp.Errors) > 0 {
		return nil, fmt.Errorf("ShipStation API error: %s", labelResp.Errors[0].Message)
	}

	return &labelResp.Label, nil
}

func (s *ShippingService) VoidLabel(labelID string) error {
	voidResp, err := s.client.VoidLabel(labelID)
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
	rateReq := &RateRequest{
		Shipment: Shipment{
			ValidateAddress: "validate_only",
			ShipFrom:        s.addressFromConfig(),
			ShipTo:          addr,
			Packages: []Package{
				{
					PackageCode: "package",
					Weight:      Weight{Value: 1, Unit: "ounce"},
					Dimensions:  Dimensions{Length: 1, Width: 1, Height: 1, Unit: "inch"},
				},
			},
		},
	}

	_, err := s.client.GetRates(rateReq)
	return err
}