package client

import (
	"context"

	quantadmin "github.com/quantcdn/quant-admin-go"
)

type Client struct {
	Admin *quantadmin.APIClient
	Auth  context.Context

	Organization string
	Project      string
}

func New(secret string, org string) *Client {
	// Create the dashboard api client.
	cfg := quantadmin.NewConfiguration()
	client := quantadmin.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), quantadmin.ContextAccessToken, secret)

	// @todo create lua api client.

	return &Client{
		Admin:        client,
		Auth:         ctx,
		Organization: org,
	}
}
