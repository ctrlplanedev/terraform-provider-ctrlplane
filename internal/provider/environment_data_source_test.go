// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentDataSource(t *testing.T) {
	envName := fmt.Sprintf("test-env-%s", acctest.RandString(8))
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the resources first
			{
				Config: testAccEnvironmentDataSourceConfigSetup(systemName, envName),
			},
			// Basic data source test
			{
				Config: testAccEnvironmentDataSourceConfigBasic(systemName, envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.ctrlplane_environment.test", "id"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test", "name", envName),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test", "description", "Test environment"),
					resource.TestCheckResourceAttrSet("data.ctrlplane_environment.test", "system_id"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test", "metadata.key1", "value1"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test", "metadata.key2", "value2"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test", "resource_filter.type", "metadata"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test", "resource_filter.key", "environment"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test", "resource_filter.operator", "equals"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test", "resource_filter.value", "staging"),
				),
			},
			// Test with complex filter
			{
				Config: testAccEnvironmentDataSourceConfigWithComplexFilter(systemName, envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.ctrlplane_environment.test_complex", "id"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test_complex", "name", envName),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test_complex", "resource_filter.type", "comparison"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test_complex", "resource_filter.operator", "and"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test_complex", "resource_filter.conditions.#", "2"),
				),
			},
		},
	})
}

func TestAccEnvironmentDataSourceErrorHandling(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test with missing required fields (should fail)
			{
				Config:      testAccEnvironmentDataSourceConfigMissingRequired(),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
			// Test with non-existent environment (should fail)
			{
				Config:      testAccEnvironmentDataSourceConfigNonExistent(),
				ExpectError: regexp.MustCompile(`Environment not found`),
			},
		},
	})
}

func testAccEnvironmentDataSourceConfigSetup(systemName, envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = %[1]q
  description = "Test system"
  slug        = %[1]q
}

resource "ctrlplane_environment" "test" {
  name        = %[2]q
  description = "Test environment"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "key1" = "value1"
    "key2" = "value2"
  }
  resource_filter = {
    type     = "metadata"
    key      = "environment"
    operator = "equals"
    value    = "staging"
  }
}

resource "ctrlplane_environment" "test_complex" {
  name        = "complex-%[2]q"
  description = "Test environment with complex filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
  }
  resource_filter = {
    type     = "comparison"
    operator = "and"
    conditions = [
      {
        type     = "metadata"
        key      = "environment"
        operator = "equals"
        value    = "staging"
      },
      {
        type     = "kind"
        operator = "equals"
        value    = "Deployment"
      }
    ]
  }
}
`, systemName, envName)
}

func testAccEnvironmentDataSourceConfigBasic(systemName, envName string) string {
	return fmt.Sprintf(`
%s

data "ctrlplane_environment" "test" {
  name      = ctrlplane_environment.test.name
  system_id = ctrlplane_system.test.id
}
`, testAccEnvironmentDataSourceConfigSetup(systemName, envName))
}

func testAccEnvironmentDataSourceConfigWithComplexFilter(systemName, envName string) string {
	return fmt.Sprintf(`
%s

data "ctrlplane_environment" "test_complex" {
  name      = ctrlplane_environment.test_complex.name
  system_id = ctrlplane_system.test.id
}
`, testAccEnvironmentDataSourceConfigSetup(systemName, envName))
}

func testAccEnvironmentDataSourceConfigMissingRequired() string {
	return `
data "ctrlplane_environment" "test" {
  # Missing name
  system_id = "00000000-0000-0000-0000-000000000000"
}
`
}

func testAccEnvironmentDataSourceConfigNonExistent() string {
	return `
resource "ctrlplane_system" "test" {
  name        = "test-system-nonexistent"
  description = "Test system"
  slug        = "test-system-nonexistent"
}

data "ctrlplane_environment" "test" {
  name      = "non-existent-environment"
  system_id = ctrlplane_system.test.id
}
`
}
