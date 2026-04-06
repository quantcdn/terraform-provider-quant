package client

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// Maximum number of requests per second
	RequestsPerSecond float64
	// Maximum number of retry attempts
	MaxRetries int
	// Base delay for exponential backoff (in milliseconds)
	BaseDelay time.Duration
	// Maximum delay for exponential backoff (in milliseconds)
	MaxDelay time.Duration
	// Whether to add jitter to delays
	EnableJitter bool
	// Custom retry condition function
	RetryCondition func(*http.Response, error) bool
	// Per-request timeout (optional, for fine-grained timeout control)
	RequestTimeout time.Duration
}

// DefaultRateLimitConfig returns sensible defaults for rate limiting
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerSecond: 10.0, // 10 requests per second
		MaxRetries:        3,
		BaseDelay:         500 * time.Millisecond,
		MaxDelay:          30 * time.Second,
		EnableJitter:      true,
		RetryCondition:    DefaultRetryCondition,
	}
}

// DefaultRetryCondition determines if a request should be retried
func DefaultRetryCondition(resp *http.Response, err error) bool {
	// Retry on network errors, including timeout errors
	if err != nil {
		// Check for specific timeout-related errors that might indicate deadlocks
		errStr := err.Error()
		if strings.Contains(errStr, "context deadline exceeded") ||
			strings.Contains(errStr, "Client.Timeout exceeded") ||
			strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "connection refused") {
			return true
		}
		// Retry on other network errors as well
		return true
	}

	// Retry on server errors and rate limiting
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusTooManyRequests: // 429
			return true
		case http.StatusInternalServerError: // 500
			return true
		case http.StatusBadGateway: // 502
			return true
		case http.StatusServiceUnavailable: // 503
			return true
		case http.StatusGatewayTimeout: // 504
			return true
		case http.StatusRequestTimeout: // 408
			return true
		}
	}

	return false
}

// RateLimitedRoundTripper implements http.RoundTripper with rate limiting and retry logic
type RateLimitedRoundTripper struct {
	transport http.RoundTripper
	config    *RateLimitConfig
	// Channel for rate limiting
	rateLimiter chan struct{}
	// Ticker for rate limiting
	ticker *time.Ticker
	// Done channel for cleanup
	done chan bool
}

// NewRateLimitedRoundTripper creates a new rate-limited round tripper
func NewRateLimitedRoundTripper(transport http.RoundTripper, config *RateLimitConfig) *RateLimitedRoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	// Create rate limiter channel
	rateLimiterCapacity := int(math.Ceil(config.RequestsPerSecond))
	if rateLimiterCapacity < 1 {
		rateLimiterCapacity = 1
	}
	rateLimiter := make(chan struct{}, rateLimiterCapacity)

	// Fill the initial bucket
	for i := 0; i < cap(rateLimiter); i++ {
		rateLimiter <- struct{}{}
	}

	rt := &RateLimitedRoundTripper{
		transport:   transport,
		config:      config,
		rateLimiter: rateLimiter,
		done:        make(chan bool),
	}

	// Start the rate limiter ticker
	if config.RequestsPerSecond > 0 {
		interval := time.Duration(float64(time.Second) / config.RequestsPerSecond)
		rt.ticker = time.NewTicker(interval)

		go func() {
			for {
				select {
				case <-rt.ticker.C:
					select {
					case rt.rateLimiter <- struct{}{}:
					default:
						// Channel is full, skip this tick
					}
				case <-rt.done:
					return
				}
			}
		}()
	}

	return rt
}

// RoundTrip implements the http.RoundTripper interface with rate limiting and retries
func (rt *RateLimitedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	// Buffer the request body so retries can replay it. Without this,
	// the body is consumed on the first attempt and retries send an empty
	// body, causing 400 errors from servers/CDNs.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	for attempt := 0; attempt <= rt.config.MaxRetries; attempt++ {
		// Restore body for retries
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		// Wait for rate limiter token
		if rt.config.RequestsPerSecond > 0 {
			select {
			case <-rt.rateLimiter:
				// Got a token, proceed
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}

		// Apply per-request timeout if configured
		var cancel context.CancelFunc
		requestReq := req
		if rt.config.RequestTimeout > 0 {
			requestCtx, c := context.WithTimeout(context.Background(), rt.config.RequestTimeout)
			cancel = c
			requestReq = req.WithContext(requestCtx)
		}

		// Make the request
		resp, err := rt.transport.RoundTrip(requestReq)
		
		// Clean up timeout context
		if cancel != nil {
			cancel()
		}

		// Check if we should retry
		if attempt < rt.config.MaxRetries && rt.config.RetryCondition(resp, err) {
			lastResp = resp
			lastErr = err

			// Calculate delay with exponential backoff
			delay := rt.calculateDelay(attempt, resp)

			// Wait before retrying, but respect context cancellation
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-req.Context().Done():
				if resp != nil {
					_ = resp.Body.Close()
				}
				return nil, req.Context().Err()
			}

			// Close the response body before retrying
			if resp != nil {
				_ = resp.Body.Close()
			}

			continue
		}

		// Success or non-retryable error
		return resp, err
	}

	// All retries exhausted
	return lastResp, lastErr
}

// calculateDelay calculates the delay for the next retry attempt
func (rt *RateLimitedRoundTripper) calculateDelay(attempt int, resp *http.Response) time.Duration {
	// Check for Retry-After header first
	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				delay := time.Duration(seconds) * time.Second
				if delay > rt.config.MaxDelay {
					delay = rt.config.MaxDelay
				}
				return rt.addJitter(delay)
			}
		}
	}

	// Exponential backoff: baseDelay * 2^attempt
	delay := time.Duration(float64(rt.config.BaseDelay) * math.Pow(2, float64(attempt)))

	// Cap at maximum delay
	if delay > rt.config.MaxDelay {
		delay = rt.config.MaxDelay
	}

	return rt.addJitter(delay)
}

// addJitter adds random jitter to the delay to avoid thundering herd
func (rt *RateLimitedRoundTripper) addJitter(delay time.Duration) time.Duration {
	if !rt.config.EnableJitter {
		return delay
	}

	// Add up to 25% jitter
	jitter := time.Duration(rand.Float64() * float64(delay) * 0.25)
	return delay + jitter
}

// Close cleans up the rate limiter resources
func (rt *RateLimitedRoundTripper) Close() {
	if rt.ticker != nil {
		rt.ticker.Stop()
	}
	close(rt.done)
}

// RateLimitedHTTPClient wraps an HTTP client with rate limiting
type RateLimitedHTTPClient struct {
	*http.Client
	rateLimiter *RateLimitedRoundTripper
}

// NewRateLimitedHTTPClient creates a new HTTP client with rate limiting
func NewRateLimitedHTTPClient(config *RateLimitConfig) *RateLimitedHTTPClient {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	// Create base HTTP client with reasonable timeouts
	baseClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	rateLimiter := NewRateLimitedRoundTripper(baseClient.Transport, config)

	return &RateLimitedHTTPClient{
		Client: &http.Client{
			Transport: rateLimiter,
			Timeout:   baseClient.Timeout,
		},
		rateLimiter: rateLimiter,
	}
}

// Close cleans up resources
func (c *RateLimitedHTTPClient) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Close()
	}
}

// WithContext creates a new request with context for better cancellation handling
func WithContext(ctx context.Context, req *http.Request) *http.Request {
	return req.WithContext(ctx)
}
