// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package deployment_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	ctrlacctest "terraform-provider-ctrlplane/testing/acctest"
	"terraform-provider-ctrlplane/testing/testutils"
)

func TestAccDeploymentResource_update(t *testing.T) {
	resourceName := "ctrlplane_deployment.test"
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	initialName := fmt.Sprintf("test-deployment-%s", rName)
	updatedName := fmt.Sprintf("updated-deployment-%s", rName)
	initialDescription := "Initial deployment description"
	updatedDescription := "Updated deployment description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfigWithDesc(rName, initialName, initialDescription),
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", initialName),
					resource.TestCheckResourceAttr(resourceName, "description", initialDescription),
				),
			},
			{
				Config: testAccDeploymentConfigWithDesc(rName, updatedName, updatedDescription),
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "description", updatedDescription),
				),
			},
		},
	})
}

func testAccDeploymentConfigWithDesc(rName, name, description string) string {
	return ctrlacctest.ProviderConfig() + fmt.Sprintf(`
	# First create a system
	resource "ctrlplane_system" "test" {
		name        = "test-system-%[1]s"
		description = "Test system for deployment tests"
		slug        = "test-system-%[1]s"
	}
	
	# Create the deployment
	resource "ctrlplane_deployment" "test" {
		name            = %[2]q
		description     = %[3]q
		system_id       = ctrlplane_system.test.id
		slug            = "deployment-%[1]s"
		job_agent_config = {
			"key1" = "value1"
			"key2" = "value2"
		}
		retry_count     = 3
		timeout         = 300
	}
	`, rName, name, description)
}
