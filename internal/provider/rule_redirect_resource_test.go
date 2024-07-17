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
	rules, _, err := client.RulesRedirectAPI.RulesRedirectList(ctx, "quant", "terraform-project-17-02-2024-3").Execute()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rules))
}

// func TestCreateRedirectResource(t *testing.T) {
//  bearer := os.Getenv("QUANT_BEARER")
// 	cfg := openapi.NewConfiguration()
// 	client := openapi.NewAPIClient(cfg)
// 	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

// 	req := *openapi.NewRuleRedirectRequestWithDefaults()
// 	req.SetName("SDK-TF: Redirect Rule")
// 	req.SetRedirectTo("https://www.google.com")
// 	req.SetRedirectCode("301")
// 	req.SetUrl([]string{"/test"})

// 	rule, _, err := client.RulesRedirectAPI.RulesRedirectCreate(ctx, "quant", "terraform-project-17-02-2024-3").RuleRedirectRequest(req).Execute()
// 	assert.Nil(t, err)
// 	assert.Equal(t, "redirect", rule.GetAction())
// }
