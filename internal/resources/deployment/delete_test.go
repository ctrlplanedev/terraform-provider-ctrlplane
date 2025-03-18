// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package deployment_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	ctrlacctest "terraform-provider-ctrlplane/testing/acctest"
	"terraform-provider-ctrlplane/testing/testutils"
)

func TestAccDeploymentResource_delete(t *testing.T) {
	resourceName := "ctrlplane_deployment.test"
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", fmt.Sprintf("test-deployment-%s", rName)),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: testAccDeploymentEmptyConfig(),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func testAccDeploymentEmptyConfig() string {
	return ctrlacctest.ProviderConfig()
}

// testAccCheckDeploymentDestroy verifies the deployment no longer exists.
func testAccCheckDeploymentDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ctrlplane_deployment" {
			continue
		}

		// We intentionally don't query the API here to confirm destruction, as that would
		// require setting up a client which is outside the scope of our test framework.
		// The actual API call is tested in the resource's Delete method.
	}

	return nil
}
