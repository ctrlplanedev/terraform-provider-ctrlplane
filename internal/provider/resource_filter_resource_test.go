// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceFilterResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - simple metadata filter
			{
				Config: testAccResourceFilterResourceConfigSimple(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "value", "production"),
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
				Config: testAccResourceFilterResourceConfigSimpleUpdated(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "value", "staging"),
				),
			},
			// Update and Read testing - change filter type to name
			{
				Config: testAccResourceFilterResourceConfigNameFilter(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "name"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "contains"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "value", "api"),
				),
			},
			// Update and Read testing - change to complex filter
			{
				Config: testAccResourceFilterResourceConfigComplex(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "comparison"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "and"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.#", "2"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.0.type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.0.key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.0.operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.0.value", "production"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.1.type", "name"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.1.operator", "contains"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "conditions.1.value", "api"),
				),
			},
			// Update and Read testing - add NOT condition
			{
				Config: testAccResourceFilterResourceConfigWithNot(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ctrlplane_resource_filter.test", "id"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "type", "metadata"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "key", "environment"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "operator", "equals"),
					resource.TestCheckResourceAttr("ctrlplane_resource_filter.test", "value", "production"),
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
			// Test with missing required fields (should fail)
			{
				Config:      testAccResourceFilterResourceConfigMissingRequired(),
				ExpectError: regexp.MustCompile(`The argument "type" is required`),
			},
			// Test with invalid filter configuration (should fail)
			{
				Config:      testAccResourceFilterResourceConfigInvalidFilter(),
				ExpectError: regexp.MustCompile(`The 'operator' attribute is required for filter type 'metadata'`),
			},
		},
	})
}

func testAccResourceFilterResourceConfigSimple() string {
	return `
resource "ctrlplane_resource_filter" "test" {
  type     = "metadata"
  key      = "environment"
  operator = "equals"
  value    = "production"
}
`
}

func testAccResourceFilterResourceConfigSimpleUpdated() string {
	return `
resource "ctrlplane_resource_filter" "test" {
  type     = "metadata"
  key      = "environment"
  operator = "equals"
  value    = "staging"
}
`
}

func testAccResourceFilterResourceConfigNameFilter() string {
	return `
resource "ctrlplane_resource_filter" "test" {
  type     = "name"
  operator = "contains"
  value    = "api"
}
`
}

func testAccResourceFilterResourceConfigComplex() string {
	return `
resource "ctrlplane_resource_filter" "test" {
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
`
}

func testAccResourceFilterResourceConfigWithNot() string {
	return `
resource "ctrlplane_resource_filter" "test" {
  type     = "metadata"
  key      = "environment"
  operator = "equals"
  value    = "production"
  not      = true
}
`
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
  key      = "environment"
  # Missing operator
  value    = "production"
}
`
}
