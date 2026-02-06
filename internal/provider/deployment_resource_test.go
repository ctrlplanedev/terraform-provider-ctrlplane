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

func TestAccDeploymentResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-dep-%d", time.Now().UnixNano())
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentResourceConfig(name, "successful", "value"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
				},
			},
			{
				Config: testAccDeploymentResourceConfig(updatedName, "failure", "updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_deployment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updatedName),
					),
				},
			},
		},
	})
}

func testAccDeploymentResourceConfig(name string, status string, metadataValue string) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_system" "test" {
  name = %q
}

resource "ctrlplane_job_agent" "test" {
  name = %q

  test_runner {
    delay_seconds = 5
    status        = %q
  }
}

resource "ctrlplane_deployment" "test" {
  system_id = ctrlplane_system.test.id
  name      = %q
  metadata = {
    key = %q
  }

  resource_selector = "resource.name == '%s'"

  job_agent {
    id = ctrlplane_job_agent.test.id
    test_runner {
      delay_seconds = 10
    }
  }
}
`, testAccProviderConfig(), name, name+"-ja", status, name, metadataValue, name)
}
