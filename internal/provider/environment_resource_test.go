// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestAccEnvironmentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResourceConfig("test-env"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "description", "Test environment"),
					resource.TestCheckResourceAttrSet("ctrlplane_environment.test", "system_id"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "metadata.key1", "value1"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "metadata.key2", "value2"),
				),
			},
			// Update testing - modify name, description, and metadata
			{
				Config: testAccEnvironmentResourceConfigUpdated("test-env-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env-updated"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "description", "Updated test environment"),
					resource.TestCheckResourceAttrSet("ctrlplane_environment.test", "system_id"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "metadata.key1", "updated1"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "metadata.key3", "new_value"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "ctrlplane_environment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test with simple filter
			{
				Config: testAccEnvironmentResourceConfigWithSimpleFilter("test-env-simple-filter"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env-simple-filter"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.value", "staging"),
				),
			},
			// Test with complex filter
			{
				Config: testAccEnvironmentResourceConfigWithComplexFilter("test-env-complex-filter"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env-complex-filter"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "comparison"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "and"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.not", "false"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.#", "2"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.value", "staging"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.1.type", "kind"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.1.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.1.value", "Deployment"),
				),
			},
			// Update filter test - change from simple to complex filter
			{
				Config: testAccEnvironmentResourceConfigUpdateFilter("test-env-filter-update"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env-filter-update"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "comparison"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "or"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.#", "2"),
				),
			},
			// Test with name filter type
			{
				Config: testAccEnvironmentResourceConfigWithNameFilter("test-env-name-filter"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env-name-filter"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "name"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "contains"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.value", "service"),
				),
			},
			// Test with kind filter type
			{
				Config: testAccEnvironmentResourceConfigWithKindFilter("test-env-kind-filter"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env-kind-filter"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "kind"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.value", "Deployment"),
				),
			},
			// Test with NOT condition
			{
				Config: testAccEnvironmentResourceConfigWithNotCondition("test-env-not-condition"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env-not-condition"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.value", "production"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.not", "true"),
				),
			},
		},
	})
}

func TestAccEnvironmentResourceErrorHandling(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test with missing required fields (should fail)
			{
				Config:      testAccEnvironmentResourceConfigMissingRequired(),
				ExpectError: regexp.MustCompile(`The argument "system_id" is required`),
			},
			// Test with invalid filter configuration (should fail)
			{
				Config:      testAccEnvironmentResourceConfigInvalidFilter("test-env-invalid-filter"),
				ExpectError: regexp.MustCompile(`String must contain at least 1 character`),
			},
		},
	})
}

func testAccEnvironmentResourceConfig(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
  description = "Test environment"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "key1" = "value1"
    "key2" = "value2"
  }
}
`, envName)
}

func testAccEnvironmentResourceConfigUpdated(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
  description = "Updated test environment"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "key1" = "updated1"
    "key3" = "new_value"
  }
}
`, envName)
}

func testAccEnvironmentResourceConfigWithSimpleFilter(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
  description = "Test environment with simple filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
    "env"  = "integration"
  }
  resource_filter = {
    type     = "metadata"
    key      = "environment"
    operator = "equals"
    value    = "staging"
  }
}
`, envName)
}

func testAccEnvironmentResourceConfigWithComplexFilter(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
  description = "Test environment with complex filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
    "env"  = "integration"
  }
  resource_filter = {
    type     = "comparison"
    operator = "and"
    not      = false
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
`, envName)
}

func testAccEnvironmentResourceConfigUpdateFilter(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
  description = "Test environment with updated filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
    "env"  = "integration"
  }
  resource_filter = {
    type     = "comparison"
    operator = "or"
    conditions = [
      {
        type     = "metadata"
        key      = "environment"
        operator = "equals"
        value    = "staging"
      },
      {
        type     = "metadata"
        key      = "environment"
        operator = "equals"
        value    = "development"
      }
    ]
  }
}
`, envName)
}

func testAccEnvironmentResourceConfigWithNameFilter(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
  description = "Test environment with name filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
  }
  resource_filter = {
    type     = "name"
    operator = "contains"
    value    = "service"
  }
}
`, envName)
}

func testAccEnvironmentResourceConfigWithKindFilter(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
  description = "Test environment with kind filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
  }
  resource_filter = {
    type     = "kind"
    operator = "equals"
    value    = "Deployment"
  }
}
`, envName)
}

func testAccEnvironmentResourceConfigWithNotCondition(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
  description = "Test environment with NOT condition"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
  }
  resource_filter = {
    type     = "metadata"
    key      = "environment"
    operator = "equals"
    value    = "production"
    not      = true
  }
}
`, envName)
}

func testAccEnvironmentResourceConfigMissingRequired() string {
	return `
resource "ctrlplane_environment" "test" {
  name        = "test-env-missing-required"
  description = "Test environment missing required fields"
  # Missing system_id
}
`
}

func testAccEnvironmentResourceConfigInvalidFilter(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
  description = "Test environment with invalid filter"
  system_id   = ctrlplane_system.test.id
  resource_filter = {
    type     = "metadata"
    # Missing key which is required for metadata type
    operator = "equals"
    value    = "staging"
  }
}
`, envName)
}

func TestEnvironmentSchema(t *testing.T) {
	t.Run("ResourceFilter should be optional", func(t *testing.T) {
		schema := GetEnvironmentResourceSchema()
		resourceFilter, exists := schema.Attributes["resource_filter"]

		assert.True(t, exists, "resource_filter should exist in schema")
		assert.True(t, resourceFilter.(resourceschema.SingleNestedAttribute).Optional, "resource_filter should be optional")
	})
}
