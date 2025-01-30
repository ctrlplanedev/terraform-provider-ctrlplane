// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSystemResource(t *testing.T) {
	firstName := acctest.RandString(10)
	secondName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSystemResourceConfig(firstName, firstName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", firstName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", firstName),
				),
			},
			// Update and Read testing
			{
				Config: testAccSystemResourceConfig(secondName, secondName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", secondName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", secondName),
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
