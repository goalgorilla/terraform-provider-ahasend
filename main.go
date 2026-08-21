// Package main is the entrypoint for the AhaSend Terraform provider binary.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/goalgorilla/terraform-provider-ahasend/internal/provider"
)

var (
	// version is set by GoReleaser via ldflags on release builds.
	version string = "dev"
)

// main serves the provider at registry.terraform.io/goalgorilla/ahasend.
func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/goalgorilla/ahasend",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
