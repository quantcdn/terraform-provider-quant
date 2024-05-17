package provider_test

import (
	"context"
	"os"
	"testing"

	openapi "github.com/quantcdn/quant-admin-go"
)

func TestReadProject(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")

	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	project, res, err := client.ProjectsAPI.ProjectsRead(ctx, "quant", "").Execute()

	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	t.Logf("Project: %v", project)
	t.Logf("Resposne: %v", res)
}

func TestCreateProject(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")

	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	req := *openapi.NewProjectRequestWithDefaults()
	req.SetName("go sdk test")

	p, res, err := client.ProjectsAPI.ProjectsCreate(ctx, "quant").ProjectRequest(req).Execute()

	if err != nil {
		t.Fatalf("unexpected error, %v", err)
	}

	t.Logf("Project: %v", p)
	t.Logf("Response: %v", res)

}

// func TestUpdateProject(t *testing.T) {
// 	bearer := os.Getenv("QUANT_BEARER")

// 	cfg := openapi.NewConfiguration()
// 	client := openapi.NewAPIClient(cfg)
// 	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)
// }

// func TestDeleteProject(t *testing.T) {

// }
