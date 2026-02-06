package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithConfigure = &EnvironmentResource{}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

type EnvironmentResource struct {
	workspace *api.WorkspaceClient
}

// Configure implements resource.ResourceWithConfigure.
func (r *EnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceId := r.workspace.ID
	envResp, err := r.workspace.Client.CreateEnvironmentWithResponse(ctx, workspaceId.String(), api.CreateEnvironmentJSONRequestBody{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
		SystemId:     data.SystemId.ValueString(),
		Metadata:    stringMapPointer(data.Metadata),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create environment", err.Error())
		return
	}

	if envResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to create environment", formatResponseError(envResp.StatusCode(), envResp.Body))
		return
	}

	if envResp.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to create environment", "Empty response from server")
		return
	}

	envId := envResp.JSON202.Id
	if envId == "" {
		resp.Diagnostics.AddError("Failed to create environment", "Empty environment ID in response")
		return
	}

	data.ID = types.StringValue(envId)
	data.WorkspaceId = types.StringValue(workspaceId.String())
	data.Description = descriptionValue(envResp.JSON202.Description)
	data.Metadata = stringMapValue(envResp.JSON202.Metadata)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Delete implements resource.Resource.
func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientResp, err := r.workspace.Client.DeleteEnvironmentWithResponse(ctx, data.WorkspaceId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete environment", fmt.Sprintf("Failed to delete environment: %s", err.Error()))
		return
	}

	if clientResp.StatusCode() != http.StatusAccepted || clientResp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError("Failed to delete environment", formatResponseError(clientResp.StatusCode(), clientResp.Body))
		return
	}
}

// Read implements resource.Resource.
func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID := data.WorkspaceId.ValueString()
	if data.WorkspaceId.IsNull() || workspaceID == "" {
		workspaceID = r.workspace.ID.String()
	}

	envResp, err := r.workspace.Client.GetEnvironmentWithResponse(ctx, workspaceID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read environment",
			fmt.Sprintf("Failed to read environment with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	switch envResp.StatusCode() {
	case http.StatusOK:
		if envResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read environment", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	case http.StatusBadRequest:
		if envResp.JSON400 != nil && envResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to read environment", fmt.Sprintf("Bad request: %s", *envResp.JSON400.Error))
			return
		}
		resp.Diagnostics.AddError("Failed to read environment", "Bad request")
		return
	}

	if envResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Failed to read environment", formatResponseError(envResp.StatusCode(), envResp.Body))
		return
	}

	if envResp.JSON200.Id == "" {
		resp.Diagnostics.AddError("Failed to read environment", "Empty environment ID in response")
		return
	}
	if envResp.JSON200.Name == "" {
		resp.Diagnostics.AddError("Failed to read environment", "Empty environment name in response")
		return
	}

	data.ID = types.StringValue(envResp.JSON200.Id)
	data.WorkspaceId = types.StringValue(workspaceID)
	data.Name = types.StringValue(envResp.JSON200.Name)
	data.Description = descriptionValue(envResp.JSON200.Description)
	data.SystemId = types.StringValue(envResp.JSON200.SystemId)
	data.Metadata = stringMapValue(envResp.JSON200.Metadata)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Schema implements resource.Resource.
func (r *EnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the environment",
				PlanModifiers: []planmodifier.String{},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the environment",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the environment",
			},
			"system_id": schema.StringAttribute{
				Required:    true,
				Description: "The system ID this environment belongs to",
			},
			"workspace_id": schema.StringAttribute{
				Computed:    true,
				Description: "The workspace ID the environment belongs to",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Description: "The metadata of the environment",
				ElementType: types.StringType,
			},
		},
	}
}

// Update implements resource.Resource.
func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envResp, err := r.workspace.Client.UpsertEnvironmentByIdWithResponse(ctx, data.WorkspaceId.ValueString(), data.ID.ValueString(), api.UpsertEnvironmentByIdJSONRequestBody{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
		SystemId:     data.SystemId.ValueString(),
		Metadata:    stringMapPointer(data.Metadata),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update environment",
			fmt.Sprintf("Failed to update environment with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	if envResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update environment", formatResponseError(envResp.StatusCode(), envResp.Body))
		return
	}

	if envResp.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to update environment", "Empty response from server")
		return
	}

	envId := envResp.JSON202.Id
	if envResp.JSON202.Name == "" {
		resp.Diagnostics.AddError("Failed to update environment", "Empty environment name in response")
		return
	}

	data.ID = types.StringValue(envId)
	data.WorkspaceId = types.StringValue(r.workspace.ID.String())
	data.Name = types.StringValue(envResp.JSON202.Name)
	data.Description = descriptionValue(envResp.JSON202.Description)
	data.SystemId = types.StringValue(envResp.JSON202.SystemId)
	data.Metadata = stringMapValue(envResp.JSON202.Metadata)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

type EnvironmentResourceModel struct {
	ID          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	Name        types.String `tfsdk:"name"`
	SystemId    types.String `tfsdk:"system_id"`
	Description types.String `tfsdk:"description"`
	Metadata    types.Map `tfsdk:"metadata"`
}
