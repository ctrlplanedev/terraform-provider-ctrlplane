// Copyright (c) HashiCorp, Inc.

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDeploymentSystemLinkResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-dsl-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentSystemLinkResourceConfig(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment_system_link.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment_system_link.test",
						tfjsonpath.New("system_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment_system_link.test",
						tfjsonpath.New("deployment_id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccDeploymentSystemLinkResourceConfig(name string) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_system" "test" {
  name = %q
}

resource "ctrlplane_deployment" "test" {
  name = %q
}

resource "ctrlplane_deployment_system_link" "test" {
  deployment_id = ctrlplane_deployment.test.id
  system_id     = ctrlplane_system.test.id
}
`, testAccProviderConfig(), name, name)
}
