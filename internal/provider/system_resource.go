// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &systemResource{}
	_ resource.ResourceWithConfigure   = &systemResource{}
	_ resource.ResourceWithImportState = &systemResource{}
)

func NewSystemResource() resource.Resource {
	return &systemResource{}
}

type systemResource struct {
	client    *client.ClientWithResponses
	workspace uuid.UUID
}

// Configure sets up the client and workspace using SystemModel.
func (r *systemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	systemModel, ok := req.ProviderData.(*DataSourceModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Configure Type",
			fmt.Sprintf("Expected *SystemModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = systemModel.Client
	r.workspace = systemModel.Workspace
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

// descriptionPtr returns a pointer to the string value if not null/empty.
func descriptionPtr(attr types.String) *string {
	if attr.IsNull() || attr.ValueString() == "" {
		return nil
	}
	return attr.ValueStringPointer()
}

// processSlug generates or validates the slug.
func processSlug(ctx context.Context, slugAttr types.String, name string) (string, error) {
	if slugAttr.IsNull() || slugAttr.ValueString() == "" {
		generated := Slugify(name)
		tflog.Info(ctx, "Generated slug from name", map[string]interface{}{
			"name": name,
			"slug": generated,
		})
		if err := ValidateSlugLength(generated); err != nil {
			return "", err
		}
		return generated, nil
	}

	value := slugAttr.ValueString()
	if err := ValidateSlugLength(value); err != nil {
		return "", err
	}
	pattern := `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	regex := regexp.MustCompile(pattern)
	tflog.Info(ctx, "Validating slug format", map[string]interface{}{
		"slug":    value,
		"pattern": pattern,
		"matches": regex.MatchString(value),
	})
	if !regex.MatchString(value) {
		return "", fmt.Errorf("slug must contain only lowercase alphanumeric characters and hyphens, and must start and end with an alphanumeric character")
	}
	return value, nil
}

// extractSystemAttributes extracts common attributes from different system types.
func extractSystemAttributes(system interface{}) (uuid.UUID, string, string, *string, error) {
	switch s := system.(type) {
	case *client.System:
		return s.Id, s.Name, s.Slug, s.Description, nil
	case *struct {
		Deployments  *[]client.Deployment `json:"deployments,omitempty"`
		Description  *string              `json:"description,omitempty"`
		Environments *[]client.Environment `json:"environments,omitempty"`
		Id           uuid.UUID            `json:"id"`
		Name         string               `json:"name"`
		Slug         string               `json:"slug"`
		WorkspaceId  uuid.UUID            `json:"workspaceId"`
	}:
		return s.Id, s.Name, s.Slug, s.Description, nil
	default:
		return uuid.Nil, "", "", nil, fmt.Errorf("unsupported system type: %T", system)
	}
}

// setSystemResourceData maps system attributes into the Terraform state.
func setSystemResourceData(plan *systemResourceModel, system interface{}) {
	id, name, slug, description, err := extractSystemAttributes(system)
	if err != nil {
		tflog.Error(context.Background(), "Failed to extract system attributes", map[string]interface{}{"error": err.Error()})
		return
	}

	plan.Id = types.StringValue(id.String())
	plan.Name = types.StringValue(name)
	plan.Slug = types.StringValue(slug)
	if description == nil || *description == "" {
		plan.Description = types.StringNull()
	} else {
		plan.Description = types.StringValue(*description)
	}
}

// Create creates the system and sets the initial Terraform state.
func (r *systemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data systemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newSlug, err := processSlug(ctx, data.Slug, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("slug"),
			"Invalid Slug",
			err.Error(),
		)
		return
	}
	data.Slug = types.StringValue(newSlug)
	descriptionValue := descriptionPtr(data.Description)

	system, err := r.client.CreateSystemWithResponse(ctx, client.CreateSystemJSONRequestBody{
		Name:        data.Name.ValueString(),
		Slug:        data.Slug.ValueString(),
		Description: descriptionValue,
		WorkspaceId: r.workspace,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create system",
			fmt.Sprintf("Failed to create system with slug '%s': %s", data.Slug.ValueString(), err),
		)
		return
	}

	if system.StatusCode() == http.StatusBadRequest {
		errorMsg := "Bad Request"
		if system.JSON400 != nil && system.JSON400.Error != nil && len(*system.JSON400.Error) > 0 {
			firstError := (*system.JSON400.Error)[0]
			errorMsg = firstError.Message
			tflog.Info(ctx, "BadRequest Error Details", map[string]interface{}{
				"error": *system.JSON400.Error,
			})
			if strings.Contains(strings.ToLower(errorMsg), "slug must not exceed") {
				resp.Diagnostics.AddAttributeError(
					path.Root("slug"),
					"Invalid Slug Length",
					errorMsg,
				)
				return
			}
			if strings.Contains(strings.ToLower(errorMsg), "slug format") ||
				strings.Contains(strings.ToLower(errorMsg), "invalid slug") ||
				strings.Contains(strings.ToLower(errorMsg), "must contain only") {
				resp.Diagnostics.AddAttributeError(
					path.Root("slug"),
					"Invalid Slug Format",
					errorMsg,
				)
				return
			}
		}
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

	originalDescription := state.Description
	setSystemResourceData(&state, system.JSON200)
	if originalDescription.IsNull() {
		state.Description = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update updates the system and sets the updated Terraform state.
func (r *systemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data systemResourceModel
	var state systemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Slug.IsNull() && data.Slug.ValueString() != "" {
		if err := ValidateSlugLength(data.Slug.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("slug"),
				"Invalid Slug Length",
				err.Error(),
			)
			return
		}
	}

	systemID := state.Id.ValueString()
	if systemID == "" {
		resp.Diagnostics.AddError(
			"Missing Resource ID",
			"Cannot update system: resource ID is empty or not set",
		)
		return
	}

	descriptionValue := descriptionPtr(data.Description)
	system, err := r.client.UpdateSystemWithResponse(ctx, uuid.MustParse(systemID), client.UpdateSystemJSONRequestBody{
		Name:        data.Name.ValueStringPointer(),
		Slug:        data.Slug.ValueStringPointer(),
		Description: descriptionValue,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update system",
			fmt.Sprintf("Failed to update system with ID '%s': %s", systemID, err),
		)
		return
	}

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

	setSystemResourceData(&data, system.JSON200)
	var planModel systemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if planModel.Description.IsNull() {
		data.Description = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the system and removes the Terraform state.
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
	systemId := req.ID
	_, err := uuid.Parse(systemId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid System ID",
			fmt.Sprintf("The provided ID %q is not a valid UUID: %s", systemId, err),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), systemId)...)
}
