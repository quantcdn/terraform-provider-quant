package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// RetryRuleRead retries a rule read operation with exponential backoff to handle API eventual consistency.
// The API takes 5-10s to index new rules, so we retry up to ~25s total.
func RetryRuleRead[T any](
	ctx context.Context,
	readFunc func() (T, *http.Response, error),
	resourceType string,
) (T, *http.Response, error) {
	var result T
	var httpResp *http.Response
	var err error
	
	// Retry with exponential backoff: 0s, 3s, 5s, 7s, 10s = ~25s total
	retryDelays := []int{0, 3, 5, 7, 10}
	
	for attempt, delay := range retryDelays {
		if attempt > 0 {
			tflog.Debug(ctx, fmt.Sprintf("[%s] Retrying rule read (attempt %d/%d) after %ds", 
				resourceType, attempt+1, len(retryDelays), delay))
			time.Sleep(time.Duration(delay) * time.Second)
		}
		
		result, httpResp, err = readFunc()
		
		// If successful or non-retryable error, break
		if err == nil {
			if attempt > 0 {
				tflog.Debug(ctx, fmt.Sprintf("[%s] Rule read succeeded on attempt %d", resourceType, attempt+1))
			}
			break
		}
		
		// Only retry on 404 or 400 (eventual consistency errors)
		if httpResp != nil && httpResp.StatusCode != 404 && httpResp.StatusCode != 400 {
			tflog.Debug(ctx, fmt.Sprintf("[%s] Non-retryable error (status %d), not retrying", 
				resourceType, httpResp.StatusCode))
			break
		}
		
		if attempt < len(retryDelays)-1 {
			tflog.Debug(ctx, fmt.Sprintf("[%s] Got status %d, will retry...", 
				resourceType, httpResp.StatusCode))
		}
	}
	
	return result, httpResp, err
}

