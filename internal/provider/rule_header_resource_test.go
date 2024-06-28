package provider_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	openapi "github.com/quantcdn/quant-admin-go"
	"github.com/stretchr/testify/assert"
)

func TestListHeaderResources(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	rules, _, err := client.RulesHeadersAPI.RulesHeadersList(ctx, "quant", "api-test").Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	for _, rule := range rules {
		assert.Equal(t, "headers", rule.GetAction())
	}
}

func TestCreateHeaderResource(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	req := *openapi.NewRuleHeaderRequestWithDefaults()
	req.SetName(fmt.Sprintf("SDK-TF: Header Rule %v", time.Now().Unix()))

	var headers = map[string] string{
		"X-Test-Header": "test",
	}

	req.SetHeaders(headers)

	rule, _, err := client.RulesHeadersAPI.RulesHeadersCreate(ctx, "quant", "api-test").RuleHeaderRequest(req).Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	assert.Equal(t, "headers", rule.GetAction())
}
