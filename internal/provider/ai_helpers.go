package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
)

// doAIRequest makes an authenticated HTTP request to the AI v3 API.
// It uses the same HTTP client and bearer token as the SDK client.
func doAIRequest(c *client.Client, method, path string, body interface{}) (*http.Response, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	baseURL := c.Instance.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s%s", baseURL, path)

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Bearer))

	return c.Instance.GetConfig().HTTPClient.Do(req)
}
