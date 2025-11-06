package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	easyPostBaseURL = "https://api.easypost.com/v2"
)

type CarrierAccount struct {
	ID          string                 `json:"id"`
	Object      string                 `json:"object"`
	Type        string                 `json:"type"`
	Readable    string                 `json:"readable"`
	Credentials map[string]interface{} `json:"credentials"`
	TestData    struct {
		Enabled bool `json:"enabled"`
	} `json:"test_credentials"`
}

type CarrierAccountsResponse struct {
	CarrierAccounts []CarrierAccount `json:"carrier_accounts"`
}

type Address struct {
	Name    string `json:"name,omitempty"`
	Company string `json:"company,omitempty"`
	Street1 string `json:"street1"`
	Street2 string `json:"street2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
	Phone   string `json:"phone,omitempty"`
	Email   string `json:"email,omitempty"`
}

type Parcel struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Weight float64 `json:"weight"`
}

type ShipmentRequest struct {
	Shipment struct {
		ToAddress   Address `json:"to_address"`
		FromAddress Address `json:"from_address"`
		Parcel      Parcel  `json:"parcel"`
	} `json:"shipment"`
}

type Rate struct {
	ID               string  `json:"id"`
	Carrier          string  `json:"carrier"`
	Service          string  `json:"service"`
	Rate             string  `json:"rate"`
	DeliveryDays     *int    `json:"delivery_days"`
	DeliveryDate     *string `json:"delivery_date"`
	CarrierAccountID string  `json:"carrier_account_id"`
}

type ShipmentResponse struct {
	ID       string `json:"id"`
	Rates    []Rate `json:"rates"`
	Messages []struct {
		Carrier string `json:"carrier"`
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"messages"`
}

func main() {
	apiKey := os.Getenv("EASYPOST_API_KEY")
	if apiKey == "" {
		log.Fatal("EASYPOST_API_KEY environment variable is not set")
	}

	fmt.Println("=== EasyPost Carrier Diagnostics ===")
	fmt.Printf("API Key: %s...%s\n", apiKey[:10], apiKey[len(apiKey)-10:])
	keyType := getKeyType(apiKey)
	fmt.Printf("Key Type: %s\n\n", keyType)

	var carriers *CarrierAccountsResponse

	// Step 1: List carrier accounts (only works with production keys)
	if strings.HasPrefix(apiKey, "EZAK") {
		fmt.Println("📋 Step 1: Fetching configured carrier accounts...")
		var err error
		carriers, err = listCarrierAccounts(apiKey)
		if err != nil {
			log.Fatalf("Failed to list carrier accounts: %v", err)
		}

		fmt.Printf("✅ Found %d carrier account(s):\n\n", len(carriers.CarrierAccounts))
		for i, carrier := range carriers.CarrierAccounts {
			fmt.Printf("%d. %s\n", i+1, carrier.Readable)
			fmt.Printf("   ID: %s\n", carrier.ID)
			fmt.Printf("   Type: %s\n", carrier.Type)
			fmt.Printf("   Test Mode: %v\n", carrier.TestData.Enabled)
			fmt.Println()
		}
	} else {
		fmt.Println("⚠️  Step 1: Skipping carrier account listing (requires production key)")
		fmt.Println("   Using test key - will only show carriers that return rates")
	}

	// Step 2: Create test shipment and get rates
	fmt.Println("\n📦 Step 2: Creating test shipment and requesting rates...")
	shipment, err := createTestShipment(apiKey)
	if err != nil {
		log.Fatalf("Failed to create test shipment: %v", err)
	}

	fmt.Printf("✅ Shipment created: %s\n\n", shipment.ID)

	if len(shipment.Messages) > 0 {
		fmt.Println("⚠️  Carrier Messages:")
		for _, msg := range shipment.Messages {
			fmt.Printf("   [%s] %s: %s\n", msg.Type, msg.Carrier, msg.Message)
		}
		fmt.Println()
	}

	// Step 3: Show rates by carrier
	fmt.Printf("📊 Step 3: Analyzing rates (%d total):\n\n", len(shipment.Rates))

	carrierGroups := make(map[string][]Rate)
	for _, rate := range shipment.Rates {
		carrierGroups[rate.Carrier] = append(carrierGroups[rate.Carrier], rate)
	}

	for carrier, rates := range carrierGroups {
		fmt.Printf("🚚 %s (%d rate(s)):\n", strings.ToUpper(carrier), len(rates))
		for _, rate := range rates {
			deliveryInfo := "N/A"
			if rate.DeliveryDays != nil {
				deliveryInfo = fmt.Sprintf("%d days", *rate.DeliveryDays)
			}
			if rate.DeliveryDate != nil {
				deliveryInfo += fmt.Sprintf(" (by %s)", *rate.DeliveryDate)
			}
			fmt.Printf("   • %s: $%s - %s\n", rate.Service, rate.Rate, deliveryInfo)
			fmt.Printf("     Carrier Account: %s\n", rate.CarrierAccountID)
		}
		fmt.Println()
	}

	// Step 4: Summary
	fmt.Println("=== Summary ===")
	if carriers != nil {
		fmt.Printf("Configured Carriers: %d\n", len(carriers.CarrierAccounts))
	} else {
		fmt.Println("Configured Carriers: N/A (use production key to list)")
	}
	fmt.Printf("Carriers Returning Rates: %d\n", len(carrierGroups))
	fmt.Printf("Total Rates: %d\n", len(shipment.Rates))

	if carriers != nil && len(carrierGroups) < len(carriers.CarrierAccounts) {
		fmt.Println("\n⚠️  WARNING: Some configured carriers did not return rates!")
		fmt.Println("This could be due to:")
		fmt.Println("  - Carrier account not fully activated")
		fmt.Println("  - Carrier doesn't service this route")
		fmt.Println("  - Package dimensions/weight outside carrier limits")
		fmt.Println("  - Wallet funding required (for production keys)")
	}

	if len(carrierGroups) == 0 {
		fmt.Println("\n❌ ERROR: No carriers returned rates!")
		fmt.Println("Possible reasons:")
		fmt.Println("  - No carrier accounts configured in EasyPost dashboard")
		fmt.Println("  - API key doesn't have access to carrier accounts")
		fmt.Println("  - All carriers rejected the shipment parameters")
	}
}

func getKeyType(key string) string {
	if strings.HasPrefix(key, "EZTK") {
		return "Test Key"
	} else if strings.HasPrefix(key, "EZAK") {
		return "Production Key"
	}
	return "Unknown"
}

func listCarrierAccounts(apiKey string) (*CarrierAccountsResponse, error) {
	req, err := http.NewRequest("GET", easyPostBaseURL+"/carrier_accounts", nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(apiKey, "")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result CarrierAccountsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w\nBody: %s", err, string(body))
	}

	return &result, nil
}

func createTestShipment(apiKey string) (*ShipmentResponse, error) {
	shipmentReq := ShipmentRequest{}
	shipmentReq.Shipment.ToAddress = Address{
		Name:    "Test Customer",
		Street1: "388 Townsend St",
		City:    "San Francisco",
		State:   "CA",
		Zip:     "94107",
		Country: "US",
		Phone:   "415-123-4567",
	}
	shipmentReq.Shipment.FromAddress = Address{
		Company: "Logan's 3D Creations",
		Street1: "417 Montgomery St",
		Street2: "Floor 5",
		City:    "San Francisco",
		State:   "CA",
		Zip:     "94104",
		Country: "US",
		Phone:   "715-703-3768",
	}
	shipmentReq.Shipment.Parcel = Parcel{
		Length: 10.0,
		Width:  7.0,
		Height: 4.0,
		Weight: 15.0, // 15 oz
	}

	jsonData, err := json.Marshal(shipmentReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", easyPostBaseURL+"/shipments", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(apiKey, "")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var shipment ShipmentResponse
	if err := json.Unmarshal(body, &shipment); err != nil {
		return nil, fmt.Errorf("failed to parse shipment response: %w\nBody: %s", err, string(body))
	}

	return &shipment, nil
}
