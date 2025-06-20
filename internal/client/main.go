package client

import (
	"context"
	"net/http"

	openapi "github.com/quantcdn/quant-admin-go"
)

type Client struct {
	AuthContext     context.Context
	Bearer          string
	Organization    string
	Instance        *openapi.APIClient
	httpClient      *RateLimitedHTTPClient
	rateLimitConfig *RateLimitConfig
}

// ClientOptions allows customization of the client
type ClientOptions struct {
	RateLimitConfig *RateLimitConfig
	HTTPClient      *http.Client
	BaseURL         string
}

// NewWithOptions creates a new client with custom options
func NewWithOptions(bearer string, organization string, opts *ClientOptions) *Client {
	var rateLimitConfig *RateLimitConfig
	var httpClient *RateLimitedHTTPClient
	var baseURL string

	// Use provided rate limit config or default
	if opts != nil && opts.RateLimitConfig != nil {
		rateLimitConfig = opts.RateLimitConfig
	} else {
		rateLimitConfig = DefaultRateLimitConfig()
	}

	// Use provided base URL or default
	if opts != nil && opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}

	// Create rate-limited HTTP client
	if opts != nil && opts.HTTPClient != nil {
		// Wrap existing HTTP client with rate limiting
		rateLimiter := NewRateLimitedRoundTripper(opts.HTTPClient.Transport, rateLimitConfig)
		httpClient = &RateLimitedHTTPClient{
			Client: &http.Client{
				Transport: rateLimiter,
				Timeout:   opts.HTTPClient.Timeout,
			},
			rateLimiter: rateLimiter,
		}
	} else {
		httpClient = NewRateLimitedHTTPClient(rateLimitConfig)
	}

	// Configure OpenAPI client
	cfg := openapi.NewConfiguration()
	cfg.HTTPClient = httpClient.Client
	
	// Set custom base URL if provided
	if baseURL != "" {
		cfg.Servers = []openapi.ServerConfiguration{
			{
				URL: baseURL,
			},
		}
	}
	
	// Add default headers to the configuration
	cfg.AddDefaultHeader("Authorization", "Bearer "+bearer)
	
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	return &Client{
		Bearer:          bearer,
		AuthContext:     ctx,
		Instance:        client,
		Organization:    organization,
		httpClient:      httpClient,
		rateLimitConfig: rateLimitConfig,
	}
}

// Rather than the practioner providing an organization for all resources
// managed by the terraform instance we scope the data client to an organization
// with provider configuration.
func New(bearer string, organization string) *Client {
	return NewWithOptions(bearer, organization, nil)
}

// NewWithRateLimit creates a new client with custom rate limiting configuration
func NewWithRateLimit(bearer string, organization string, config *RateLimitConfig) *Client {
	return NewWithOptions(bearer, organization, &ClientOptions{
		RateLimitConfig: config,
	})
}

// NewWithBaseURL creates a new client with custom base URL
func NewWithBaseURL(bearer string, organization string, baseURL string) *Client {
	return NewWithOptions(bearer, organization, &ClientOptions{
		BaseURL: baseURL,
	})
}

// NewWithRateLimitAndBaseURL creates a new client with both rate limiting and base URL configuration
func NewWithRateLimitAndBaseURL(bearer string, organization string, config *RateLimitConfig, baseURL string) *Client {
	return NewWithOptions(bearer, organization, &ClientOptions{
		RateLimitConfig: config,
		BaseURL:         baseURL,
	})
}

// Close cleans up client resources
func (c *Client) Close() {
	if c.httpClient != nil {
		c.httpClient.Close()
	}
}

// GetRateLimitConfig returns the current rate limit configuration
func (c *Client) GetRateLimitConfig() *RateLimitConfig {
	return c.rateLimitConfig
}

// UpdateRateLimitConfig updates the rate limit configuration
// Note: This requires creating a new HTTP client instance
func (c *Client) UpdateRateLimitConfig(config *RateLimitConfig) {
	// Close existing client
	if c.httpClient != nil {
		c.httpClient.Close()
	}

	// Create new rate-limited HTTP client
	c.httpClient = NewRateLimitedHTTPClient(config)
	c.rateLimitConfig = config

	// Update OpenAPI client configuration
	cfg := openapi.NewConfiguration()
	cfg.HTTPClient = c.httpClient.Client
	cfg.AddDefaultHeader("Authorization", "Bearer "+c.Bearer)
	
	c.Instance = openapi.NewAPIClient(cfg)
}
