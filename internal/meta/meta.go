package meta

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	apiVersion = "v21.0"
	apiBaseURL = "https://graph.facebook.com"
)

type Client struct {
	pixelID     string
	accessToken string
	httpClient  *http.Client
}

func NewClient() *Client {
	return &Client{
		pixelID:     os.Getenv("META_PIXEL_ID"),
		accessToken: os.Getenv("META_ACCESS_TOKEN"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) IsConfigured() bool {
	return c.pixelID != "" && c.accessToken != ""
}

type EventRequest struct {
	Data []Event `json:"data"`
}

type Event struct {
	EventName      string      `json:"event_name"`
	EventTime      int64       `json:"event_time"`
	EventID        string      `json:"event_id,omitempty"`
	EventSourceURL string      `json:"event_source_url,omitempty"`
	ActionSource   string      `json:"action_source"`
	UserData       UserData    `json:"user_data"`
	CustomData     *CustomData `json:"custom_data,omitempty"`
}

type UserData struct {
	Email           string `json:"em,omitempty"`
	Phone           string `json:"ph,omitempty"`
	FirstName       string `json:"fn,omitempty"`
	LastName        string `json:"ln,omitempty"`
	City            string `json:"ct,omitempty"`
	State           string `json:"st,omitempty"`
	Zip             string `json:"zp,omitempty"`
	Country         string `json:"country,omitempty"`
	ExternalID      string `json:"external_id,omitempty"`
	ClientIPAddress string `json:"client_ip_address,omitempty"`
	ClientUserAgent string `json:"client_user_agent,omitempty"`
	FBC             string `json:"fbc,omitempty"`
	FBP             string `json:"fbp,omitempty"`
}

type CustomData struct {
	Value           float64       `json:"value,omitempty"`
	Currency        string        `json:"currency,omitempty"`
	ContentName     string        `json:"content_name,omitempty"`
	ContentCategory string        `json:"content_category,omitempty"`
	ContentIDs      []string      `json:"content_ids,omitempty"`
	Contents        []ContentItem `json:"contents,omitempty"`
	ContentType     string        `json:"content_type,omitempty"`
	NumItems        int           `json:"num_items,omitempty"`
	OrderID         string        `json:"order_id,omitempty"`
	Status          string        `json:"status,omitempty"`
}

type ContentItem struct {
	ID       string  `json:"id"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"item_price,omitempty"`
}

type APIResponse struct {
	EventsReceived int      `json:"events_received"`
	Messages       []string `json:"messages,omitempty"`
	FBTraceID      string   `json:"fbtrace_id,omitempty"`
}

func hashValue(value string) string {
	if value == "" {
		return ""
	}
	value = strings.ToLower(strings.TrimSpace(value))
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func (c *Client) SendEvent(event Event) error {
	if !c.IsConfigured() {
		slog.Debug("meta pixel not configured, skipping event", "event_name", event.EventName)
		return nil
	}

	event.ActionSource = "website"
	if event.EventTime == 0 {
		event.EventTime = time.Now().Unix()
	}

	event.UserData.Email = hashValue(event.UserData.Email)
	event.UserData.Phone = hashValue(event.UserData.Phone)
	event.UserData.FirstName = hashValue(event.UserData.FirstName)
	event.UserData.LastName = hashValue(event.UserData.LastName)
	event.UserData.City = hashValue(event.UserData.City)
	event.UserData.State = hashValue(event.UserData.State)
	event.UserData.Zip = hashValue(event.UserData.Zip)
	event.UserData.Country = hashValue(event.UserData.Country)

	payload := EventRequest{
		Data: []Event{event},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal meta event", "error", err)
		return fmt.Errorf("marshal event: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s/events?access_token=%s", apiBaseURL, apiVersion, c.pixelID, c.accessToken)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("failed to create meta request", "error", err)
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("failed to send meta event", "error", err, "event_name", event.EventName)
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		slog.Error("failed to decode meta response", "error", err)
		return fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("meta api error", "status", resp.StatusCode, "messages", apiResp.Messages)
		return fmt.Errorf("api error: status %d", resp.StatusCode)
	}

	slog.Debug("meta event sent successfully",
		"event_name", event.EventName,
		"events_received", apiResp.EventsReceived,
		"trace_id", apiResp.FBTraceID,
	)

	return nil
}

func (c *Client) SendEventAsync(event Event) {
	go func() {
		if err := c.SendEvent(event); err != nil {
			slog.Error("async meta event failed", "error", err, "event_name", event.EventName)
		}
	}()
}

func (c *Client) TrackPurchase(orderID string, totalValue float64, currency string, email string, items []ContentItem, ipAddress, userAgent, sourceURL string) {
	event := Event{
		EventName:      "Purchase",
		EventID:        fmt.Sprintf("purchase_%s", orderID),
		EventSourceURL: sourceURL,
		UserData: UserData{
			Email:           email,
			ClientIPAddress: ipAddress,
			ClientUserAgent: userAgent,
		},
		CustomData: &CustomData{
			Value:       totalValue,
			Currency:    currency,
			ContentType: "product",
			Contents:    items,
			NumItems:    len(items),
			OrderID:     orderID,
		},
	}
	c.SendEventAsync(event)
}

func (c *Client) TrackAddToCart(productID string, productName string, value float64, currency string, email, ipAddress, userAgent, sourceURL string) {
	event := Event{
		EventName:      "AddToCart",
		EventID:        fmt.Sprintf("atc_%s_%d", productID, time.Now().UnixNano()),
		EventSourceURL: sourceURL,
		UserData: UserData{
			Email:           email,
			ClientIPAddress: ipAddress,
			ClientUserAgent: userAgent,
		},
		CustomData: &CustomData{
			Value:       value,
			Currency:    currency,
			ContentName: productName,
			ContentIDs:  []string{productID},
			ContentType: "product",
		},
	}
	c.SendEventAsync(event)
}

func (c *Client) TrackInitiateCheckout(value float64, currency string, numItems int, email, ipAddress, userAgent, sourceURL string) {
	event := Event{
		EventName:      "InitiateCheckout",
		EventID:        fmt.Sprintf("ic_%d", time.Now().UnixNano()),
		EventSourceURL: sourceURL,
		UserData: UserData{
			Email:           email,
			ClientIPAddress: ipAddress,
			ClientUserAgent: userAgent,
		},
		CustomData: &CustomData{
			Value:       value,
			Currency:    currency,
			NumItems:    numItems,
			ContentType: "product",
		},
	}
	c.SendEventAsync(event)
}

func (c *Client) TrackContact(email, firstName, lastName, subject, ipAddress, userAgent, sourceURL string) {
	event := Event{
		EventName:      "Contact",
		EventID:        fmt.Sprintf("contact_%d", time.Now().UnixNano()),
		EventSourceURL: sourceURL,
		UserData: UserData{
			Email:           email,
			FirstName:       firstName,
			LastName:        lastName,
			ClientIPAddress: ipAddress,
			ClientUserAgent: userAgent,
		},
		CustomData: &CustomData{
			ContentName: subject,
		},
	}
	c.SendEventAsync(event)
}

func (c *Client) TrackLead(email, firstName, contentName, ipAddress, userAgent, sourceURL string) {
	event := Event{
		EventName:      "Lead",
		EventID:        fmt.Sprintf("lead_%d", time.Now().UnixNano()),
		EventSourceURL: sourceURL,
		UserData: UserData{
			Email:           email,
			FirstName:       firstName,
			ClientIPAddress: ipAddress,
			ClientUserAgent: userAgent,
		},
		CustomData: &CustomData{
			ContentName: contentName,
		},
	}
	c.SendEventAsync(event)
}

func (c *Client) TrackCompleteRegistration(email, firstName, lastName, ipAddress, userAgent, sourceURL string) {
	event := Event{
		EventName:      "CompleteRegistration",
		EventID:        fmt.Sprintf("reg_%d", time.Now().UnixNano()),
		EventSourceURL: sourceURL,
		UserData: UserData{
			Email:           email,
			FirstName:       firstName,
			LastName:        lastName,
			ClientIPAddress: ipAddress,
			ClientUserAgent: userAgent,
		},
	}
	c.SendEventAsync(event)
}

func (c *Client) TrackViewContent(productID, productName, category string, value float64, currency, ipAddress, userAgent, sourceURL string) {
	event := Event{
		EventName:      "ViewContent",
		EventID:        fmt.Sprintf("vc_%s_%d", productID, time.Now().UnixNano()),
		EventSourceURL: sourceURL,
		UserData: UserData{
			ClientIPAddress: ipAddress,
			ClientUserAgent: userAgent,
		},
		CustomData: &CustomData{
			Value:           value,
			Currency:        currency,
			ContentName:     productName,
			ContentCategory: category,
			ContentIDs:      []string{productID},
			ContentType:     "product",
		},
	}
	c.SendEventAsync(event)
}
