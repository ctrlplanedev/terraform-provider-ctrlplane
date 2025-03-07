// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
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

func TestEnvironmentSchema(t *testing.T) {
	t.Run("ResourceFilter should be optional", func(t *testing.T) {
		schema := GetEnvironmentResourceSchema()
		resourceFilter, exists := schema.Attributes["resource_filter"]

		assert.True(t, exists, "resource_filter should exist in schema")
		assert.True(t, resourceFilter.(resourceschema.ObjectAttribute).Optional, "resource_filter should be optional")
	})
}
