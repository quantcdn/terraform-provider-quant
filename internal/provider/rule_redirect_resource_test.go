package provider_test

import (
	"context"
	"os"

	"testing"

	quantadmingo "github.com/quantcdn/quant-admin-go"
	"github.com/stretchr/testify/assert"
)

func TestListRedirectResource(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")

	cfg := quantadmingo.NewConfiguration()
	client := quantadmingo.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), quantadmingo.ContextAccessToken, bearer)
	rules, _, err := client.RulesRedirectAPI.RulesRedirectList(ctx, "quant", "api-test").Execute()
	assert.Nil(t, err)
	assert.Equal(t, 4, len(rules))
}
