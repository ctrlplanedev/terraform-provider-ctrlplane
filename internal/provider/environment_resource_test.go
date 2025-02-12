package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResourceConfig("test-env"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env"),
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "description", "Test environment"),
					resource.TestCheckResourceAttrSet("ctrlplane_environment.test", "system_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "ctrlplane_environment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccEnvironmentResourceConfig("test-env-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_environment.test", "name", "test-env-updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccEnvironmentResourceConfig(envName string) string {
	return fmt.Sprintf(`
resource "ctrlplane_system" "test" {
  name        = "test-system"
  description = "Test system"
  slug        = "test-system"
}

resource "ctrlplane_environment" "test" {
  name        = %[1]q
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
`, envName)
}
