package recaptcha

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type VerifyResponse struct {
	Success     bool     `json:"success"`
	Score       float64  `json:"score"`
	Action      string   `json:"action"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

func Verify(token string) (*VerifyResponse, error) {
	secretKey := os.Getenv("RECAPTCHA_SECRET_KEY")
	if secretKey == "" {
		return nil, fmt.Errorf("RECAPTCHA_SECRET_KEY not set")
	}

	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   {secretKey},
		"response": {token},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func IsValid(token string) (bool, float64, error) {
	result, err := Verify(token)
	if err != nil {
		return false, 0, err
	}

	if !result.Success {
		return false, result.Score, fmt.Errorf("recaptcha verification failed: %v", result.ErrorCodes)
	}

	minScoreStr := os.Getenv("RECAPTCHA_MIN_SCORE")
	minScore := 0.5
	if minScoreStr != "" {
		if parsed, err := strconv.ParseFloat(minScoreStr, 64); err == nil {
			minScore = parsed
		}
	}

	return result.Score >= minScore, result.Score, nil
}
