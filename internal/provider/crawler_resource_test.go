package provider_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	openapi "github.com/quantcdn/quant-admin-go"
	// "github.com/stretchr/testify/assert"
)

func TestListCrawlers(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	crawlers, _, err := client.CrawlersAPI.CrawlersList(ctx, "quant", "api-test").Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	t.Logf("Crawlers: %v", crawlers)

}

func TestCreateCrawler(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	req := *openapi.NewCrawlerRequestWithDefaults()

	req.SetDomain("https://www.quantcdn.io")
	req.SetBrowserMode(true)
	req.SetName(fmt.Sprintf("SDK TF crawler %v", time.Now().Unix()))

	crawler, _, err := client.CrawlersAPI.CrawlersCreate(ctx, "quant", "api-test").CrawlerRequest(req).Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	t.Logf("Crawler: %v", crawler)
}
