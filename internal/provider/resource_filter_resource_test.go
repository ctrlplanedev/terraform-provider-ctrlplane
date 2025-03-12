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

func TestAccResourceFilterResource(t *testing.T) {
	rName := acctest.RandString(8)
	prodEnvValue := fmt.Sprintf("production-%s", rName)
	stagingEnvValue := fmt.Sprintf("staging-%s", rName)
	apiValue := fmt.Sprintf("api-%s", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - simple metadata filter
			{
				Config: testAccResourceFilterResourceConfigSimple(prodEnvValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "value", prodEnvValue),
				),
			},
			// ImportState testing
			{
				ResourceName:      "ctrlplane_resource_filter.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing - change values
			{
				Config: testAccResourceFilterResourceConfigSimpleUpdated(stagingEnvValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "value", stagingEnvValue),
				),
			},
			// Update and Read testing - change filter type to name
			{
				Config: testAccResourceFilterResourceConfigNameFilter(apiValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "name"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "contains"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "value", apiValue),
				),
			},
			// Update and Read testing - change to complex filter
			{
				Config: testAccResourceFilterResourceConfigComplex(stagingEnvValue, apiValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "comparison"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "or"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.#", "2"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.0.type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.0.key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.0.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.0.value", stagingEnvValue),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.1.type", "name"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.1.operator", "contains"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.1.value", apiValue),
				),
			},
			// Update and Read testing - add not condition
			{
				Config: testAccResourceFilterResourceConfigWithNot(prodEnvValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "value", prodEnvValue),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "not", "true"),
				),
			},
		},
	})
}

func TestAccResourceFilterResourceErrorHandling(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test missing required attributes
			{
				Config:      testAccResourceFilterResourceConfigMissingRequired(),
				ExpectError: regexp.MustCompile(`The argument "type" is required`),
			},
			// Test invalid filter type
			{
				Config:      testAccResourceFilterResourceConfigInvalidFilter(),
				ExpectError: regexp.MustCompile(`The 'key' attribute is required for filter type 'metadata'`),
			},
		},
	})
}

func testAccResourceFilterResourceConfigSimple(envValue string) string {
	return fmt.Sprintf(`
resource "ctrlplane_resource_filter" "test" {
  type     = "metadata"
  key      = "environment"
  operator = "equals"
  value    = %q
}
`, envValue)
}

func testAccResourceFilterResourceConfigSimpleUpdated(envValue string) string {
	return fmt.Sprintf(`
resource "ctrlplane_resource_filter" "test" {
  type     = "metadata"
  key      = "environment"
  operator = "equals"
  value    = %q
}
`, envValue)
}

func testAccResourceFilterResourceConfigNameFilter(nameValue string) string {
	return fmt.Sprintf(`
resource "ctrlplane_resource_filter" "test" {
  type     = "name"
  operator = "contains"
  value    = %q
}
`, nameValue)
}

func testAccResourceFilterResourceConfigComplex(envValue string, nameValue string) string {
	return fmt.Sprintf(`
resource "ctrlplane_resource_filter" "test" {
  type     = "comparison"
  operator = "or"
  conditions = [
    {
      type     = "metadata"
      key      = "environment"
      operator = "equals"
      value    = %q
    },
    {
      type     = "name"
      operator = "contains"
      value    = %q
    }
  ]
}
`, envValue, nameValue)
}

func testAccResourceFilterResourceConfigWithNot(envValue string) string {
	return fmt.Sprintf(`
resource "ctrlplane_resource_filter" "test" {
  type     = "metadata"
  key      = "environment"
  operator = "equals"
  value    = %q
  not      = true
}
`, envValue)
}

func testAccResourceFilterResourceConfigMissingRequired() string {
	return `
resource "ctrlplane_resource_filter" "test" {
  # Missing type
  key      = "environment"
  operator = "equals"
  value    = "production"
}
`
}

func testAccResourceFilterResourceConfigInvalidFilter() string {
	return `
resource "ctrlplane_resource_filter" "test" {
  type     = "metadata"
  # Missing key which is required for metadata type
  operator = "equals"
  value    = "production"
}
`
}
