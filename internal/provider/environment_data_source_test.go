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
	rName := acctest.RandString(8)
	systemName := fmt.Sprintf("test-system-%s", rName)
	envName := fmt.Sprintf("test-env-%s", rName)
	complexName := fmt.Sprintf("complex-%s", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test basic configuration
			{
				Config: testAccEnvironmentDataSourceConfigSetup(systemName, envName, complexName) +
					testAccEnvironmentDataSourceConfigBasic(envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test", "name", envName),
					resource.TestCheckResourceAttrPair("data.ctrlplane_environment.test", "id", "ctrlplane_environment.test", "id"),
					resource.TestCheckResourceAttrPair("data.ctrlplane_environment.test", "system_id", "ctrlplane_environment.test", "system_id"),
					resource.TestCheckResourceAttrPair("data.ctrlplane_environment.test", "description", "ctrlplane_environment.test", "description"),
				),
			},
			// Test complex filter
			{
				Config: testAccEnvironmentDataSourceConfigSetup(systemName, envName, complexName) +
					testAccEnvironmentDataSourceConfigComplex(complexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test_complex", "name", complexName),
					resource.TestCheckResourceAttrPair("data.ctrlplane_environment.test_complex", "id", "ctrlplane_environment.test_complex", "id"),
					resource.TestCheckResourceAttrPair("data.ctrlplane_environment.test_complex", "system_id", "ctrlplane_environment.test_complex", "system_id"),
					resource.TestCheckResourceAttrPair("data.ctrlplane_environment.test_complex", "description", "ctrlplane_environment.test_complex", "description"),
				),
			},
			// Test with complex filter
			{
				Config: testAccEnvironmentDataSourceConfigWithComplexFilter(systemName, envName, complexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.ctrlplane_environment.test_complex", "id"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test_complex", "name", complexName),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test_complex", "resource_filter.type", "comparison"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test_complex", "resource_filter.operator", "and"),
					resource.TestCheckResourceAttr("data.ctrlplane_environment.test_complex", "resource_filter.conditions.#", "2"),
				),
			},
		},
	})
}

func TestAccEnvironmentDataSourceErrorHandling(t *testing.T) {
	rName := acctest.RandString(8)
	nonExistentEnvName := fmt.Sprintf("non-existent-env-%s", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test with missing name (should fail)
			{
				Config:      testAccEnvironmentDataSourceConfigMissingName(),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
			// Test with a non-existent environment name (should fail)
			{
				Config:      testAccEnvironmentDataSourceConfigNonExistentEnv(nonExistentEnvName),
				ExpectError: regexp.MustCompile(`Environment Not Found`),
			},
		},
	})
}

func testAccEnvironmentDataSourceConfigSetup(systemName, envName, complexName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "%[1]s"
  slug        = "%[1]s"
  description = "Test system"
}

resource "ctrlplane_environment" "test" {
  name        = "%[2]s"
  system_id   = ctrlplane_system.test.id
  description = "Test environment"
  metadata = {
    "key1" = "value1"
    "key2" = "value2"
    "test" = "true"
  }
  resource_filter = {
    type     = "metadata"
    key      = "environment"
    operator = "equals"
    value    = "production"
  }
}

resource "ctrlplane_environment" "test_complex" {
  name        = "%[3]s"
  system_id   = ctrlplane_system.test.id
  description = "Complex environment"
  metadata = {
    "key1"    = "complex_value"
    "complex" = "true"
  }
  resource_filter = {
    type     = "comparison"
    operator = "and"
    conditions = [
      {
        type     = "metadata"
        key      = "environment"
        operator = "equals"
        value    = "production"
      },
      {
        type     = "name"
        operator = "contains"
        value    = "api"
      }
    ]
  }
}
`, systemName, envName, complexName)
}

func testAccEnvironmentDataSourceConfigBasic(envName string) string {
	return fmt.Sprintf(`
data "ctrlplane_environment" "test" {
  name      = "%[1]s"
  system_id = ctrlplane_system.test.id
}
`, envName)
}

func testAccEnvironmentDataSourceConfigWithComplexFilter(systemName, envName, complexName string) string {
	return fmt.Sprintf(`
%s

data "ctrlplane_environment" "test_complex" {
  name      = "%[2]s"
  system_id = ctrlplane_system.test.id
}
`, testAccEnvironmentDataSourceConfigSetup(systemName, envName, complexName), complexName)
}

func testAccEnvironmentDataSourceConfigMissingName() string {
	return `
resource "ctrlplane_system" "test" {
  name        = "test-system-missing-name"
  slug        = "test-system-missing-name"
  description = "Test system"
}

data "ctrlplane_environment" "test" {
  # Missing name field
  system_id = ctrlplane_system.test.id
}
`
}

func testAccEnvironmentDataSourceConfigNonExistentEnv(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system-nonexistent"
  slug        = "test-system-nonexistent" 
  description = "Test system"
}

data "ctrlplane_environment" "test" {
  name      = "%[1]s"
  system_id = ctrlplane_system.test.id
}
`, envName)
}

func testAccEnvironmentDataSourceConfigComplex(complexName string) string {
	return fmt.Sprintf(`
data "ctrlplane_environment" "test_complex" {
  name      = "%[1]s"
  system_id = ctrlplane_system.test.id
}
`, complexName)
}
