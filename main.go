// terraform-provider-uptimewarden — official Terraform provider for UptimeWarden.
//
// Published to the Terraform Registry under namespace uptimewarden/uptimewarden.
// Usage:
//
//	terraform {
//	  required_providers {
//	    uptimewarden = {
//	      source  = "uptimewarden/uptimewarden"
//	      version = "~> 0.1"
//	    }
//	  }
//	}
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/provider"
)

// version is injected by goreleaser at release time.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/uptimewarden/uptimewarden",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err)
	}
}
