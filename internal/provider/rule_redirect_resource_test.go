package provider_test

import (
	"context"
	"os"

	"testing"

	openapi "github.com/quantcdn/quant-admin-go"
	"github.com/stretchr/testify/assert"
)

func TestListRedirectResource(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")

	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)
	rules, _, err := client.RulesRedirectAPI.RulesRedirectList(ctx, "quant", "api-test").Execute()
	assert.Nil(t, err)
	assert.Equal(t, 4, len(rules))
}
