// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSystemResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSystemResourceConfig("one", "one", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", "one"),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", "one"),
				),
			},
			// Update and Read testing
			{
				Config: testAccSystemResourceConfig("two", "two", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", "two"),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", "two"),
				),
			},
		},
	})
}

func testAccSystemResourceConfig(name string, slug string, description *string) string {
	configTemplate := `
resource "ctrlplane_system" "test" {
  name = %[1]q
  slug = %[2]q
  %s
}`

	descriptionBlock := ""
	if description != nil {
		descriptionBlock = fmt.Sprintf("  description = %q", *description)
	}

	resourceConfig := fmt.Sprintf(configTemplate, name, slug, descriptionBlock)

	return fmt.Sprintf("%s\n%s", providerConfig, resourceConfig)
}
