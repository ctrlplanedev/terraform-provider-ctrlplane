// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithImportState = &EnvironmentResource{}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

// ResourceFilter describes the resource filter configuration
type ResourceFilter struct {
	FilterType types.String `tfsdk:"filter_type"`
	Namespace  types.String `tfsdk:"namespace"`
}

// EnvironmentResource defines the resource implementation.
type EnvironmentResource struct {
	client    *client.ClientWithResponses
	workspace uuid.UUID
}

// EnvironmentResourceModel describes the resource data model.
type EnvironmentResourceModel struct {
	ID             types.String  `tfsdk:"id"`
	Name           types.String  `tfsdk:"name"`
	Description    types.String  `tfsdk:"description"`
	SystemID       types.String  `tfsdk:"system_id"`
	PolicyID       types.String  `tfsdk:"policy_id"`
	Metadata       types.Map     `tfsdk:"metadata"`
	ResourceFilter types.Dynamic `tfsdk:"resource_filter"`
}

func (r *EnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Environment resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Environment identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the environment",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the environment",
				Optional:            true,
			},
			"system_id": schema.StringAttribute{
				MarkdownDescription: "System ID the environment belongs to",
				Required:            true,
			},
			"policy_id": schema.StringAttribute{
				MarkdownDescription: "Policy ID for the environment",
				Optional:            true,
			},
			"metadata": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Metadata for the environment",
				Optional:            true,
			},
			"resource_filter": schema.DynamicAttribute{
				MarkdownDescription: "Resource filter for the environment",
				Optional:           true,
			},
		},
	}
}

func (r *EnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	dataSourceModel, ok := req.ProviderData.(*DataSourceModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *DataSourceModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = dataSourceModel.Client
	r.workspace = dataSourceModel.Workspace
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *EnvironmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert metadata to map[string]string
	metadata := make(map[string]string)
	if !data.Metadata.IsNull() {
		resp.Diagnostics.Append(data.Metadata.ElementsAs(ctx, &metadata, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Convert resource filter to map[string]interface{}
	var resourceFilter map[string]interface{}
	if !data.ResourceFilter.IsNull() {
		val, err := data.ResourceFilter.ToTerraformValue(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read resource filter: %s", err))
			return
		}
		if err := val.As(&resourceFilter); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse resource filter: %s", err))
			return
		}
	}

	// Create new environment
	createResp, err := r.client.CreateEnvironmentWithResponse(ctx, client.CreateEnvironmentJSONRequestBody{
		Name:           data.Name.ValueString(),
		Description:    stringToPtr(data.Description.ValueString()),
		SystemId:       data.SystemID.ValueString(),
		PolicyId:       stringToPtr(data.PolicyID.ValueString()),
		Metadata:       &metadata,
		ResourceFilter: &resourceFilter,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create environment, got error: %s", err))
		return
	}

	if createResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to create environment, got empty response")
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(createResp.JSON200.Id.String())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *EnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.client.GetEnvironmentWithResponse(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	if getResp.JSON200 == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(getResp.JSON200.Name)
	if getResp.JSON200.Description != nil {
		data.Description = types.StringValue(*getResp.JSON200.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.SystemID = types.StringValue(getResp.JSON200.SystemId.String())
	if getResp.JSON200.PolicyId != nil {
		data.PolicyID = types.StringValue(getResp.JSON200.PolicyId.String())
	} else {
		data.PolicyID = types.StringNull()
	}

	if getResp.JSON200.Metadata != nil {
		metadata := make(map[string]attr.Value)
		for k, v := range *getResp.JSON200.Metadata {
			metadata[k] = types.StringValue(v)
		}
		data.Metadata = types.MapValueMust(types.StringType, metadata)
	} else {
		data.Metadata = types.MapNull(types.StringType)
	}

	if getResp.JSON200.ResourceFilter != nil {
		data.ResourceFilter = types.DynamicValue(types.ObjectValueMust(
			map[string]attr.Type{
				"filter_type": types.StringType,
				"namespace":   types.StringType,
			},
			map[string]attr.Value{
				"filter_type": types.StringValue((*getResp.JSON200.ResourceFilter)["type"].(string)),
				"namespace":   types.StringValue((*getResp.JSON200.ResourceFilter)["namespace"].(string)),
			},
		))
	} else {
		data.ResourceFilter = types.DynamicNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *EnvironmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update environment using API - Note: The API doesn't seem to have a direct update endpoint
	// We might need to delete and recreate, or implement a different update strategy
	resp.Diagnostics.AddError(
		"Update Not Implemented",
		"The provider does not support updating environments at this time",
	)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *EnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete environment
	_, err := r.client.DeleteEnvironmentWithResponse(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete environment, got error: %s", err))
		return
	}
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *ResourceFilter) IsNull() bool {
	return r == nil
}

func (r *ResourceFilter) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if r == nil {
		return tftypes.NewValue(tftypes.Object{}, nil), nil
	}
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"filter_type": tftypes.String,
			"namespace":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"filter_type": tftypes.NewValue(tftypes.String, r.FilterType.ValueString()),
		"namespace":   tftypes.NewValue(tftypes.String, r.Namespace.ValueString()),
	}), nil
}
