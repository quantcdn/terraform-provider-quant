package main

import (
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfgen"

	quant "github.com/quantcdn/terraform-provider-quant/v5/pulumi/provider"
)

func main() {
	tfgen.Main("quant", quant.Provider())
}
