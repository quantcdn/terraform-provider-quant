package shim

import (
	"terraform-provider-quant/internal/provider"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
)

func NewProvider() fwprovider.Provider {
	return provider.New()()
}
