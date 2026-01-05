package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultOllamaURL = "http://localhost:11434"
	defaultModel     = "mistral:7b"
	defaultTimeout   = 120 * time.Second
)

type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Model              string `json:"model"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

type TagsResponse struct {
	Models []ModelInfo `json:"models"`
}

type ModelInfo struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

func NewClient() *Client {
	baseURL := os.Getenv("OLLAMA_URL")
	if baseURL == "" {
		baseURL = defaultOllamaURL
	}

	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = defaultModel
	}

	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) GetModel() string {
	return c.model
}

func (c *Client) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Debug("ollama not available", "error", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var tags TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return false
	}

	for _, m := range tags.Models {
		if m.Name == c.model || strings.HasPrefix(m.Name, c.model+":") {
			return true
		}
	}

	slog.Warn("ollama available but model not found", "model", c.model, "available_models", len(tags.Models))
	return false
}

func (c *Client) GenerateDescription(ctx context.Context, productName, originalDescription string) (string, error) {
	if originalDescription == "" {
		return "", fmt.Errorf("no original description to convert")
	}

	prompt := buildDescriptionPrompt(productName, originalDescription)

	req := GenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	slog.Debug("calling ollama for description generation",
		"model", c.model,
		"product", productName,
		"original_length", len(originalDescription),
		"prompt", prompt,
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if genResp.Response == "" {
		return "", fmt.Errorf("empty response from model")
	}

	generated := strings.TrimSpace(genResp.Response)

	slog.Debug("ollama description generated",
		"model", genResp.Model,
		"product", productName,
		"generated_length", len(generated),
		"eval_count", genResp.EvalCount,
		"response", generated,
	)

	return generated, nil
}

func buildDescriptionPrompt(name, originalDesc string) string {
	// Pre-filter both name and description to remove printing-related content
	cleanedName := FilterPrintingJunkFromName(name)
	cleanedDesc := filterPrintingJunk(originalDesc)

	slog.Debug("prompt after filtering",
		"original_name", name,
		"cleaned_name", cleanedName,
		"original_desc_len", len(originalDesc),
		"cleaned_desc_len", len(cleanedDesc),
		"cleaned_desc", cleanedDesc,
	)

	return fmt.Sprintf(`Product: %s

Source notes: %s

Write a 2-paragraph product description for a gift shop. Keep it simple and honest.

RULES:
- Only describe what's in the source notes
- Do NOT invent sizes, textures, or features
- Do NOT say "soft", "plush", "flexible material", or "any position"
- Do NOT mention materials, plastic, how it was made
- If it has articulated joints, just say "features moving joints"
- Keep it short and casual`, cleanedName, cleanedDesc)
}

// FilterPrintingJunkFromName removes printing jargon from product names (exported for use by handlers)
func FilterPrintingJunkFromName(name string) string {
	// Phrases to remove from product names
	junkPhrases := []string{
		"Now with 3MF Files Included",
		"with 3MF Files Included",
		"3MF Files Included",
		"Now with 3MF",
		"with 3MF",
		"3MF Included",
		"STL Files Included",
		"with STL",
		"Print-in-Place",
		"Print in Place",
		"Articulated Print in Place",
		"- 3MF",
		"(3MF)",
		"[3MF]",
	}

	result := name
	for _, phrase := range junkPhrases {
		// Case-insensitive replacement
		lower := strings.ToLower(result)
		lowerPhrase := strings.ToLower(phrase)
		if idx := strings.Index(lower, lowerPhrase); idx != -1 {
			result = result[:idx] + result[idx+len(phrase):]
		}
	}

	// Clean up extra spaces and trailing punctuation
	result = strings.TrimSpace(result)
	result = strings.TrimRight(result, " -–—")
	result = strings.ReplaceAll(result, "  ", " ")

	return result
}

// filterPrintingJunk removes sentences containing 3D printing terminology
// that's irrelevant to customers buying finished products
func filterPrintingJunk(text string) string {
	// Words that indicate a sentence is about printing, not the product
	junkWords := []string{
		// File formats
		"stl", "3mf", "obj", "gcode",
		// Printers
		"prusa", "bambu", "ender", "creality",
		// Print settings
		"print-in-place", "print in place", "printed", "printing",
		"no supports", "no support", "supports needed", "support free",
		"infill", "layer height", "nozzle",
		"filament", "pla", "petg", "abs",
		"slicer", "slicing",
		// Licensing/legal
		"license", "licensed", "licensing",
		"copyright", "commercial use", "permission",
		"patreon", "patron",
		// Downloads/files
		"download", "file", "files", "project",
		"version as a", "starting point",
		"coloring", "colored version",
		// Websites/sharing/promo
		"cults3d", "thingiverse", "printables", "myminifactory",
		".com", ".org", ".net", "https://", "http://", "www.",
		"share your", "tag us", "tag me", "show us",
		"we'd love to see", "love to see",
		"check out my", "check out our", "other models",
		"follow me", "follow us",
		// Designer credits (customers don't care)
		"designed by", "created by", "modeled by", "i created",
		"flexifactory", "flexi factory",
		"- dan", "-dan", "thank you", "thanks",
		"please note", "note:",
		"p.s.", "p. s.", "ps.", "ps:",
		"in my opinion", "imo",
		// Other printing jargon
		"print settings", "print time", "print speed",
		"layer", "layers",
	}

	// Split into sentences and filter
	sentences := strings.Split(text, ".")
	var kept []string

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		lower := strings.ToLower(sentence)
		hasJunk := false
		for _, junk := range junkWords {
			if strings.Contains(lower, junk) {
				hasJunk = true
				break
			}
		}

		if !hasJunk {
			kept = append(kept, sentence)
		}
	}

	result := strings.Join(kept, ". ")
	if len(kept) > 0 && !strings.HasSuffix(result, ".") {
		result += "."
	}

	return result
}
