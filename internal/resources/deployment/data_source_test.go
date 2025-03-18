// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package deployment_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	ctrlacctest "terraform-provider-ctrlplane/testing/acctest"
	"terraform-provider-ctrlplane/testing/testutils"
)

func TestAccDeploymentDataSource_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "ctrlplane_deployment.test"
	dataSourceName := "data.ctrlplane_deployment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckResourceExists(resourceName),
				),
			},
			{
				Config: testAccDeploymentDataSourceConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "description", resourceName, "description"),
					resource.TestCheckResourceAttrPair(dataSourceName, "system_id", resourceName, "system_id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "slug", resourceName, "slug"),
					resource.TestCheckResourceAttrPair(dataSourceName, "job_agent_id", resourceName, "job_agent_id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "retry_count", resourceName, "retry_count"),
					resource.TestCheckResourceAttrPair(dataSourceName, "timeout", resourceName, "timeout"),
				),
			},
		},
	})
}

func testAccDeploymentDataSourceConfig(rName string) string {
	return testAccDeploymentConfig(rName) + `
	data "ctrlplane_deployment" "test" {
		id = ctrlplane_deployment.test.id
	}
	`
}
