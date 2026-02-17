// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DeploymentSystemLinkResource{}
var _ resource.ResourceWithImportState = &DeploymentSystemLinkResource{}
var _ resource.ResourceWithConfigure = &DeploymentSystemLinkResource{}

func NewDeploymentSystemLinkResource() resource.Resource {
	return &DeploymentSystemLinkResource{}
}

type DeploymentSystemLinkResource struct {
	workspace *api.WorkspaceClient
}

type DeploymentSystemLinkResourceModel struct {
	ID           types.String `tfsdk:"id"`
	SystemID     types.String `tfsdk:"system_id"`
	DeploymentID types.String `tfsdk:"deployment_id"`
}

func (r *DeploymentSystemLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_system_link"
}

func (r *DeploymentSystemLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format: system_id/deployment_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("system_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deployment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *DeploymentSystemLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	workspace, ok := req.ProviderData.(*api.WorkspaceClient)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "The provider data is not a *api.WorkspaceClient")
		return
	}

	r.workspace = workspace
}

func (r *DeploymentSystemLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Links a deployment to a system in Ctrlplane. A deployment can be linked to multiple systems.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier in the format system_id/deployment_id",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"system_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the system to link the deployment to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the deployment to link",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *DeploymentSystemLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentSystemLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := r.workspace.ID.String()
	systemID := data.SystemID.ValueString()
	deploymentID := data.DeploymentID.ValueString()

	linkResp, err := r.workspace.Client.LinkDeploymentToSystemWithResponse(
		ctx, workspaceID, systemID, deploymentID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to link deployment to system", err.Error())
		return
	}

	if linkResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to link deployment to system", formatResponseError(linkResp.StatusCode(), linkResp.Body))
		return
	}

	data.ID = types.StringValue(systemID + "/" + deploymentID)

	err = waitForResource(ctx, func() (bool, error) {
		depResp, err := r.workspace.Client.GetDeploymentWithResponse(ctx, workspaceID, deploymentID)
		if err != nil {
			return false, err
		}
		if depResp.StatusCode() != http.StatusOK || depResp.JSON200 == nil {
			return false, nil
		}
		for _, sid := range depResp.JSON200.Deployment.SystemIds {
			if sid == systemID {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to link deployment to system", fmt.Sprintf("Link not confirmed after creation: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentSystemLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentSystemLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := r.workspace.ID.String()
	deploymentID := data.DeploymentID.ValueString()
	systemID := data.SystemID.ValueString()

	depResp, err := r.workspace.Client.GetDeploymentWithResponse(ctx, workspaceID, deploymentID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read deployment system link", err.Error())
		return
	}

	switch depResp.StatusCode() {
	case http.StatusOK:
		if depResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read deployment system link", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Failed to read deployment system link", formatResponseError(depResp.StatusCode(), depResp.Body))
		return
	}

	found := false
	for _, sid := range depResp.JSON200.Deployment.SystemIds {
		if sid == systemID {
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(systemID + "/" + deploymentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentSystemLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Deployment system links cannot be updated in-place. Changing system_id or deployment_id requires resource replacement.",
	)
}

func (r *DeploymentSystemLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentSystemLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := r.workspace.ID.String()
	systemID := data.SystemID.ValueString()
	deploymentID := data.DeploymentID.ValueString()

	unlinkResp, err := r.workspace.Client.UnlinkDeploymentFromSystemWithResponse(
		ctx, workspaceID, systemID, deploymentID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to unlink deployment from system", err.Error())
		return
	}

	switch unlinkResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusNotFound:
		return
	default:
		resp.Diagnostics.AddError("Failed to unlink deployment from system", formatResponseError(unlinkResp.StatusCode(), unlinkResp.Body))
	}
}
