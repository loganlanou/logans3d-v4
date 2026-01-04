package importer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// HTTPClient is a rate-limited HTTP client for scraping
type HTTPClient struct {
	client      *http.Client
	rateLimit   time.Duration
	lastRequest time.Time
	mu          sync.Mutex
	userAgents  []string
}

// NewHTTPClient creates a new rate-limited HTTP client
func NewHTTPClient(requestsPerMinute int) *HTTPClient {
	rateLimit := time.Minute / time.Duration(requestsPerMinute)

	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimit: rateLimit,
		userAgents: []string{
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		},
	}
}

// Get performs a rate-limited GET request
func (c *HTTPClient) Get(ctx context.Context, url string) ([]byte, error) {
	c.mu.Lock()

	// Wait for rate limit
	elapsed := time.Since(c.lastRequest)
	if elapsed < c.rateLimit {
		wait := c.rateLimit - elapsed
		c.mu.Unlock()

		slog.Debug("rate limiting", "wait", wait, "url", url)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		c.mu.Lock()
	}

	c.lastRequest = time.Now()
	c.mu.Unlock()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", c.randomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")

	// Execute request with retries
	var resp *http.Response
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if i < maxRetries-1 {
			backoff := time.Duration(1<<uint(i)) * time.Second
			slog.Debug("retrying request", "attempt", i+1, "backoff", backoff, "url", url)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after retries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}

func (c *HTTPClient) randomUserAgent() string {
	//nolint:gosec // math/rand is fine for user-agent rotation, not security-sensitive
	return c.userAgents[rand.Intn(len(c.userAgents))]
}
