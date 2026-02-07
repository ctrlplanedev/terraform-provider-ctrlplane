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

var _ resource.Resource = &SystemResource{}

var _ resource.ResourceWithImportState = &SystemResource{}
var _ resource.ResourceWithConfigure = &SystemResource{}

func NewSystemResource() resource.Resource {
	return &SystemResource{}
}

type SystemResource struct {
	workspace *api.WorkspaceClient
}

// ImportState implements resource.ResourceWithImportState.
func (r *SystemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Configure implements resource.ResourceWithConfigure.
func (r *SystemResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create implements resource.Resource.
func (r *SystemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SystemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := api.RequestSystemCreationJSONRequestBody{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
		Metadata:    stringMapPointer(data.Metadata),
	}
	workspaceId := r.workspace.ID
	system, err := r.workspace.Client.RequestSystemCreationWithResponse(ctx, workspaceId.String(), requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create system", err.Error())
		return
	}

	if system.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to create system", formatResponseError(system.StatusCode(), system.Body))
		return
	}

	if system.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to create system", "Empty response from server")
		return
	}

	systemId := system.JSON202.Id
	if systemId == "" {
		resp.Diagnostics.AddError("Failed to create system", "Empty system ID in response")
		return
	}

	data.ID = types.StringValue(systemId)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Delete implements resource.Resource.
func (r *SystemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SystemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientResp, err := r.workspace.Client.RequestSystemDeletionWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete system", fmt.Sprintf("Failed to delete system: %s", err.Error()))
		return
	}

	switch clientResp.StatusCode() {
	case http.StatusAccepted:
		return
	case http.StatusBadRequest:
		if clientResp.JSON400 != nil && clientResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to delete system", fmt.Sprintf("Bad request: %s", *clientResp.JSON400.Error))
			return
		}
	case http.StatusNotFound:
		if clientResp.JSON404 != nil && clientResp.JSON404.Error != nil {
			resp.Diagnostics.AddError("Failed to delete system", fmt.Sprintf("Not found: %s", *clientResp.JSON404.Error))
			return
		}
	}

	if clientResp.StatusCode() != http.StatusAccepted || clientResp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError("Failed to delete system", formatResponseError(clientResp.StatusCode(), clientResp.Body))
		return
	}
}

// Read implements resource.Resource.
func (r *SystemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SystemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	system, err := r.workspace.Client.GetSystemWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read system",
			fmt.Sprintf("Failed to read system with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	switch system.StatusCode() {
	case http.StatusOK:
		if system.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read system", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	case http.StatusBadRequest:
		if system.JSON400 != nil && system.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to read system", fmt.Sprintf("Bad request: %s", *system.JSON400.Error))
			return
		}
		resp.Diagnostics.AddError("Failed to read system", "Bad request")
		return
	}

	if system.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Failed to read system", formatResponseError(system.StatusCode(), system.Body))
		return
	}

	if system.JSON200.Id == "" {
		resp.Diagnostics.AddError("Failed to read system", "Empty system ID in response")
		return
	}
	if system.JSON200.Name == "" {
		resp.Diagnostics.AddError("Failed to read system", "Empty system name in response")
		return
	}

	data.Name = types.StringValue(system.JSON200.Name)
	data.Description = descriptionValue(system.JSON200.Description)
	data.Metadata = stringMapValue(system.JSON200.Metadata)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Schema implements resource.Resource.
func (r *SystemResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the system",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the system",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the system",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Description: "The metadata of the system",
				ElementType: types.StringType,
			},
		},
	}
}

// Update implements resource.Resource.
func (r *SystemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SystemResourceModel
	var state SystemResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID since it is computed and not known from the plan.
	data.ID = state.ID

	requestBody := api.RequestSystemUpdateJSONRequestBody{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
		Metadata:    stringMapPointer(data.Metadata),
	}
	system, err := r.workspace.Client.RequestSystemUpdateWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(), requestBody,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update system",
			fmt.Sprintf("Failed to update system with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	if system.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update system", formatResponseError(system.StatusCode(), system.Body))
		return
	}

	if system.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to update system", "Empty response from server")
		return
	}

	systemId := system.JSON202.Id
	data.ID = types.StringValue(systemId)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *SystemResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system"
}

func formatResponseError(statusCode int, body []byte) string {
	if len(body) > 0 {
		return fmt.Sprintf("Status %d: %s", statusCode, strings.TrimSpace(string(body)))
	}

	if statusCode == 0 {
		return "Missing response status from server"
	}

	return fmt.Sprintf("Status %d: %s", statusCode, http.StatusText(statusCode))
}

type SystemResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Metadata    types.Map    `tfsdk:"metadata"`
}
