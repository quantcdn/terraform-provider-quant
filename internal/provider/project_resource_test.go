package provider_test

import (
	"context"
	"os"
	"testing"

	openapi "github.com/quantcdn/quant-admin-go"
	"github.com/stretchr/testify/assert"
)

func TestReadProject(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")

	cfg := openapi.NewConfiguration()
	client := openapi.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

	project, res, err := client.ProjectsAPI.ProjectsRead(ctx, "quant", "api-test").Execute()

	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	assert.Equal(t, "api-test", project.GetName())
	assert.Equal(t, res.StatusCode, 200)
}

// func TestCreateProject(t *testing.T) {
// 	bearer := os.Getenv("QUANT_BEARER")

// 	cfg := openapi.NewConfiguration()
// 	client := openapi.NewAPIClient(cfg)
// 	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

// 	req := *openapi.NewProjectRequestWithDefaults()
// 	req.SetName("tf test 7")

// 	p, res, err := client.ProjectsAPI.ProjectsCreate(ctx, "quant").ProjectRequest(req).Execute()

// 	t.Logf("Response: %v", res)

// 	if err != nil {
// 		t.Fatalf("unexpected error, %v", err)
// 	}

// 	t.Logf("Project: %v", p)
// }
