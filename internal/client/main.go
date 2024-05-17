package client

import (
	"context"

	openapi "github.com/quantcdn/quant-admin-go"
)

type Client struct {
	AuthContext  context.Context
	Bearer       string
	Organization string
	Instance     *openapi.APIClient
}

// Rather than the practioner providing an organization for all resources
// managed by the terraform instance we scope the data client to an organization
// with provider configuration.
func New(bearer string, organization string) *Client {
	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	return &Client{
		Bearer:       bearer,
		AuthContext:  ctx,
		Instance:     client,
		Organization: organization,
	}
}
