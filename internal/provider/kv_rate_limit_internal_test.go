package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// limited429 answers 429 with Retry-After for the first n requests, then 200.
func limited429(t *testing.T, n int64, retryAfter string) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= n {
			w.Header().Set("Retry-After", retryAfter)
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func getStatus(url string) func() (int, *http.Response, error) {
	return func() (int, *http.Response, error) {
		resp, err := http.Get(url)
		if err != nil {
			return 0, nil, err
		}
		_ = resp.Body.Close()
		return resp.StatusCode, resp, nil
	}
}

func TestKVRetry429_WaitsTheFullRetryAfterUntilSuccess(t *testing.T) {
	srv, calls := limited429(t, 2, "1")
	start := time.Now()
	status, _, err := kvRetry429(context.Background(), time.Minute, getStatus(srv.URL))
	if err != nil || status != http.StatusOK {
		t.Fatalf("got %d %v, want 200", status, err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
	if waited := time.Since(start); waited < 2*time.Second {
		t.Fatalf("waited %s, want at least 2s (Retry-After 1, twice)", waited)
	}
}

func TestKVRetry429_StopsWhenTheWaitPassesTheTimeout(t *testing.T) {
	srv, calls := limited429(t, 5, "120")
	start := time.Now()
	status, _, _ := kvRetry429(context.Background(), time.Second, getStatus(srv.URL))
	if status != http.StatusTooManyRequests || calls.Load() != 1 {
		t.Fatalf("got %d after %d calls, want one 429", status, calls.Load())
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("it waited although the Retry-After passes the timeout")
	}
}

func TestKVRetryAfter_HasNoCap(t *testing.T) {
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	cases := map[string]time.Duration{
		"120":                           120 * time.Second,
		"3600":                          time.Hour,
		"Sat, 24 Oct 2026 00:01:30 GMT": 90 * time.Second,
		"":                              KVRateLimitDefaultWait,
		"soon":                          KVRateLimitDefaultWait,
	}
	for header, want := range cases {
		resp := &http.Response{Header: http.Header{"Retry-After": []string{header}}}
		if got := kvRetryAfter(resp, now); got != want {
			t.Errorf("%q: got %s, want %s", header, got, want)
		}
	}
}

func TestKVAPIErrorDetail_ExplainsTheRateLimit(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusTooManyRequests}
	if got := kvAPIErrorDetail(resp, errString("429 Too Many Requests")); !strings.Contains(got, "10 index and ledger changes per minute per project") {
		t.Fatalf("detail lacks the limit: %s", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
