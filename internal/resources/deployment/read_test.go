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

func TestAccDeploymentResource_read(t *testing.T) {
	resourceName := "ctrlplane_deployment.test"
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", fmt.Sprintf("test-deployment-%s", rName)),
					resource.TestCheckResourceAttr(resourceName, "slug", fmt.Sprintf("test-deployment-%s", rName)),
					resource.TestCheckResourceAttrSet(resourceName, "system_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: testAccDeploymentConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", fmt.Sprintf("test-deployment-%s", rName)),
					resource.TestCheckResourceAttr(resourceName, "description", "Test deployment for read test"),
					resource.TestCheckResourceAttr(resourceName, "slug", fmt.Sprintf("test-deployment-%s", rName)),
					resource.TestCheckResourceAttrSet(resourceName, "system_id"),
					resource.TestCheckResourceAttr(resourceName, "job_agent_config.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "job_agent_config.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "retry_count", "3"),
					resource.TestCheckResourceAttr(resourceName, "timeout", "300"),
				),
			},
		},
	})
}

func testAccDeploymentConfigBasic(rName string) string {
	return ctrlacctest.ProviderConfig() + fmt.Sprintf(`
	# First create a system
	resource "ctrlplane_system" "test" {
		name        = "test-system-%[1]s"
		description = "Test system for deployment read test"
		slug        = "test-system-%[1]s"
	}
	
	# Create the deployment
	resource "ctrlplane_deployment" "test" {
		name            = "test-deployment-%[1]s"
		description     = "Test deployment for read test"
		system_id       = ctrlplane_system.test.id
		slug            = "test-deployment-%[1]s"
		job_agent_config = {
			"key1" = "value1"
			"key2" = "value2"
		}
		retry_count     = 3
		timeout         = 300
	}
	`, rName)
}
