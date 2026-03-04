package main

import (
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfgen"

	quant "terraform-provider-quant/pulumi/provider"
)

func main() {
	tfgen.Main("quant", quant.Provider())
}
