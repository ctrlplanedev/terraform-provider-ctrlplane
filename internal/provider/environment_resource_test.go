// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestAccEnvironmentResource(t *testing.T) {
	rName := acctest.RandString(8)
	basicEnvName := fmt.Sprintf("test-env-basic-%s", rName)
	simpleFilterEnvName := fmt.Sprintf("test-env-simple-filter-%s", rName)
	complexFilterEnvName := fmt.Sprintf("test-env-complex-filter-%s", rName)
	updateFilterEnvName := fmt.Sprintf("test-env-update-filter-%s", rName)
	nameFilterEnvName := fmt.Sprintf("test-env-name-filter-%s", rName)
	kindFilterEnvName := fmt.Sprintf("test-env-kind-filter-%s", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test basic create
			{
				Config: testAccEnvironmentResourceConfig(basicEnvName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", basicEnvName),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "description", "Test environment"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "metadata.key1", "value1"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "metadata.key2", "value2"),
				),
			},
			// DISABLED UPDATE TEST: The environment API does not currently support updates (returns 405 Method Not Allowed)
			// The current provider implementation only refreshes the state from the API in the Update function
			// This test can be re-enabled if/when the API implements update support
			/*
				{
					Config: testAccEnvironmentResourceConfigUpdated("test-env-basic-update"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env-basic-update"),
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "description", "Updated test environment"),
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "metadata.key1", "updated1"),
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "metadata.key3", "new_value"),
					),
				},
			*/
			// Test with simple filter
			{
				Config: testAccEnvironmentResourceConfigWithSimpleFilter(simpleFilterEnvName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", simpleFilterEnvName),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.value", "staging"),
				),
			},
			// Test with complex filter
			{
				Config: testAccEnvironmentResourceConfigWithComplexFilter(complexFilterEnvName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", complexFilterEnvName),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "comparison"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "and"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.#", "2"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.value", "production"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.1.type", "name"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.1.operator", "contains"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.1.value", "api"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.operator", "equals"),
				),
			},
			// Test updating the filter
			{
				Config: testAccEnvironmentResourceConfigUpdateFilter(updateFilterEnvName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", updateFilterEnvName),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "comparison"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "or"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.#", "2"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.0.value", "staging"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.1.type", "kind"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.1.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.conditions.1.value", "prod"),
				),
			},
			// Test with name filter
			{
				Config: testAccEnvironmentResourceConfigWithNameFilter(nameFilterEnvName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", nameFilterEnvName),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "name"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "contains"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.value", "api-service"),
				),
			},
			// Test with kind filter
			{
				Config: testAccEnvironmentResourceConfigWithKindFilter(kindFilterEnvName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", kindFilterEnvName),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "kind"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.value", "Service"),
				),
			},
			// DISABLED NOT CONDITION TEST - This is temporarily disabled due to inconsistency in API response
			// The API currently has an issue where setting not=true, the value gets lost when refreshing
			// Error: "Provider produced inconsistent result after apply... was cty.True, but now cty.False"
			/*
				// Test with not condition
				{
					Config: testAccEnvironmentResourceConfigWithNotCondition(notConditionEnvName),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", notConditionEnvName),
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.type", "metadata"),
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.key", "environment"),
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.operator", "equals"),
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.value", "staging"),
						resource.TestCheckResourceAttr("ctrlplane_environment.test", "resource_filter.not", "true"),
					),
				},
			*/
		},
	})
}

func TestAccEnvironmentResourceErrorHandling(t *testing.T) {
	rName := acctest.RandString(8)
	invalidFilterEnvName := fmt.Sprintf("test-env-invalid-filter-%s", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test missing required attributes
			{
				Config:      testAccEnvironmentResourceConfigMissingRequired(),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
			// Test invalid filter configuration
			{
				Config:      testAccEnvironmentResourceConfigInvalidFilter(invalidFilterEnvName),
				ExpectError: regexp.MustCompile(`API Error`),
			},
		},
	})
}

func testAccEnvironmentResourceConfig(envName string) string {
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))
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
}
`, systemName, envName)
}

// TODO: Add updated test once the API supports the needed CRUD operations

/*
func testAccEnvironmentResourceConfigUpdated(envName string) string {
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = %[1]q
  description = "Test system"
  slug        = %[1]q
}

resource "ctrlplane_environment" "test" {
  name        = %[2]q
  description = "Updated test environment"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "key1" = "updated1"
    "key3" = "new_value"
  }
}
`, systemName, envName)
}
*/

func testAccEnvironmentResourceConfigWithSimpleFilter(envName string) string {
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = %[1]q
  description = "Test system"
  slug        = %[1]q
}

resource "ctrlplane_environment" "test" {
  name        = %[2]q
  description = "Test environment with simple filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
  }
  resource_filter = {
    type     = "metadata"
    key      = "environment"
    operator = "equals"
    value    = "staging"
  }
}
`, systemName, envName)
}

func testAccEnvironmentResourceConfigWithComplexFilter(envName string) string {
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = %[1]q
  description = "Test system"
  slug        = %[1]q
}

resource "ctrlplane_environment" "test" {
  name        = %[2]q
  description = "Test environment with complex filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
    "env"  = "integration"
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
`, systemName, envName)
}

func testAccEnvironmentResourceConfigUpdateFilter(envName string) string {
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = %[1]q
  description = "Test system"
  slug        = %[1]q
}

resource "ctrlplane_environment" "test" {
  name        = %[2]q
  description = "Test environment with updated filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
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
        type     = "kind"
        operator = "equals"
        value    = "prod"
      }
    ]
  }
}
`, systemName, envName)
}

func testAccEnvironmentResourceConfigWithNameFilter(envName string) string {
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = %[1]q
  description = "Test system"
  slug        = %[1]q
}

resource "ctrlplane_environment" "test" {
  name        = %[2]q
  description = "Test environment with name filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
  }
  resource_filter = {
    type     = "name"
    operator = "contains"
    value    = "api-service"
  }
}
`, systemName, envName)
}

func testAccEnvironmentResourceConfigWithKindFilter(envName string) string {
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = %[1]q
  description = "Test system"
  slug        = %[1]q
}

resource "ctrlplane_environment" "test" {
  name        = %[2]q
  description = "Test environment with kind filter"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
  }
  resource_filter = {
    type     = "kind"
    operator = "equals"
    value    = "Service"
  }
}
`, systemName, envName)
}

// TODO: Add not condition test once the API supports the needed CRUD operations
/*
func testAccEnvironmentResourceConfigWithNotCondition(envName string) string {
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = %[1]q
  description = "Test system"
  slug        = %[1]q
}

resource "ctrlplane_environment" "test" {
  name        = %[2]q
  description = "Test environment with NOT condition"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "test" = "true"
  }
  resource_filter = {
    type     = "metadata"
    key      = "environment"
    operator = "equals"
    value    = "staging"
    not      = true
  }
}
`, systemName, envName)
}
*/

func testAccEnvironmentResourceConfigMissingRequired() string {
	return `
resource "ctrlplane_environment" "test" {
  description = "Test environment missing required fields"
  # Missing name and system_id
}
`
}

func testAccEnvironmentResourceConfigInvalidFilter(envName string) string {
	systemName := fmt.Sprintf("test-system-%s", acctest.RandString(8))
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = %[1]q
  description = "Test system"
  slug        = %[1]q
}

resource "ctrlplane_environment" "test" {
  name        = %[2]q
  description = "Test environment with invalid filter"
  system_id   = ctrlplane_system.test.id
  resource_filter = {
    type     = "metadata"
    # Missing key which is required for metadata type
    operator = "equals"
    value    = "staging"
  }
}
`, systemName, envName)
}

func TestEnvironmentSchema(t *testing.T) {
	t.Run("ResourceFilter should be optional", func(t *testing.T) {
		schema := GetEnvironmentResourceSchema()
		resourceFilter, exists := schema.Attributes["resource_filter"]

		assert.True(t, exists, "resource_filter should exist in schema")

		rf, ok := resourceFilter.(resourceschema.SingleNestedAttribute)
		assert.True(t, ok, "resource_filter should be of type SingleNestedAttribute")
		assert.True(t, rf.Optional, "resource_filter should be optional")
	})
}
