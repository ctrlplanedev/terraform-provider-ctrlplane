// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &systemResource{}
	_ resource.ResourceWithConfigure = &systemResource{}
)

func NewSystemResource() resource.Resource {
	return &systemResource{}
}

type systemResource struct {
	client    *client.ClientWithResponses
	workspace uuid.UUID
}

func (r *systemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataSourceModel, ok := req.ProviderData.(*DataSourceModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = dataSourceModel.Client
	r.workspace = dataSourceModel.Workspace
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
		},
	}
}

type systemResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	Description types.String `tfsdk:"description"`
}

func getDescription(description *string) *string {
	if description != nil && *description == "" {
		return nil
	}
	return description
}

func setSystemResourceData(plan *systemResourceModel, system *client.System) {
	plan.Id = types.StringValue(system.Id.String())
	plan.Name = types.StringValue(system.Name)
	plan.Slug = types.StringValue(system.Slug)
	description := getDescription(system.Description)
	if description != nil {
		plan.Description = types.StringValue(*description)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *systemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	system, err := r.client.CreateSystemWithResponse(ctx, client.CreateSystemJSONRequestBody{
		Name:        plan.Name.ValueString(),
		Slug:        plan.Slug.ValueString(),
		Description: getDescription(plan.Description.ValueStringPointer()),
		WorkspaceId: r.workspace,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create system", fmt.Sprintf("Failed to create system: %s", err))
		return
	}

	if system.JSON201 == nil {
		if system.JSON400 != nil && system.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to create system", fmt.Sprintf("Failed to create system: %s", *system.JSON400.Error))
			return
		}

		if system.JSON500 != nil && system.JSON500.Error != nil {
			resp.Diagnostics.AddError("Failed to create system", fmt.Sprintf("Failed to create system: %s", *system.JSON500.Error))
			return
		}

		resp.Diagnostics.AddError("Failed to create system", fmt.Sprintf("Failed to create system: %s", system.Status()))
		return
	}
	setSystemResourceData(&plan, system.JSON201)

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

	systemId, err := uuid.Parse(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid system ID", fmt.Sprintf("Invalid system ID: %s", err))
		return
	}

	system, err := r.client.GetSystemWithResponse(ctx, systemId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read system", fmt.Sprintf("Failed to read system: %s", err))
		return
	}

	if system.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to read system", fmt.Sprintf("Failed to read system: %s", system.Status()))
		return
	}

	state.Id = types.StringValue(system.JSON200.Id.String())
	state.Name = types.StringValue(system.JSON200.Name)
	state.Slug = types.StringValue(system.JSON200.Slug)
	description := getDescription(system.JSON200.Description)
	if description != nil {
		state.Description = types.StringValue(*description)
	}

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

	systemId, err := uuid.Parse(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid system ID", fmt.Sprintf("Invalid system ID: %s", err))
		return
	}

	system, err := r.client.UpdateSystemWithResponse(ctx, systemId, client.UpdateSystemJSONRequestBody{
		Name:        plan.Name.ValueStringPointer(),
		Slug:        plan.Slug.ValueStringPointer(),
		Description: getDescription(plan.Description.ValueStringPointer()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update system", fmt.Sprintf("Failed to update system: %s", err))
		return
	}

	if system.JSON200 == nil {
		if system.JSON404 != nil && system.JSON404.Error != nil {
			resp.Diagnostics.AddError("Failed to update system", fmt.Sprintf("Failed to update system: %s", *system.JSON404.Error))
			return
		}

		if system.JSON500 != nil && system.JSON500.Error != nil {
			resp.Diagnostics.AddError("Failed to update system", fmt.Sprintf("Failed to update system: %s", *system.JSON500.Error))
			return
		}

		resp.Diagnostics.AddError("Failed to update system", fmt.Sprintf("Failed to update system: %s", system.Status()))
		return
	}

	setSystemResourceData(&plan, system.JSON200)

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

	clientResp, err := r.client.DeleteSystemWithResponse(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete system", fmt.Sprintf("Failed to delete system: %s", err))
		return
	}

	if clientResp.JSON404 != nil && clientResp.JSON404.Error != nil {
		resp.Diagnostics.AddError("Failed to delete system", fmt.Sprintf("Failed to delete system: %s", *clientResp.JSON404.Error))
		return
	}

	if clientResp.JSON500 != nil && clientResp.JSON500.Error != nil {
		resp.Diagnostics.AddError("Failed to delete system", fmt.Sprintf("Failed to delete system: %s", *clientResp.JSON500.Error))
		return
	}

	if clientResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to delete system", fmt.Sprintf("Failed to delete system: %s", clientResp.Status()))
		return
	}
}
