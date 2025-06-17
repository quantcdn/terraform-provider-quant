package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()
	
	if config.RequestsPerSecond != 10.0 {
		t.Errorf("Expected RequestsPerSecond to be 10.0, got %f", config.RequestsPerSecond)
	}
	
	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}
	
	if config.BaseDelay != 500*time.Millisecond {
		t.Errorf("Expected BaseDelay to be 500ms, got %v", config.BaseDelay)
	}
	
	if !config.EnableJitter {
		t.Error("Expected EnableJitter to be true")
	}
}

func TestDefaultRetryCondition(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		expected   bool
	}{
		{"Network error", 0, http.ErrHandlerTimeout, true},
		{"429 Too Many Requests", 429, nil, true},
		{"500 Internal Server Error", 500, nil, true},
		{"502 Bad Gateway", 502, nil, true},
		{"503 Service Unavailable", 503, nil, true},
		{"504 Gateway Timeout", 504, nil, true},
		{"200 OK", 200, nil, false},
		{"404 Not Found", 404, nil, false},
		{"400 Bad Request", 400, nil, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			if tt.statusCode > 0 {
				resp = &http.Response{StatusCode: tt.statusCode}
			}
			
			result := DefaultRetryCondition(resp, tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for status %d", tt.expected, result, tt.statusCode)
			}
		})
	}
}

func TestRateLimitedHTTPClient(t *testing.T) {
	// Create a test HTTP server that counts requests
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()
	
	// Create rate-limited client with very low rate limit for testing
	config := &RateLimitConfig{
		RequestsPerSecond: 2.0, // 2 requests per second
		MaxRetries:        1,
		BaseDelay:         100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		EnableJitter:      false, // Disable jitter for predictable testing
		RetryCondition:    DefaultRetryCondition,
	}
	
	client := NewRateLimitedHTTPClient(config)
	defer client.Close()
	
	// Make requests and measure timing
	start := time.Now()
	
	// Make 3 requests - should take at least 1 second due to rate limiting
	for i := 0; i < 3; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		resp.Body.Close()
	}
	
	duration := time.Since(start)
	
	// Should have made all 3 requests
	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
	
	// Should take at least 500ms due to rate limiting (2 RPS for 3 requests)
	// The first request is immediate, second after 500ms, third after another 500ms
	if duration < 500*time.Millisecond {
		t.Errorf("Expected at least 500ms due to rate limiting, took %v", duration)
	}
}

func TestRateLimitedHTTPClientRetry(t *testing.T) {
	// Create a test server that fails the first few requests
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Return 500 for first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Success on 3rd attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer server.Close()
	
	config := &RateLimitConfig{
		RequestsPerSecond: 100.0, // High rate limit to focus on retry logic
		MaxRetries:        3,
		BaseDelay:         10 * time.Millisecond, // Short delay for testing
		MaxDelay:          1 * time.Second,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}
	
	client := NewRateLimitedHTTPClient(config)
	defer client.Close()
	
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	// Should have succeeded after retries
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}
	
	// Should have made 3 attempts (2 failures + 1 success)
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestRateLimitedRoundTripperClose(t *testing.T) {
	config := DefaultRateLimitConfig()
	rt := NewRateLimitedRoundTripper(nil, config)
	
	// Should not panic when closing
	rt.Close()
}

func TestNewRateLimitedHTTPClientWithNilConfig(t *testing.T) {
	client := NewRateLimitedHTTPClient(nil)
	defer client.Close()
	
	// Should use default configuration
	if client.rateLimiter.config.RequestsPerSecond != 10.0 {
		t.Errorf("Expected default RequestsPerSecond of 10.0, got %f", client.rateLimiter.config.RequestsPerSecond)
	}
}

func TestClientIntegration(t *testing.T) {
	// Test that the client integration works properly
	config := &RateLimitConfig{
		RequestsPerSecond: 5.0,
		MaxRetries:        2,
		BaseDelay:         50 * time.Millisecond,
		MaxDelay:          5 * time.Second,
		EnableJitter:      true,
		RetryCondition:    DefaultRetryCondition,
	}
	
	client := NewWithRateLimit("test-token", "test-org", config)
	defer client.Close()
	
	// Verify the client was created with the correct configuration
	if client.GetRateLimitConfig().RequestsPerSecond != 5.0 {
		t.Errorf("Expected RequestsPerSecond of 5.0, got %f", client.GetRateLimitConfig().RequestsPerSecond)
	}
	
	if client.Bearer != "test-token" {
		t.Errorf("Expected Bearer token 'test-token', got %s", client.Bearer)
	}
	
	if client.Organization != "test-org" {
		t.Errorf("Expected Organization 'test-org', got %s", client.Organization)
	}
} 