package provider_test

import (
	"context"
	"os"
	"testing"

	quantadmingo "github.com/quantcdn/quant-admin-go"
)

// Simple test to verify that the API method works as expected.
func TestProjectsList(t *testing.T) {

	bearer := os.Getenv("QUANT_BEARER")

	cfg := quantadmingo.NewConfiguration()
	client := quantadmingo.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), quantadmingo.ContextAccessToken, bearer)

	projects, _, err := client.ProjectsAPI.ProjectsList(ctx, "quant").Execute()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	for _, p := range projects {
		t.Logf("Successfully retrieved project: %v", p.GetName())
	}

}
