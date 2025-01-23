package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &systemResource{}
	_ resource.ResourceWithConfigure = &systemResource{}
)

// NewSystemResource is a helper function to simplify the provider implementation.
func NewSystemResource() resource.Resource {
	return &systemResource{}
}

// systemResource is the resource implementation.
type systemResource struct {
	client *client.Client
}

func (r *systemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *systemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system"
}

// Schema defines the schema for the resource.
func (r *systemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"slug": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"workspace_id": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

type systemResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	Description types.String `tfsdk:"description"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
}

func getDescription(description *string) *string {
	if description != nil && *description == "" {
		return nil
	}
	return description
}

// Create creates the resource and sets the initial Terraform state.
func (r *systemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceId, err := uuid.Parse(plan.WorkspaceId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid workspace ID", fmt.Sprintf("Invalid workspace ID: %s", err))
		return
	}

	clientResp, err := r.client.CreateSystem(ctx, client.CreateSystemJSONRequestBody{
		Name:        plan.Name.ValueString(),
		Slug:        plan.Slug.ValueString(),
		Description: getDescription(plan.Description.ValueStringPointer()),
		WorkspaceId: workspaceId,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create system", fmt.Sprintf("Failed to create system: %s", err))
		return
	}

	var result struct {
		System client.System `json:"system"`
	}

	if err := json.NewDecoder(clientResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to unmarshal system", fmt.Sprintf("Failed to unmarshal system: %s", err))
		return
	}

	plan.Id = types.StringValue(result.System.Id.String())
	plan.Name = types.StringValue(result.System.Name)
	plan.Slug = types.StringValue(result.System.Slug)
	description := getDescription(result.System.Description)
	if description != nil {
		plan.Description = types.StringValue(*description)
	}
	plan.WorkspaceId = types.StringValue(result.System.WorkspaceId.String())

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *systemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientResp, err := r.client.GetSystem(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read system", fmt.Sprintf("Failed to read system: %s", err))
		return
	}

	var result struct {
		System client.System `json:"system"`
	}

	if err := json.NewDecoder(clientResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to unmarshal system", fmt.Sprintf("Failed to unmarshal system: %s", err))
		return
	}

	state.Id = types.StringValue(result.System.Id.String())
	state.Name = types.StringValue(result.System.Name)
	state.Slug = types.StringValue(result.System.Slug)
	description := getDescription(result.System.Description)
	if description != nil {
		state.Description = types.StringValue(*description)
	}
	state.WorkspaceId = types.StringValue(result.System.WorkspaceId.String())

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *systemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state systemResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan systemResourceModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceId, err := uuid.Parse(plan.WorkspaceId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid workspace ID", fmt.Sprintf("Invalid workspace ID: %s", err))
		return
	}
	workspaceIdString := workspaceId.String()

	systemId := state.Id.ValueString()

	clientResp, err := r.client.UpdateSystem(ctx, systemId, client.UpdateSystemJSONRequestBody{
		Name:        plan.Name.ValueStringPointer(),
		Slug:        plan.Slug.ValueStringPointer(),
		Description: getDescription(plan.Description.ValueStringPointer()),
		WorkspaceId: &workspaceIdString,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update system", fmt.Sprintf("Failed to update system: %s", err))
		return
	}

	var result struct {
		System client.System `json:"system"`
	}

	if err := json.NewDecoder(clientResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to unmarshal system", fmt.Sprintf("Failed to unmarshal system: %s", err))
		return
	}

	plan.Id = types.StringValue(result.System.Id.String())
	plan.Name = types.StringValue(result.System.Name)
	plan.Slug = types.StringValue(result.System.Slug)
	description := getDescription(result.System.Description)
	if description != nil {
		plan.Description = types.StringValue(*description)
	}
	plan.WorkspaceId = types.StringValue(result.System.WorkspaceId.String())

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *systemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientResp, err := r.client.DeleteSystem(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete system", fmt.Sprintf("Failed to delete system: %s", err))
		return
	}

	if clientResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Failed to delete system", fmt.Sprintf("Failed to delete system: %s", clientResp.Status))
		return
	}
}
