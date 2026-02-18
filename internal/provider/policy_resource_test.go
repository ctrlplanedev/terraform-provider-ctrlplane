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

func TestAccPolicyResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-policy-%d", time.Now().UnixNano())
	updatedName := name + "-updated"
	description := "Terraform acceptance test policy"
	updatedDescription := "Terraform acceptance test policy updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig(name, description, 100, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(description),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("priority"),
						knownvalue.Int64Exact(100),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("enabled"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				Config: testAccPolicyResourceConfig(updatedName, updatedDescription, 200, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updatedName),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(updatedDescription),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("priority"),
						knownvalue.Int64Exact(200),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_policy.test",
						tfjsonpath.New("enabled"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func testAccPolicyResourceConfig(name, description string, priority int, enabled bool) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_policy" "test" {
  name        = %q
  description = %q
  priority    = %d
  enabled     = %t
  selector    = "deployment.name == '%s'"

  version_cooldown {
    duration = "1h"
  }

  deployment_window {
    duration_minutes = 480
    rrule            = "DTSTART:20000101T160000\nRRULE:FREQ=WEEKLY;WKST=MO;BYDAY=MO,TU,WE,TH,FR"
    timezone         = "America/New_York"
    allow_window     = true
  }

  verification {
    trigger_on = "jobSuccess"

    metric {
      name     = "Cluster Agent Deployment Available"
      interval = "30s"
      count    = 3

      success {
        condition = "true"
        threshold = 1
      }

      datadog {
        site     = "us5.datadoghq.com"
        interval = "1m"
        queries = {
          avail = "avg:kubernetes_state.deployment.replicas_available{kube_deployment:datadog-cluster-agent}"
        }
        api_key = "dummy"
        app_key = "dummy"
      }
    }
  }

  gradual_rollout {
    rollout_type        = "linear-normalized"
    time_scale_interval = 14400
  }

  any_approval {
    min_approvals = 1
  }

  environment_progression {
    depends_on_environment_selector = "environment.name == 'qa'"
    minimum_success_percentage      = 80
  }
}
`, testAccProviderConfig(), name, description, priority, enabled, name)
}
