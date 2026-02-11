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

func TestAccWorkflowTemplateResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-wft-%d", time.Now().UnixNano())
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccWorkflowTemplateResourceConfig(name, "default-val", 5, "successful", `resource.name == "test"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow_template.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow_template.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow_template.test",
						tfjsonpath.New("input"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow_template.test",
						tfjsonpath.New("job"),
						knownvalue.ListSizeExact(1),
					),
				},
			},
			// Update and verify
			{
				Config: testAccWorkflowTemplateResourceConfig(updatedName, "updated-val", 10, "failure", `resource.name == "updated"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow_template.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow_template.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updatedName),
					),
				},
			},
		},
	})
}

func testAccWorkflowTemplateResourceConfig(name string, defaultInput string, delaySeconds int, status string, ifExpr string) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_job_agent" "test" {
  name = %q

  test_runner {
    delay_seconds = %d
    status        = %q
  }
}

resource "ctrlplane_workflow_template" "test" {
  name = %q

  input {
    key = "env"
    string {
      default = %q
    }
  }

  job {
    key = "deploy"
    if  = %q

    agent {
      ref = ctrlplane_job_agent.test.id

      test_runner {
        delay_seconds = %d
        status        = %q
      }
    }
  }
}
`, testAccProviderConfig(), name+"-ja", delaySeconds, status, name, defaultInput, ifExpr, delaySeconds, status)
}
