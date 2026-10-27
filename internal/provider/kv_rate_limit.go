package provider

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// The KV API allows 10 index and ledger changes (declare, raise, drop) per
// minute per project, and answers 429 with Retry-After above that. The
// global client retry caps its wait, so these calls retry 429 themselves and
// wait the full Retry-After, until the wait timeout of the resource ends.

// KVRateLimitDefaultWait is the wait after a 429 without a valid Retry-After:
// one change slot at 10 changes per minute.
var KVRateLimitDefaultWait = 6 * time.Second

const kvRateLimitHint = " The KV API allows 10 index and ledger changes per minute per project; the provider retried until the wait timeout ended."

// kvRetry429 runs call again after each 429 answer, after the full
// Retry-After. It returns the first other answer, or the last 429 when the
// next wait would end after timeout, or when ctx ends.
func kvRetry429[T any](ctx context.Context, timeout time.Duration, call func() (T, *http.Response, error)) (T, *http.Response, error) {
	deadline := time.Now().Add(timeout)
	for {
		v, resp, err := call()
		if kvAPIStatus(resp) != http.StatusTooManyRequests {
			return v, resp, err
		}
		wait := kvRetryAfter(resp, time.Now())
		if time.Now().Add(wait).After(deadline) || sleepCtx(ctx, wait) != nil {
			return v, resp, err
		}
	}
}

// kvRetryAfter is the Retry-After of resp (seconds or an HTTP date), with
// no upper cap, or KVRateLimitDefaultWait when the header is not valid.
func kvRetryAfter(resp *http.Response, now time.Time) time.Duration {
	v := resp.Header.Get("Retry-After")
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if at, err := http.ParseTime(v); err == nil {
		return max(at.Sub(now), 0)
	}
	return KVRateLimitDefaultWait
}

// kvAPIErrorDetail is the API error message, with the rate limit explained
// after a 429.
func kvAPIErrorDetail(resp *http.Response, err error) string {
	msg := extractAPIErrorMessage(resp, err)
	if kvAPIStatus(resp) == http.StatusTooManyRequests {
		msg += kvRateLimitHint
	}
	return msg
}
