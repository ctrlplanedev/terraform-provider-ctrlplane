// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package deployment

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DeploymentModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	SystemID       types.String `tfsdk:"system_id"`
	Slug           types.String `tfsdk:"slug"`
	JobAgentID     types.String `tfsdk:"job_agent_id"`
	JobAgentConfig types.Map    `tfsdk:"job_agent_config"`
	RetryCount     types.Int64  `tfsdk:"retry_count"`
	Timeout        types.Int64  `tfsdk:"timeout"`
	ResourceFilter types.Map    `tfsdk:"resource_filter"`
}

type DeploymentResourceModel = DeploymentModel
type DeploymentDataSourceModel = DeploymentModel
