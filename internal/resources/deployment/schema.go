// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package deployment

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func GetDeploymentDataSourceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Fetch a deployment resource by ID or other attributes",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Deployment identifier",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Name of the deployment",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the deployment",
			},
			"system_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "System ID this deployment belongs to",
			},
			"slug": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Slug identifier for the deployment",
			},
			"job_agent_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Job agent ID used for this deployment",
			},
			"job_agent_config": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Job agent configuration",
			},
			"retry_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of retry attempts",
			},
			"timeout": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Timeout in seconds",
			},
			"resource_filter": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Resource filter configuration",
			},
		},
	}
}
