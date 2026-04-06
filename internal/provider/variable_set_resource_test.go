// Copyright IBM Corp. 2021, 2026

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

func TestAccVariableSetResource_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-varset-%d", time.Now().UnixNano())
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with basic attributes
			{
				Config: testAccVariableSetResourceConfig(name, "Test variable set", "true", 1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test variable set"),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("selector"),
						knownvalue.StringExact("true"),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("priority"),
						knownvalue.Int64Exact(1),
					),
				},
			},
			// ImportState
			{
				ResourceName:      "ctrlplane_variable_set.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name, description, and priority
			{
				Config: testAccVariableSetResourceConfig(updatedName, "Updated description", "true", 5),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updatedName),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("priority"),
						knownvalue.Int64Exact(5),
					),
				},
			},
		},
	})
}

func TestAccVariableSetResource_withLiteralVariables(t *testing.T) {
	name := fmt.Sprintf("tf-acc-varset-lit-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with literal string variables
			{
				Config: testAccVariableSetWithLiteralVariablesConfig(name, "val1", "val2"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("variables"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
			// Update variable values
			{
				Config: testAccVariableSetWithLiteralVariablesConfig(name, "updated1", "updated2"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("variables"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
		},
	})
}

func TestAccVariableSetResource_withReferenceVariables(t *testing.T) {
	name := fmt.Sprintf("tf-acc-varset-ref-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVariableSetWithReferenceVariableConfig(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("variables"),
						knownvalue.ListSizeExact(1),
					),
				},
			},
		},
	})
}

func TestAccVariableSetResource_defaultPriority(t *testing.T) {
	name := fmt.Sprintf("tf-acc-varset-defpri-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVariableSetDefaultPriorityConfig(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("priority"),
						knownvalue.Int64Exact(0),
					),
				},
			},
		},
	})
}

func TestAccVariableSetResource_noVariables(t *testing.T) {
	name := fmt.Sprintf("tf-acc-varset-novar-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVariableSetResourceConfig(name, "No variables", "true", 0),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
				},
			},
			// Add variables in an update
			{
				Config: testAccVariableSetWithLiteralVariablesConfig(name, "new-val1", "new-val2"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ctrlplane_variable_set.test",
						tfjsonpath.New("variables"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
		},
	})
}

func testAccVariableSetResourceConfig(name, description, selector string, priority int) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_variable_set" "test" {
  name        = %q
  description = %q
  selector    = %q
  priority    = %d
}
`, testAccProviderConfig(), name, description, selector, priority)
}

func testAccVariableSetWithLiteralVariablesConfig(name, val1, val2 string) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_variable_set" "test" {
  name        = %q
  description = "Variable set with literal variables"
  selector    = "true"
  priority    = 1

  variables = [
    {
      key   = "VAR_ONE"
      value = %q
    },
    {
      key   = "VAR_TWO"
      value = %q
    },
  ]
}
`, testAccProviderConfig(), name, val1, val2)
}

func testAccVariableSetWithReferenceVariableConfig(name string) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_variable_set" "test" {
  name        = %q
  description = "Variable set with reference variable"
  selector    = "true"
  priority    = 1

  variables = [
    {
      key = "REGION"
      reference_value = {
        reference = "resource"
        path      = ["metadata", "region"]
      }
    },
  ]
}
`, testAccProviderConfig(), name)
}

func testAccVariableSetDefaultPriorityConfig(name string) string {
	return fmt.Sprintf(`
%s
resource "ctrlplane_variable_set" "test" {
  name        = %q
  description = "Default priority test"
  selector    = "true"
}
`, testAccProviderConfig(), name)
}
