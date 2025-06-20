package client

import (
	"testing"
)

func TestNewWithBaseURL(t *testing.T) {
	bearer := "test-token"
	organization := "test-org"
	baseURL := "https://custom-api.example.com"

	client := NewWithBaseURL(bearer, organization, baseURL)

	if client.Bearer != bearer {
		t.Errorf("Expected bearer to be %s, got %s", bearer, client.Bearer)
	}

	if client.Organization != organization {
		t.Errorf("Expected organization to be %s, got %s", organization, client.Organization)
	}

	// The OpenAPI client configuration is internal, so we can't easily test it
	// But we can verify the client was created successfully
	if client.Instance == nil {
		t.Error("Expected OpenAPI client instance to be created")
	}
}

func TestNewWithRateLimitAndBaseURL(t *testing.T) {
	bearer := "test-token"
	organization := "test-org"
	baseURL := "https://custom-api.example.com"
	config := DefaultRateLimitConfig()

	client := NewWithRateLimitAndBaseURL(bearer, organization, config, baseURL)

	if client.Bearer != bearer {
		t.Errorf("Expected bearer to be %s, got %s", bearer, client.Bearer)
	}

	if client.Organization != organization {
		t.Errorf("Expected organization to be %s, got %s", organization, client.Organization)
	}

	if client.rateLimitConfig != config {
		t.Error("Expected rate limit config to be set")
	}

	if client.Instance == nil {
		t.Error("Expected OpenAPI client instance to be created")
	}
}

func TestNewWithOptionsWithBaseURL(t *testing.T) {
	bearer := "test-token"
	organization := "test-org"
	baseURL := "https://custom-api.example.com"

	opts := &ClientOptions{
		BaseURL: baseURL,
	}

	client := NewWithOptions(bearer, organization, opts)

	if client.Bearer != bearer {
		t.Errorf("Expected bearer to be %s, got %s", bearer, client.Bearer)
	}

	if client.Organization != organization {
		t.Errorf("Expected organization to be %s, got %s", organization, client.Organization)
	}

	if client.Instance == nil {
		t.Error("Expected OpenAPI client instance to be created")
	}
} 