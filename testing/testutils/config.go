// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package testutils

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-ctrlplane/internal/provider"
	"terraform-provider-ctrlplane/testing/acctest"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during acceptance testing.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ctrlplane": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// TestCase wraps the resource.TestCase with provider configuration.
func TestCase(t *testing.T, steps []resource.TestStep) resource.TestCase {
	return resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	}
}

// CheckDestroy verifies the resource has been destroyed.
func CheckDestroy(s *terraform.State, resourceType string) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != resourceType {
			continue
		}
		return fmt.Errorf("resource %s still exists", resourceType)
	}
	return nil
}

// CheckResourceExists checks if a resource exists.
func CheckResourceExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		return nil
	}
}

// ConfigCompose combines multiple configurations into one.
func ConfigCompose(configs ...string) string {
	var config string
	for _, c := range configs {
		config += c + "\n"
	}
	return config
}
