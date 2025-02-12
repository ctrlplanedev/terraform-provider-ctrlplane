// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"terraform-provider-ctrlplane/testing/acctest"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ctrlplane": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Enable Terraform debug logging
	if os.Getenv("TF_LOG") == "" {
		os.Setenv("TF_LOG", "DEBUG")
	}

	// Check for required environment variables
	requiredEnvVars := []string{
		"CTRLPLANE_TOKEN",
		"CTRLPLANE_WORKSPACE",
		"CTRLPLANE_BASE_URL",
	}

	for _, envVar := range requiredEnvVars {
		if value := acctest.GetTestEnv(t, envVar); value == "" {
			t.Fatalf("%s must be set for acceptance tests", envVar)
		} else {
			t.Logf("%s is set to: %s", envVar, value)
		}
	}

	// Call the common pre-check function
	acctest.PreCheck(t)
}
