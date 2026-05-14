// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

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

func TestAccWorkflowResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-wf-%d", time.Now().UnixNano())
	updatedName := name + "-updated"
	// name is already lowercase + digits + hyphens, so slugify is a no-op.
	derivedSlug := name

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowConfig(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow.test",
						tfjsonpath.New("slug"),
						knownvalue.StringExact(derivedSlug),
					),
				},
			},
			{
				Config: testAccWorkflowConfig(updatedName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updatedName),
					),
					// Slug is sticky: renaming the workflow must NOT change the slug.
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow.test",
						tfjsonpath.New("slug"),
						knownvalue.StringExact(derivedSlug),
					),
				},
			},
			{
				ResourceName:      "ctrlplane_workflow.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccWorkflowResource_ExplicitSlug(t *testing.T) {
	name := fmt.Sprintf("tf-acc-wf-explicit-%d", time.Now().UnixNano())
	slug := fmt.Sprintf("custom-slug-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowConfigWithSlug(name, slug),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_workflow.test",
						tfjsonpath.New("slug"),
						knownvalue.StringExact(slug),
					),
				},
			},
			{
				ResourceName:      "ctrlplane_workflow.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccWorkflowConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "ctrlplane_job_agent" "test" {
  name = %q

  test_runner {
    delay_seconds = 5
    status        = "successful"
  }
}

resource "ctrlplane_workflow" "test" {
  name = %q

  job_agent {
    name     = "test-agent"
    ref      = ctrlplane_job_agent.test.id
    config   = { "delaySeconds" = "5", "status" = "successful" }
    selector = "true"
  }
}
`, testAccProviderConfig(), name+"-agent", name)
}

func testAccWorkflowConfigWithSlug(name, slug string) string {
	return fmt.Sprintf(`
%s

resource "ctrlplane_job_agent" "test" {
  name = %q

  test_runner {
    delay_seconds = 5
    status        = "successful"
  }
}

resource "ctrlplane_workflow" "test" {
  name = %q
  slug = %q

  job_agent {
    name     = "test-agent"
    ref      = ctrlplane_job_agent.test.id
    config   = { "delaySeconds" = "5", "status" = "successful" }
    selector = "true"
  }
}
`, testAccProviderConfig(), name+"-agent", name, slug)
}
