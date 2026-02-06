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

func TestAccDeploymentVariableResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-var-%d", time.Now().UnixNano())
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentVariableResourceConfig(name, "value-1"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment_variable.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment_variable.test",
						tfjsonpath.New("key"),
						knownvalue.StringExact(name),
					),
				},
			},
			{
				Config: testAccDeploymentVariableResourceConfig(updatedName, "value-2"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment_variable.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment_variable.test",
						tfjsonpath.New("key"),
						knownvalue.StringExact(updatedName),
					),
				},
			},
		},
	})
}

func testAccDeploymentVariableResourceConfig(key, defaultValue string) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_system" "test" {
  name = %q
}

resource "ctrlplane_deployment" "test" {
  system_id = ctrlplane_system.test.id
  name      = %q
  resource_selector = "resource.name == '%s'"
}

resource "ctrlplane_deployment_variable" "test" {
  deployment_id = ctrlplane_deployment.test.id
  key           = %q
  description   = "Terraform acceptance test variable"
  default_value = %q
}
`, testAccProviderConfig(), key, key+"-deployment", key, key, defaultValue)
}
