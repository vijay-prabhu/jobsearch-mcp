package classifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ProgressCallback is called with progress updates during batch classification
type ProgressCallback func(current, total int)

// concurrentClassifications is the number of parallel LLM classification calls
const concurrentClassifications = 5

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
	IsJobRelated   bool    `json:"is_job_related"`
	Confidence     float64 `json:"confidence"`
	Company        *string `json:"company,omitempty"`
	Position       *string `json:"position,omitempty"`
	RecruiterName  *string `json:"recruiter_name,omitempty"`
	Classification *string `json:"classification,omitempty"`
	Reasoning      *string `json:"reasoning,omitempty"`
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

// BatchClassifyResult holds the result for a single email in a batch
type BatchClassifyResult struct {
	Index    int
	Response *ClassifyResponse
	Error    error
}

// ClassifyBatch classifies multiple emails in parallel
func (c *Client) ClassifyBatch(ctx context.Context, requests []ClassifyRequest, primary, fallback string) []BatchClassifyResult {
	return c.ClassifyBatchWithProgress(ctx, requests, primary, fallback, nil)
}

// ClassifyBatchWithProgress classifies multiple emails in parallel with progress reporting
func (c *Client) ClassifyBatchWithProgress(ctx context.Context, requests []ClassifyRequest, primary, fallback string, progress ProgressCallback) []BatchClassifyResult {
	results := make([]BatchClassifyResult, len(requests))
	resultChan := make(chan BatchClassifyResult, len(requests))
	var classifiedCount int64

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrentClassifications)

	total := len(requests)
	if progress != nil {
		progress(0, total)
	}

	for i, req := range requests {
		wg.Add(1)
		go func(index int, r ClassifyRequest) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				resultChan <- BatchClassifyResult{Index: index, Error: ctx.Err()}
				return
			}

			// Classify with fallback
			resp, err := c.ClassifyWithFallback(ctx, r, primary, fallback)

			// Report progress
			if progress != nil {
				current := int(atomic.AddInt64(&classifiedCount, 1))
				progress(current, total)
			}

			resultChan <- BatchClassifyResult{Index: index, Response: resp, Error: err}
		}(i, req)
	}

	// Close channel when all done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for r := range resultChan {
		results[r.Index] = r
	}

	return results
}
