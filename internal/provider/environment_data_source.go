// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &EnvironmentDataSource{}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

// EnvironmentDataSource defines the data source implementation.
type EnvironmentDataSource struct {
	client    *client.ClientWithResponses
	workspace uuid.UUID
}

// EnvironmentDataSourceModel describes the data source data model.
type EnvironmentDataSourceModel struct {
	ID             types.String      `tfsdk:"id"`
	Name           types.String      `tfsdk:"name"`
	Description    types.String      `tfsdk:"description"`
	SystemID       types.String      `tfsdk:"system_id"`
	PolicyID       types.String      `tfsdk:"policy_id"`
	Metadata       types.Map         `tfsdk:"metadata"`
	ResourceFilter *ResourceFilter   `tfsdk:"resource_filter"`
}

func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Environment data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Environment identifier",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the environment",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the environment",
			},
			"system_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "System ID the environment belongs to",
			},
			"policy_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Policy ID for the environment",
			},
			"metadata": schema.MapAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Metadata for the environment",
			},
			"resource_filter": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Resource filter for the environment",
				Attributes: map[string]schema.Attribute{
					"filter_type": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Type of resource filter",
					},
					"namespace": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Namespace for resource filter",
					},
				},
			},
		},
	}
}

func (d *EnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataSourceModel, ok := req.ProviderData.(*DataSourceModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *DataSourceModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = dataSourceModel.Client
	d.workspace = dataSourceModel.Workspace
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get environment from API
	getResp, err := d.client.GetEnvironmentWithResponse(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	if getResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read environment, got empty response")
		return
	}

	// Map response body to schema
	data.Name = types.StringValue(getResp.JSON200.Name)
	if getResp.JSON200.Description != nil {
		data.Description = types.StringValue(*getResp.JSON200.Description)
	}
	data.SystemID = types.StringValue(getResp.JSON200.SystemId.String())
	if getResp.JSON200.PolicyId != nil {
		data.PolicyID = types.StringValue(getResp.JSON200.PolicyId.String())
	}

	if getResp.JSON200.Metadata != nil {
		metadata, diag := types.MapValueFrom(ctx, types.StringType, *getResp.JSON200.Metadata)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Metadata = metadata
	}

	if getResp.JSON200.ResourceFilter != nil {
		resourceFilter := *getResp.JSON200.ResourceFilter
		data.ResourceFilter = &ResourceFilter{
			FilterType: types.StringValue(resourceFilter["type"].(string)),
			Namespace:  types.StringValue(resourceFilter["namespace"].(string)),
		}
	} else {
		data.ResourceFilter = nil
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
