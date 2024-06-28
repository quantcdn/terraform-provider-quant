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


func TestListAuthResource(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	rules, _, err := client.RulesAuthAPI.RulesAuthList(ctx, "quant", "api-test").Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	for _, rule := range rules {
		assert.Equal(t, "auth", rule.GetAction())
	}
}

func TestCreateAuthResource(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	req := *openapi.NewRuleAuthRequestWithDefaults()
	req.SetName(fmt.Sprintf("SDK-TF: Auth Rule %v", time.Now().Unix()))

	req.SetAuthUser("test")
	req.SetAuthPass("test")

	rule, _, err := client.RulesAuthAPI.RulesAuthCreate(ctx, "quant", "api-test").RuleAuthRequest(req).Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	assert.Equal(t, "auth", rule.GetAction())
}
