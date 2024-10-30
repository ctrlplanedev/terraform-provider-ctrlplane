// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTargetDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccTargetDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.ctrlplane_target.test", "id", "example-id"),
					resource.TestCheckResourceAttr("data.ctrlplane_target.test", "name", "example-target"),
					resource.TestCheckResourceAttr("data.ctrlplane_target.test", "type", "example-type"),
				),
			},
		},
	})
}

const testAccTargetDataSourceConfig = `
data "ctrlplane_target" "test" {
  id = "example-id"
}
`
