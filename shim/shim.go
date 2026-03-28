package shim

import (
	"github.com/quantcdn/terraform-provider-quant/v5/internal/provider"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
)

func NewProvider() fwprovider.Provider {
	return provider.New()()
}
