// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	providerConfig = `
	terraform {
		required_providers {
			ctrlplane = {
				source = "hashicorp/ctrlplane"
				version = "0.0.1"
			}
		}
	}

	provider "ctrlplane" {}
	`
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ctrlplane": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("CTRLPLANE_TOKEN"); v == "" {
		t.Fatal("CTRLPLANE_TOKEN must be set for acceptance tests")
	}
	if v := os.Getenv("CTRLPLANE_WORKSPACE"); v == "" {
		t.Fatal("CTRLPLANE_WORKSPACE must be set for acceptance tests")
	}
}
