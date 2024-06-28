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

func TestListCustomResponseResource(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	rules, _, err := client.RulesCustomResponseAPI.RulesCustomResponseList(ctx, "quant", "api-test").Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	for _, rule := range rules {
		assert.Equal(t, "custom_response", rule.GetAction())
	}
}

func TestCreateCustomResponseResource(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	req := *openapi.NewRuleCustomResponseRequestWithDefaults()
	req.SetName(fmt.Sprintf("SDK-TF: custom response rule %v", time.Now().Unix()))
	req.SetCustomResponseBody("<h1>Test</h1>")
	req.SetCustomResponseStatusCode(200)

	rule, _, err := client.RulesCustomResponseAPI.RulesCustomResponseCreate(ctx, "quant", "api-test").RuleCustomResponseRequest(req).Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	assert.Equal(t, "custom_response", rule.GetAction())
}
