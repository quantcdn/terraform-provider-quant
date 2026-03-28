package main

import (
	"context"
	_ "embed"

	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"

	quant "github.com/quantcdn/terraform-provider-quant/v5/pulumi/provider"
)

//go:embed schema.json
var schema []byte

func main() {
	meta := tfbridge.ProviderMetadata{PackageSchema: schema}
	tfbridge.Main(context.Background(), "quant", quant.Provider(), meta)
}
