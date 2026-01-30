package classifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the Python classification service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// ClassifyRequest is the request body for classification
type ClassifyRequest struct {
	EmailSubject string  `json:"email_subject"`
	EmailBody    string  `json:"email_body"`
	EmailFrom    string  `json:"email_from"`
	Provider     string  `json:"provider,omitempty"`
	Model        *string `json:"model,omitempty"`
}

// ClassifyResponse is the response from classification
type ClassifyResponse struct {
	IsJobRelated   bool     `json:"is_job_related"`
	Confidence     float64  `json:"confidence"`
	Company        *string  `json:"company,omitempty"`
	Position       *string  `json:"position,omitempty"`
	RecruiterName  *string  `json:"recruiter_name,omitempty"`
	Classification *string  `json:"classification,omitempty"`
	Reasoning      *string  `json:"reasoning,omitempty"`
}

// HealthResponse is the response from health check
type HealthResponse struct {
	Status          string `json:"status"`
	OllamaAvailable bool   `json:"ollama_available"`
	OpenAIAvailable bool   `json:"openai_available"`
}

// New creates a new classifier client
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Long timeout for LLM inference
		},
	}
}

// Health checks if the classification service is running
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to classifier service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("health check failed: %s", string(body))
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &health, nil
}

// IsRunning checks if the service is reachable
func (c *Client) IsRunning(ctx context.Context) bool {
	health, err := c.Health(ctx)
	return err == nil && health.Status == "ok"
}

// EnsureRunning checks if the service is running and returns a helpful error if not
func (c *Client) EnsureRunning(ctx context.Context) error {
	if c.IsRunning(ctx) {
		return nil
	}

	return fmt.Errorf(
		"classification service not running at %s\n\n"+
			"Start it with:\n"+
			"  cd classifier && uvicorn src.classifier.main:app --port 8642\n\n"+
			"Or use: make serve-classifier",
		c.baseURL,
	)
}

// Classify sends an email for classification
func (c *Client) Classify(ctx context.Context, req ClassifyRequest) (*ClassifyResponse, error) {
	// Default to ollama
	if req.Provider == "" {
		req.Provider = "ollama"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/classify", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("classification request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("classification failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result ClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ClassifyWithFallback tries the primary provider, falling back to secondary on failure
func (c *Client) ClassifyWithFallback(ctx context.Context, req ClassifyRequest, primary, fallback string) (*ClassifyResponse, error) {
	req.Provider = primary
	result, err := c.Classify(ctx, req)
	if err == nil {
		return result, nil
	}

	// Try fallback
	if fallback != "" {
		req.Provider = fallback
		return c.Classify(ctx, req)
	}

	return nil, err
}
