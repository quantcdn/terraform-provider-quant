package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// mockRoundTripper is a configurable mock for http.RoundTripper.
type mockRoundTripper struct {
	responses []*http.Response
	errors    []error
	calls     int
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := m.calls
	m.calls++
	if idx < len(m.responses) {
		return m.responses[idx], m.errors[idx]
	}
	// Return last entry if we exceed the slice length
	last := len(m.responses) - 1
	return m.responses[last], m.errors[last]
}

// newOKResponse builds a minimal 200 response with a body that can be closed.
func newOKResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("OK")),
	}
}

// new429Response builds a 429 response with optional Retry-After header.
func new429Response(retryAfter string) *http.Response {
	h := http.Header{}
	if retryAfter != "" {
		h.Set("Retry-After", retryAfter)
	}
	return &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader("rate limited")),
	}
}

// new500Response builds a 500 response.
func new500Response() *http.Response {
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("error")),
	}
}

// --- Tests for main.go ---

func TestNew(t *testing.T) {
	c := New("tok", "org")
	t.Cleanup(c.Close)

	if c.Bearer != "tok" {
		t.Fatalf("expected bearer 'tok', got %q", c.Bearer)
	}
	if c.Organization != "org" {
		t.Fatalf("expected org 'org', got %q", c.Organization)
	}
	if c.Instance == nil {
		t.Fatal("expected Instance to be non-nil")
	}
	if c.RulesMutex == nil {
		t.Fatal("expected RulesMutex to be non-nil")
	}
	if c.httpClient == nil {
		t.Fatal("expected httpClient to be non-nil")
	}
	if c.rateLimitConfig == nil {
		t.Fatal("expected rateLimitConfig to be non-nil")
	}
	// Default config should be applied
	if c.rateLimitConfig.RequestsPerSecond != 10.0 {
		t.Fatalf("expected default RPS 10, got %f", c.rateLimitConfig.RequestsPerSecond)
	}
}

func TestUpdateRateLimitConfig(t *testing.T) {
	c := New("tok", "org")
	t.Cleanup(c.Close)

	newConfig := &RateLimitConfig{
		RequestsPerSecond: 5.0,
		MaxRetries:        1,
		BaseDelay:         200 * time.Millisecond,
		MaxDelay:          2 * time.Second,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	c.UpdateRateLimitConfig(newConfig)

	if c.rateLimitConfig != newConfig {
		t.Fatal("expected rateLimitConfig to be updated")
	}
	if c.rateLimitConfig.RequestsPerSecond != 5.0 {
		t.Fatalf("expected RPS 5, got %f", c.rateLimitConfig.RequestsPerSecond)
	}
	if c.httpClient == nil {
		t.Fatal("expected httpClient to be non-nil after update")
	}
	if c.Instance == nil {
		t.Fatal("expected Instance to be non-nil after update")
	}
}

func TestNewWithOptions_CustomTransport(t *testing.T) {
	customTransport := &mockRoundTripper{
		responses: []*http.Response{newOKResponse()},
		errors:    []error{nil},
	}
	opts := &ClientOptions{
		HTTPClient: &http.Client{
			Transport: customTransport,
		},
	}

	c := NewWithOptions("tok", "org", opts)
	t.Cleanup(c.Close)

	if c.Bearer != "tok" {
		t.Fatalf("expected bearer 'tok', got %q", c.Bearer)
	}
	// Verify it uses default timeout (120s) since HTTPClient.Timeout is 0
	if c.httpClient.Client.Timeout != 120*time.Second {
		t.Fatalf("expected default timeout 120s, got %v", c.httpClient.Client.Timeout)
	}
}

func TestNewWithOptions_CustomTimeout(t *testing.T) {
	customTimeout := 45 * time.Second
	opts := &ClientOptions{
		HTTPClient: &http.Client{
			Timeout: customTimeout,
		},
	}

	c := NewWithOptions("tok", "org", opts)
	t.Cleanup(c.Close)

	if c.httpClient.Client.Timeout != customTimeout {
		t.Fatalf("expected timeout %v, got %v", customTimeout, c.httpClient.Client.Timeout)
	}
}

func TestNewWithOptions_CustomTransportAndTimeout(t *testing.T) {
	customTransport := &mockRoundTripper{
		responses: []*http.Response{newOKResponse()},
		errors:    []error{nil},
	}
	customTimeout := 90 * time.Second
	opts := &ClientOptions{
		HTTPClient: &http.Client{
			Transport: customTransport,
			Timeout:   customTimeout,
		},
	}

	c := NewWithOptions("tok", "org", opts)
	t.Cleanup(c.Close)

	if c.httpClient.Client.Timeout != customTimeout {
		t.Fatalf("expected timeout %v, got %v", customTimeout, c.httpClient.Client.Timeout)
	}
}

// --- Tests for rate_limiter.go ---

func TestNewRateLimitedRoundTripper_NilTransport(t *testing.T) {
	rt := NewRateLimitedRoundTripper(nil, DefaultRateLimitConfig())
	t.Cleanup(func() { rt.Close() })

	if rt.transport == nil {
		t.Fatal("expected transport to fall back to DefaultTransport, got nil")
	}
}

func TestNewRateLimitedRoundTripper_NilConfig(t *testing.T) {
	rt := NewRateLimitedRoundTripper(http.DefaultTransport, nil)
	t.Cleanup(func() { rt.Close() })

	if rt.config == nil {
		t.Fatal("expected config to fall back to default, got nil")
	}
	if rt.config.RequestsPerSecond != 10.0 {
		t.Fatalf("expected default RPS 10, got %f", rt.config.RequestsPerSecond)
	}
}

func TestNewRateLimitedRoundTripper_ZeroRPS(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 0,
		MaxRetries:        0,
		BaseDelay:         100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{newOKResponse()},
		errors:    []error{nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	// With 0 RPS the ticker is not started, so Close just closes done channel.
	t.Cleanup(func() { rt.Close() })

	// rateLimiterCapacity should be capped at 1
	if cap(rt.rateLimiter) != 1 {
		t.Fatalf("expected capacity 1 for 0 RPS, got %d", cap(rt.rateLimiter))
	}

	// Should still be able to make a request (no rate limiting wait)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestNewRateLimitedRoundTripper_FractionalRPS(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 0.5, // Less than 1
		MaxRetries:        0,
		BaseDelay:         100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	rt := NewRateLimitedRoundTripper(http.DefaultTransport, config)
	t.Cleanup(func() { rt.Close() })

	// math.Ceil(0.5) == 1
	if cap(rt.rateLimiter) != 1 {
		t.Fatalf("expected capacity 1 for 0.5 RPS, got %d", cap(rt.rateLimiter))
	}
}

func TestRoundTrip_ContextCancelledDuringRateLimitWait(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 1.0,
		MaxRetries:        0,
		BaseDelay:         100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{newOKResponse()},
		errors:    []error{nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	// Drain the rate limiter so the next call blocks waiting for a token
	select {
	case <-rt.rateLimiter:
	default:
	}

	// Use an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRoundTrip_PerRequestTimeout(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        0,
		BaseDelay:         10 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
		RequestTimeout:    5 * time.Second,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{newOKResponse()},
		errors:    []error{nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRoundTrip_ContextCancelledDuringRetryWait(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        3,
		BaseDelay:         5 * time.Second, // Long delay so we can cancel during wait
		MaxDelay:          30 * time.Second,
		EnableJitter:      false,
		RetryCondition:    func(resp *http.Response, err error) bool { return true },
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{new500Response(), newOKResponse()},
		errors:    []error{nil, nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error from context cancellation during retry wait")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestRoundTrip_ResponseBodyClosedBeforeRetry(t *testing.T) {
	// This test verifies that the response body is closed before retrying.
	// We use a custom body that tracks whether Close was called.
	closeCalled := false
	bodyReader := &trackingCloser{
		Reader:  strings.NewReader("error body"),
		onClose: func() { closeCalled = true },
	}

	resp500 := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{},
		Body:       bodyReader,
	}

	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        1,
		BaseDelay:         1 * time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{resp500, newOKResponse()},
		errors:    []error{nil, nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if !closeCalled {
		t.Fatal("expected response body to be closed before retry")
	}
}

// trackingCloser wraps a Reader and calls onClose when Close is called.
type trackingCloser struct {
	io.Reader
	onClose func()
}

func (tc *trackingCloser) Close() error {
	if tc.onClose != nil {
		tc.onClose()
	}
	return nil
}

func TestRoundTrip_ContextCancelledDuringRetryWait_WithNilResp(t *testing.T) {
	// Test the retry wait context cancellation path when resp is nil (error-based retry).
	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        3,
		BaseDelay:         5 * time.Second,
		MaxDelay:          30 * time.Second,
		EnableJitter:      false,
		RetryCondition:    func(resp *http.Response, err error) bool { return err != nil },
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{nil, newOKResponse()},
		errors:    []error{errors.New("network error"), nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error from context cancellation during retry wait")
	}
	// When resp is nil, the body close branch is skipped — just context error returned
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestRoundTrip_ContextCancelledDuringRetryWait_BodyCloseBeforeCancel(t *testing.T) {
	// Test the context cancellation path where resp is non-nil, so resp.Body.Close() is called.
	closeCalled := false
	bodyReader := &trackingCloser{
		Reader:  strings.NewReader("error body"),
		onClose: func() { closeCalled = true },
	}
	resp500 := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{},
		Body:       bodyReader,
	}

	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        3,
		BaseDelay:         5 * time.Second,
		MaxDelay:          30 * time.Second,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{resp500, newOKResponse()},
		errors:    []error{nil, nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
	if !closeCalled {
		t.Fatal("expected response body to be closed when context cancelled during retry wait")
	}
}

// --- Tests for calculateDelay ---

func TestCalculateDelay_RetryAfterHeader(t *testing.T) {
	config := &RateLimitConfig{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		EnableJitter: false,
	}

	rt := &RateLimitedRoundTripper{config: config}

	resp := new429Response("5")
	delay := rt.calculateDelay(0, resp)

	if delay != 5*time.Second {
		t.Fatalf("expected 5s delay from Retry-After header, got %v", delay)
	}
}

func TestCalculateDelay_RetryAfterExceedsMaxDelay(t *testing.T) {
	config := &RateLimitConfig{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     3 * time.Second,
		EnableJitter: false,
	}

	rt := &RateLimitedRoundTripper{config: config}

	resp := new429Response("60") // 60 seconds > 3s max
	delay := rt.calculateDelay(0, resp)

	if delay != 3*time.Second {
		t.Fatalf("expected max delay 3s, got %v", delay)
	}
}

func TestCalculateDelay_RetryAfterInvalidValue(t *testing.T) {
	config := &RateLimitConfig{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		EnableJitter: false,
	}

	rt := &RateLimitedRoundTripper{config: config}

	// Non-numeric Retry-After should fall through to exponential backoff
	resp := new429Response("invalid")
	delay := rt.calculateDelay(0, resp)

	// Attempt 0: 100ms * 2^0 = 100ms
	expected := 100 * time.Millisecond
	if delay != expected {
		t.Fatalf("expected %v from exponential backoff fallback, got %v", expected, delay)
	}
}

func TestCalculateDelay_NilResponse(t *testing.T) {
	config := &RateLimitConfig{
		BaseDelay:    200 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		EnableJitter: false,
	}

	rt := &RateLimitedRoundTripper{config: config}

	// nil response should use exponential backoff
	delay := rt.calculateDelay(0, nil)

	expected := 200 * time.Millisecond
	if delay != expected {
		t.Fatalf("expected %v, got %v", expected, delay)
	}
}

func TestCalculateDelay_ExponentialBackoff(t *testing.T) {
	config := &RateLimitConfig{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		EnableJitter: false,
	}

	rt := &RateLimitedRoundTripper{config: config}

	// Attempt 0: 100ms * 2^0 = 100ms
	d0 := rt.calculateDelay(0, nil)
	if d0 != 100*time.Millisecond {
		t.Fatalf("attempt 0: expected 100ms, got %v", d0)
	}

	// Attempt 1: 100ms * 2^1 = 200ms
	d1 := rt.calculateDelay(1, nil)
	if d1 != 200*time.Millisecond {
		t.Fatalf("attempt 1: expected 200ms, got %v", d1)
	}

	// Attempt 2: 100ms * 2^2 = 400ms
	d2 := rt.calculateDelay(2, nil)
	if d2 != 400*time.Millisecond {
		t.Fatalf("attempt 2: expected 400ms, got %v", d2)
	}
}

func TestCalculateDelay_ExponentialBackoffCappedAtMaxDelay(t *testing.T) {
	config := &RateLimitConfig{
		BaseDelay:    1 * time.Second,
		MaxDelay:     5 * time.Second,
		EnableJitter: false,
	}

	rt := &RateLimitedRoundTripper{config: config}

	// Attempt 10: 1s * 2^10 = 1024s >> 5s max
	delay := rt.calculateDelay(10, nil)
	if delay != 5*time.Second {
		t.Fatalf("expected max delay 5s, got %v", delay)
	}
}

// --- Tests for addJitter ---

func TestAddJitter_Disabled(t *testing.T) {
	config := &RateLimitConfig{
		EnableJitter: false,
	}
	rt := &RateLimitedRoundTripper{config: config}

	delay := 1 * time.Second
	result := rt.addJitter(delay)

	if result != delay {
		t.Fatalf("expected jitter disabled to return exact delay %v, got %v", delay, result)
	}
}

func TestAddJitter_Enabled(t *testing.T) {
	config := &RateLimitConfig{
		EnableJitter: true,
	}
	rt := &RateLimitedRoundTripper{config: config}

	delay := 1 * time.Second
	result := rt.addJitter(delay)

	// With jitter enabled, result should be >= delay and <= delay * 1.25
	if result < delay {
		t.Fatalf("expected result >= %v, got %v", delay, result)
	}
	maxJitter := time.Duration(float64(delay) * 1.25)
	if result > maxJitter {
		t.Fatalf("expected result <= %v, got %v", maxJitter, result)
	}
}

// --- Test for WithContext ---

func TestWithContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	newReq := WithContext(ctx, req)

	if newReq.Context() != ctx {
		t.Fatal("expected WithContext to set the provided context on the request")
	}
	if newReq.URL.String() != req.URL.String() {
		t.Fatalf("expected URL to remain %q, got %q", req.URL.String(), newReq.URL.String())
	}
}

// --- Test for RoundTrip with zero RPS (no rate limiter blocking) ---

func TestRoundTrip_NoRateLimiting(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 0,
		MaxRetries:        0,
		BaseDelay:         100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{newOKResponse()},
		errors:    []error{nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

// --- Test that all retries exhausted returns last response/error ---

func TestRoundTrip_AllRetriesExhausted(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        2,
		BaseDelay:         1 * time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{new500Response(), new500Response(), new500Response()},
		errors:    []error{nil, nil, nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("expected nil error after exhausted retries, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected last response to be returned")
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRoundTrip_SuccessOnRetry(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        2,
		BaseDelay:         1 * time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{new500Response(), newOKResponse()},
		errors:    []error{nil, nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRoundTrip_RetryWithRetryAfterHeader(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        1,
		BaseDelay:         1 * time.Millisecond,
		MaxDelay:          10 * time.Second,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{new429Response("1"), newOKResponse()},
		errors:    []error{nil, nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	start := time.Now()
	resp, err := rt.RoundTrip(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Should have waited at least ~1 second due to Retry-After: 1
	if elapsed < 900*time.Millisecond {
		t.Fatalf("expected delay of at least ~1s from Retry-After, got %v", elapsed)
	}
}

// Test retry with nil response (error-based retry) that completes normally
func TestRoundTrip_RetryWithNilResponse(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        1,
		BaseDelay:         1 * time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		EnableJitter:      false,
		RetryCondition:    func(resp *http.Response, err error) bool { return err != nil },
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{nil, newOKResponse()},
		errors:    []error{errors.New("network error"), nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// Test per-request timeout with retry — exercises the RequestTimeout + retry path together
func TestRoundTrip_PerRequestTimeoutWithRetry(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerSecond: 100.0,
		MaxRetries:        1,
		BaseDelay:         1 * time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		EnableJitter:      false,
		RetryCondition:    DefaultRetryCondition,
		RequestTimeout:    5 * time.Second,
	}

	mock := &mockRoundTripper{
		responses: []*http.Response{new500Response(), newOKResponse()},
		errors:    []error{nil, nil},
	}

	rt := NewRateLimitedRoundTripper(mock, config)
	t.Cleanup(func() { rt.Close() })

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// --- Test for Close with nil httpClient ---

func TestClient_Close_NilHTTPClient(t *testing.T) {
	c := &Client{}
	// Should not panic
	c.Close()
}

// --- Test for RateLimitedHTTPClient Close with nil rateLimiter ---

func TestRateLimitedHTTPClient_Close_NilRateLimiter(t *testing.T) {
	c := &RateLimitedHTTPClient{
		Client: &http.Client{},
	}
	// Should not panic
	c.Close()
}
