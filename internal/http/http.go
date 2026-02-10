package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Client wraps HTTP operations with JSON handling.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new HTTP client with a base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DefaultClient creates a client without a base URL (for absolute URLs).
func DefaultClient() *Client {
	return NewClient("")
}

// PostJSON sends a POST request with JSON body and returns JSON response.
func (c *Client) PostJSON(url string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
	// Build full URL if baseURL is set
	fullURL := url
	if c.baseURL != "" {
		fullURL = fmt.Sprintf("%s%s", c.baseURL, url)
	}

	// Marshal request body to JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse JSON response
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return responseData, nil
}

// GetJSON sends a GET request with headers and returns JSON response.
func (c *Client) GetJSON(url string, headers map[string]string) (map[string]interface{}, error) {
	// Build full URL if baseURL is set
	fullURL := url
	if c.baseURL != "" {
		fullURL = fmt.Sprintf("%s%s", c.baseURL, url)
	}

	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse JSON response
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return responseData, nil
}

// PostJSONRaw sends a POST request with raw JSON string body and returns JSON response.
// Useful when you need to send a pre-formatted JSON string.
func (c *Client) PostJSONRaw(url string, jsonBody string, headers map[string]string) (map[string]interface{}, error) {
	// Build full URL if baseURL is set
	fullURL := url
	if c.baseURL != "" {
		fullURL = fmt.Sprintf("%s%s", c.baseURL, url)
	}

	// Create request
	req, err := http.NewRequest("POST", fullURL, bytes.NewBufferString(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse JSON response
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return responseData, nil
}

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// NewErrorResponse creates a standardized error response.
func NewErrorResponse(err error, code int) ErrorResponse {
	return ErrorResponse{
		Error:   err.Error(),
		Message: err.Error(),
		Code:    code,
	}
}

// WriteJSON writes a JSON response to the HTTP response writer.
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

// WriteError writes an error response as JSON.
func WriteError(w http.ResponseWriter, statusCode int, err error) {
	WriteJSON(w, statusCode, NewErrorResponse(err, statusCode))
}
