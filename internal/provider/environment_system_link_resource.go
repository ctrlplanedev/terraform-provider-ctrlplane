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

var _ resource.Resource = &EnvironmentSystemLinkResource{}
var _ resource.ResourceWithImportState = &EnvironmentSystemLinkResource{}
var _ resource.ResourceWithConfigure = &EnvironmentSystemLinkResource{}

func NewEnvironmentSystemLinkResource() resource.Resource {
	return &EnvironmentSystemLinkResource{}
}

type EnvironmentSystemLinkResource struct {
	workspace *api.WorkspaceClient
}

type EnvironmentSystemLinkResourceModel struct {
	ID            types.String `tfsdk:"id"`
	SystemID      types.String `tfsdk:"system_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
}

func (r *EnvironmentSystemLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_system_link"
}

func (r *EnvironmentSystemLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format: system_id/environment_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("system_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *EnvironmentSystemLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EnvironmentSystemLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Links an environment to a system in Ctrlplane. An environment can be linked to multiple systems.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier in the format system_id/environment_id",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"system_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the system to link the environment to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the environment to link",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *EnvironmentSystemLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EnvironmentSystemLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := r.workspace.ID.String()
	systemID := data.SystemID.ValueString()
	environmentID := data.EnvironmentID.ValueString()

	linkResp, err := r.workspace.Client.LinkEnvironmentToSystemWithResponse(
		ctx, workspaceID, systemID, environmentID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to link environment to system", err.Error())
		return
	}

	if linkResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to link environment to system", formatResponseError(linkResp.StatusCode(), linkResp.Body))
		return
	}

	data.ID = types.StringValue(systemID + "/" + environmentID)

	err = waitForResource(ctx, func() (bool, error) {
		envResp, err := r.workspace.Client.GetEnvironmentWithResponse(ctx, workspaceID, environmentID)
		if err != nil {
			return false, err
		}
		if envResp.StatusCode() != http.StatusOK || envResp.JSON200 == nil {
			return false, nil
		}
		for _, sid := range envResp.JSON200.SystemIds {
			if sid == systemID {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to link environment to system", fmt.Sprintf("Link not confirmed after creation: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvironmentSystemLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentSystemLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := r.workspace.ID.String()
	environmentID := data.EnvironmentID.ValueString()
	systemID := data.SystemID.ValueString()

	envResp, err := r.workspace.Client.GetEnvironmentWithResponse(ctx, workspaceID, environmentID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read environment system link", err.Error())
		return
	}

	switch envResp.StatusCode() {
	case http.StatusOK:
		if envResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read environment system link", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Failed to read environment system link", formatResponseError(envResp.StatusCode(), envResp.Body))
		return
	}

	found := false
	for _, sid := range envResp.JSON200.SystemIds {
		if sid == systemID {
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(systemID + "/" + environmentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvironmentSystemLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Environment system links cannot be updated in-place. Changing system_id or environment_id requires resource replacement.",
	)
}

func (r *EnvironmentSystemLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentSystemLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := r.workspace.ID.String()
	systemID := data.SystemID.ValueString()
	environmentID := data.EnvironmentID.ValueString()

	unlinkResp, err := r.workspace.Client.UnlinkEnvironmentFromSystemWithResponse(
		ctx, workspaceID, systemID, environmentID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to unlink environment from system", err.Error())
		return
	}

	switch unlinkResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusNotFound:
		return
	default:
		resp.Diagnostics.AddError("Failed to unlink environment from system", formatResponseError(unlinkResp.StatusCode(), unlinkResp.Body))
	}
}
