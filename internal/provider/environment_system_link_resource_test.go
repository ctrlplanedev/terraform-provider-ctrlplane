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

func TestAccEnvironmentSystemLinkResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-esl-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentSystemLinkResourceConfig(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_environment_system_link.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_environment_system_link.test",
						tfjsonpath.New("system_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_environment_system_link.test",
						tfjsonpath.New("environment_id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccEnvironmentSystemLinkResourceConfig(name string) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_system" "test" {
  name = %q
}

resource "ctrlplane_environment" "test" {
  name = %q
}

resource "ctrlplane_environment_system_link" "test" {
  environment_id = ctrlplane_environment.test.id
  system_id      = ctrlplane_system.test.id
}
`, testAccProviderConfig(), name, name)
}
