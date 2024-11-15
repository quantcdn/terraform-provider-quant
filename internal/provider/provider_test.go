package provider_test

import (
	"terraform-provider-quant/internal/provider"
	"testing"
)

func TestProvider(t *testing.T) {
	// Simple test
	_ = provider.New()
}
