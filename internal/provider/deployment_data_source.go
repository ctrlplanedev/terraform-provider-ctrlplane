// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DeploymentDataSource{}
var _ datasource.DataSourceWithConfigure = &DeploymentDataSource{}

func NewDeploymentDataSource() datasource.DataSource {
	return &DeploymentDataSource{}
}

type DeploymentDataSource struct {
	workspace *api.WorkspaceClient
}

type DeploymentDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Slug             types.String `tfsdk:"slug"`
	Description      types.String `tfsdk:"description"`
	ResourceSelector types.String `tfsdk:"resource_selector"`
	JobAgentSelector types.String `tfsdk:"job_agent_selector"`
	Metadata         types.Map    `tfsdk:"metadata"`
}

func (d *DeploymentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (d *DeploymentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch an existing deployment by name within the configured workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the deployment",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the deployment to look up",
			},
			"slug": schema.StringAttribute{
				Computed:    true,
				Description: "The slug of the deployment",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the deployment",
			},
			"resource_selector": schema.StringAttribute{
				Computed:    true,
				Description: "CEL expression used to select resources",
			},
			"job_agent_selector": schema.StringAttribute{
				Computed:    true,
				Description: "CEL expression used to match a job agent",
			},
			"metadata": schema.MapAttribute{
				Computed:    true,
				Description: "The metadata of the deployment",
				ElementType: types.StringType,
			},
		},
	}
}

func (d *DeploymentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	workspace, ok := req.ProviderData.(*api.WorkspaceClient)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "The provider data is not a *api.WorkspaceClient")
		return
	}

	d.workspace = workspace
}

func (d *DeploymentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeploymentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	depResp, err := d.workspace.Client.GetDeploymentByNameWithResponse(
		ctx, d.workspace.ID.String(), data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read deployment",
			fmt.Sprintf("Failed to read deployment with name '%s': %s", data.Name.ValueString(), err.Error()),
		)
		return
	}

	if depResp.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddError(
			"Deployment not found",
			fmt.Sprintf("No deployment with name '%s' in workspace '%s'", data.Name.ValueString(), d.workspace.ID.String()),
		)
		return
	}

	if depResp.StatusCode() != http.StatusOK || depResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read deployment", formatResponseError(depResp.StatusCode(), depResp.Body))
		return
	}

	dep := depResp.JSON200.Deployment
	data.ID = types.StringValue(dep.Id)
	data.Name = types.StringValue(dep.Name)
	data.Slug = types.StringValue(dep.Slug)
	data.Description = descriptionValue(dep.Description)
	data.Metadata = stringMapValue(dep.Metadata)
	if dep.ResourceSelector != nil && *dep.ResourceSelector != "" {
		data.ResourceSelector = types.StringValue(*dep.ResourceSelector)
	} else {
		data.ResourceSelector = types.StringNull()
	}
	if dep.JobAgentSelector != "" {
		data.JobAgentSelector = types.StringValue(dep.JobAgentSelector)
	} else {
		data.JobAgentSelector = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
