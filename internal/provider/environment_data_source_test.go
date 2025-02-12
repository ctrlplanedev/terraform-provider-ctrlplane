// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccEnvironmentDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.ctrlplane_environment.test", "name"),
					resource.TestCheckResourceAttrSet("data.ctrlplane_environment.test", "system_id"),
				),
			},
		},
	})
}

const testAccEnvironmentDataSourceConfig = `
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = "test-env"
  description = "Test environment"
  system_id   = ctrlplane_system.test.id
  metadata = {
    "key1" = "value1"
    "key2" = "value2"
  }
  resource_filter = {
    "type" = "kubernetes"
  }
}

data "ctrlplane_environment" "test" {
  id = ctrlplane_environment.test.id
}
`
