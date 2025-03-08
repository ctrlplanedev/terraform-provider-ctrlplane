// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &systemResource{}
	_ resource.ResourceWithConfigure = &systemResource{}
	_ resource.ResourceWithImportState = &systemResource{}
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
		Description:         "Systems in Ctrlplane are the highest level of organizational units, designed to group related deployments and environments together.",
		MarkdownDescription: "Systems in Ctrlplane are the highest level of organizational units, designed to group related deployments and environments together.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "The ID of the system",
				MarkdownDescription: "The ID of the system",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "The name of the system",
				MarkdownDescription: "The name of the system",
			},
			"slug": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The slug of the system (must be unique to the workspace). If not provided, it will be generated from the name.",
				MarkdownDescription: "The slug of the system (must be unique to the workspace). If not provided, it will be generated from the name.",
				Validators: []validator.String{
					// Add a custom validator for the slug format
					SlugValidator{},
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Description:         "The description of the system",
				MarkdownDescription: "The description of the system",
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
	// If description is nil or an empty string, return nil
	// This ensures consistent handling of "no description" across the API
	if description == nil || (description != nil && *description == "") {
		return nil
	}
	return description
}

func setSystemResourceData(plan *systemResourceModel, system interface{}) {
	var id uuid.UUID
	var name, slug string
	var description *string

	// Handle different types of system objects
	switch s := system.(type) {
	case *client.System:
		id = s.Id
		name = s.Name
		slug = s.Slug
		description = s.Description
	case *struct {
		Deployments  *[]client.Deployment "json:\"deployments,omitempty\""
		Description  *string "json:\"description,omitempty\""
		Environments *[]client.Environment "json:\"environments,omitempty\""
		Id           uuid.UUID "json:\"id\""
		Name         string "json:\"name\""
		Slug         string "json:\"slug\""
		WorkspaceId  uuid.UUID "json:\"workspaceId\""
	}:
		id = s.Id
		name = s.Name
		slug = s.Slug
		description = s.Description
	}

	plan.Id = types.StringValue(id.String())
	plan.Name = types.StringValue(name)
	plan.Slug = types.StringValue(slug)
	
	// Handle description field properly
	if description == nil || (description != nil && *description == "") {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(*description)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *systemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data systemResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If slug is not provided, generate it from the name
	if data.Slug.IsNull() {
		slug := Slugify(data.Name.ValueString())
		data.Slug = types.StringValue(slug)
	}

	// Check if the slug exceeds the maximum length
	if err := ValidateSlugLength(data.Slug.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("slug"),
			"Invalid Slug Length",
			err.Error(),
		)
		return
	}

	// Create new system
	system, err := r.client.CreateSystemWithResponse(ctx, client.CreateSystemJSONRequestBody{
		Name:        data.Name.ValueString(),
		Slug:        data.Slug.ValueString(),
		Description: getDescription(data.Description.ValueStringPointer()),
		WorkspaceId: r.workspace,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create system",
			fmt.Sprintf("Failed to create system with slug '%s': %s", data.Slug.ValueString(), err),
		)
		return
	}

	// Handle error responses
	if system.StatusCode() == http.StatusBadRequest {
		errorMsg := "Bad Request"
		if system.JSON400 != nil && system.JSON400.Error != nil && len(*system.JSON400.Error) > 0 {
			// Extract the first error message
			firstError := (*system.JSON400.Error)[0]
			errorMsg = firstError.Message
			
			// Check for specific error messages
			if strings.Contains(strings.ToLower(errorMsg), "slug must not exceed") {
				resp.Diagnostics.AddAttributeError(
					path.Root("slug"),
					"Invalid Slug Length",
					errorMsg,
				)
				return
			}
		}

		// Check if the error is due to a duplicate slug
		if strings.Contains(strings.ToLower(errorMsg), "already exists") {
			resp.Diagnostics.AddError(
				"System already exists",
				fmt.Sprintf("A system with slug '%s' already exists. Please use a different slug.", data.Slug.ValueString()),
			)
		} else {
			resp.Diagnostics.AddError(
				"Failed to create system",
				fmt.Sprintf("Failed to create system: %s", errorMsg),
			)
		}
		return
	} else if system.StatusCode() == http.StatusInternalServerError {
		errorMsg := "Internal Server Error"
		if system.JSON500 != nil && system.JSON500.Error != nil {
			errorMsg = *system.JSON500.Error
		}

		resp.Diagnostics.AddError(
			"Failed to create system",
			fmt.Sprintf("An internal server error occurred while creating system with slug '%s'. This might be because a system with this slug already exists. Try using a different slug. Error: %s",
				data.Slug.ValueString(), errorMsg),
		)
		return
	} else if system.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError(
			"Failed to create system",
			fmt.Sprintf("Failed to create system with slug '%s'. Server returned status: %s",
				data.Slug.ValueString(), system.Status()),
		)
		return
	}
	setSystemResourceData(&data, system.JSON201)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

	// Save the current description value before updating the state
	originalDescription := state.Description
	
	// Update the state with the response
	setSystemResourceData(&state, system.JSON200)
	
	// Always use the original description value to maintain consistency
	// This is necessary because the API might return a different value than what was in the state
	state.Description = originalDescription

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *systemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data systemResourceModel
	var state systemResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	// Also read the current state to ensure we have the ID
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check if the slug exceeds the maximum length
	if !data.Slug.IsNull() && len(data.Slug.ValueString()) > 0 {
		if err := ValidateSlugLength(data.Slug.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("slug"),
				"Invalid Slug Length",
				err.Error(),
			)
			return
		}
	}

	// Use the ID from the state, not from the plan
	systemID := state.Id.ValueString()
	
	// Check if ID is empty
	if systemID == "" {
		resp.Diagnostics.AddError(
			"Missing Resource ID",
			"Cannot update system: resource ID is empty or not set",
		)
		return
	}

	// Update system
	system, err := r.client.UpdateSystemWithResponse(ctx, uuid.MustParse(systemID), client.UpdateSystemJSONRequestBody{
		Name:        data.Name.ValueStringPointer(),
		Slug:        data.Slug.ValueStringPointer(),
		Description: data.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update system",
			fmt.Sprintf("Failed to update system with ID '%s': %s", systemID, err),
		)
		return
	}

	// Handle error responses
	if system.StatusCode() == http.StatusBadRequest {
		resp.Diagnostics.AddError(
			"Failed to update system",
			"Bad Request",
		)
		return
	} else if system.StatusCode() == http.StatusInternalServerError {
		resp.Diagnostics.AddError(
			"Failed to update system",
			"Internal Server Error",
		)
		return
	} else if system.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddError(
			"System not found",
			fmt.Sprintf("System with ID '%s' not found", systemID),
		)
		return
	} else if system.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to update system",
			fmt.Sprintf("Failed to update system with ID '%s'. Server returned status: %s", systemID, system.Status()),
		)
		return
	}

	// Update resource state with updated items
	setSystemResourceData(&data, system.JSON200)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

// ImportState imports an existing system into Terraform state.
func (r *systemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is expected to be the system ID
	systemId := req.ID

	// Validate the ID is a valid UUID
	_, err := uuid.Parse(systemId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid System ID",
			fmt.Sprintf("The provided ID %q is not a valid UUID: %s", systemId, err),
		)
		return
	}

	// Set the ID in the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), systemId)...)
}
