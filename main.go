package main

import (
	"context"
	"log"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/quantcdn/quant",
	}

	err := providerserver.Serve(context.Background(), provider.New(), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
