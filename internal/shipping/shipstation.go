package shipping

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const ShipStationBaseURL = "https://api.shipstation.com"

type ShipStationClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

type Address struct {
	Name                        string `json:"name,omitempty"`
	Phone                       string `json:"phone,omitempty"`
	AddressLine1                string `json:"address_line1,omitempty"`
	AddressLine2                string `json:"address_line2,omitempty"`
	CityLocality                string `json:"city_locality,omitempty"`
	StateProvince               string `json:"state_province,omitempty"`
	PostalCode                  string `json:"postal_code,omitempty"`
	CountryCode                 string `json:"country_code,omitempty"`
	AddressResidentialIndicator string `json:"address_residential_indicator,omitempty"`
}

type Weight struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type Dimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"`
}

type Package struct {
	PackageCode string     `json:"package_code"`
	Weight      Weight     `json:"weight"`
	Dimensions  Dimensions `json:"dimensions"`
}

type Shipment struct {
	ValidateAddress string    `json:"validate_address"`
	ShipTo          Address   `json:"ship_to"`
	ShipFrom        Address   `json:"ship_from"`
	Packages        []Package `json:"packages"`
}

type RateOptions struct {
	CarrierIDs   []string `json:"carrier_ids,omitempty"`
	ServiceCodes []string `json:"service_codes,omitempty"`
}

type RateRequest struct {
	RateOptions RateOptions `json:"rate_options"`
	Shipment    Shipment    `json:"shipment"`
}

type Rate struct {
	RateID           string  `json:"rate_id"`
	ShipmentID       string  `json:"shipment_id,omitempty"` // EasyPost shipment ID for label purchase
	CarrierID        string  `json:"carrier_id"`
	CarrierCode      string  `json:"carrier_code"`
	CarrierNickname  string  `json:"carrier_nickname"`
	ServiceCode      string  `json:"service_code"`
	ServiceType      string  `json:"service_type"`
	ShippingAmount   Amount  `json:"shipping_amount"`
	DeliveryDays     int     `json:"delivery_days,omitempty"`
	EstimatedDate    string  `json:"estimated_delivery_date,omitempty"`
	ValidationStatus string  `json:"validation_status,omitempty"`
	WarningMessages  []string `json:"warning_messages,omitempty"`
	ErrorMessages    []string `json:"error_messages,omitempty"`
}

type Amount struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

type RateResponse struct {
	Rates          []Rate `json:"rates"`
	InvalidRates   []Rate `json:"invalid_rates,omitempty"`
	RateRequestID  string `json:"rate_request_id,omitempty"`
	Errors         []APIError `json:"errors,omitempty"`
}

type APIError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Type    string `json:"type,omitempty"`
}

type Carrier struct {
	CarrierID       string `json:"carrier_id"`
	CarrierCode     string `json:"carrier_code"`
	CarrierNickname string `json:"carrier_nickname"`
}

type CarriersResponse struct {
	Carriers []Carrier `json:"carriers"`
}

type LabelHrefs struct {
	PDF string `json:"pdf,omitempty"`
	PNG string `json:"png,omitempty"`
	ZPL string `json:"zpl,omitempty"`
}

type LabelDownload struct {
	Hrefs LabelHrefs `json:"href"`
}

type Label struct {
	LabelID         string        `json:"label_id"`
	TrackingNumber  string        `json:"tracking_number"`
	Status          string        `json:"status"`
	ShippingAmount  Amount        `json:"shipping_amount"`
	CarrierID       string        `json:"carrier_id"`
	ServiceCode     string        `json:"service_code"`
	LabelDownload   LabelDownload `json:"label_download"`
	CreatedAt       time.Time     `json:"created_at"`
}

type LabelResponse struct {
	Label  Label      `json:"label"`
	Errors []APIError `json:"errors,omitempty"`
}

type VoidLabelResponse struct {
	Approved bool   `json:"approved"`
	Message  string `json:"message,omitempty"`
}

func NewShipStationClient() *ShipStationClient {
	apiKey := os.Getenv("SHIPSTATION_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("SS_V2_API_KEY")
	}

	return &ShipStationClient{
		apiKey:  apiKey,
		baseURL: ShipStationBaseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ShipStationClient) IsUsingMockData() bool {
	return c.apiKey == ""
}

func (c *ShipStationClient) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func (c *ShipStationClient) GetCarriers() (*CarriersResponse, error) {
	resp, err := c.makeRequest("GET", "/v2/carriers", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, body)
	}

	var carriers CarriersResponse
	if err := json.NewDecoder(resp.Body).Decode(&carriers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &carriers, nil
}

func (c *ShipStationClient) GetRates(rateReq *RateRequest) (*RateResponse, error) {
	resp, err := c.makeRequest("POST", "/v2/rates", rateReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, body)
	}

	var rateResp RateResponse
	if err := json.Unmarshal(body, &rateResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &rateResp, nil
}

func (c *ShipStationClient) CreateLabel(rateID string) (*LabelResponse, error) {
	endpoint := fmt.Sprintf("/v2/labels/rates/%s", rateID)
	resp, err := c.makeRequest("POST", endpoint, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, body)
	}

	var labelResp LabelResponse
	if err := json.Unmarshal(body, &labelResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &labelResp, nil
}

func (c *ShipStationClient) VoidLabel(labelID string) (*VoidLabelResponse, error) {
	endpoint := fmt.Sprintf("/v2/labels/%s/void", labelID)
	resp, err := c.makeRequest("PUT", endpoint, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, body)
	}

	var voidResp VoidLabelResponse
	if err := json.Unmarshal(body, &voidResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &voidResp, nil
}

func (c *ShipStationClient) downloadLabel(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download label: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *ShipStationClient) DownloadLabelPDF(label *Label) ([]byte, error) {
	if label.LabelDownload.Hrefs.PDF == "" {
		return nil, fmt.Errorf("no PDF URL available for label")
	}
	return c.downloadLabel(label.LabelDownload.Hrefs.PDF)
}