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

func TestAccEnvironmentResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-env-%d", time.Now().UnixNano())
	updatedName := name + "-updated"
	description := "Terraform acceptance test environment"
	updatedDescription := "Terraform acceptance test environment updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig(name, description),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_environment.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_environment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_environment.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(description),
					),
				},
			},
			{
				Config: testAccEnvironmentResourceConfig(updatedName, updatedDescription),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_environment.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_environment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updatedName),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_environment.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(updatedDescription),
					),
				},
			},
		},
	})
}

func testAccEnvironmentResourceConfig(name, description string) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_system" "test" {
  name = %q
}

resource "ctrlplane_environment" "test" {
  name      = %q
  description = %q

  resource_selector = "resource.name == '%s'"
}
`, testAccProviderConfig(), name, name, description, name)
}
