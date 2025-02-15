package provider_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	quantadmingo "github.com/quantcdn/quant-admin-go"
	// "github.com/stretchr/testify/assert"
)

func TestListCrawlers(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := quantadmingo.NewConfiguration()
	client := quantadmingo.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), quantadmingo.ContextAccessToken, bearer)

	crawlers, _, err := client.CrawlersAPI.CrawlersList(ctx, "quant", "api-test").Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	t.Logf("Crawlers: %v", crawlers)

}

func TestCreateCrawler(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := quantadmingo.NewConfiguration()
	client := quantadmingo.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), quantadmingo.ContextAccessToken, bearer)

	req := *quantadmingo.NewCrawlerRequestWithDefaults()

	req.SetDomain("https://www.quantcdn.io")
	req.SetBrowserMode(true)
	req.SetName(fmt.Sprintf("SDK TF crawler %v", time.Now().Unix()))

	crawler, _, err := client.CrawlersAPI.CrawlersCreate(ctx, "quant", "api-test").CrawlerRequest(req).Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	t.Logf("Crawler: %v", crawler)
}
