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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithImportState = &EnvironmentResource{}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

type EnvironmentResource struct {
	client    *client.ClientWithResponses
	workspace uuid.UUID
}

func (r *EnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = GetEnvironmentResourceSchema()
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
	defer func() {
		var data EnvironmentModel
		diags := resp.State.Get(ctx, &data)
		if diags.HasError() {
			return
		}
		if data.ResourceFilter != nil {
			DeepFixResourceFilter(data.ResourceFilter)
			resp.State.Set(ctx, &data)
		}
	}()

	var data EnvironmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate mutually exclusive resource_filter and resource_filter_id.
	if !data.ResourceFilterID.IsNull() && data.ResourceFilter != nil {
		resp.Diagnostics.AddError(
			"Conflicting Resource Filter Configuration",
			"Both resource_filter and resource_filter_id are specified. Only one can be used at a time.",
		)
		return
	}

	tflog.Debug(ctx, "Resource filter details", map[string]interface{}{
		"has_resource_filter":    data.ResourceFilter != nil,
		"has_resource_filter_id": !data.ResourceFilterID.IsNull(),
		"type":                   getFilterTypeForLog(data.ResourceFilter),
		"conditions_count":       getConditionsCountForLog(data.ResourceFilter),
	})

	if data.ResourceFilter != nil {
		data.ResourceFilter.InitConditions()
		tflog.Debug(ctx, "Resource filter after initialization", map[string]interface{}{
			"type":                   getFilterTypeForLog(data.ResourceFilter),
			"conditions_count":       getConditionsCountForLog(data.ResourceFilter),
			"conditions_initialized": data.ResourceFilter.Conditions != nil,
		})
	}

	if data.Description.IsNull() {
		data.Description = types.StringValue("")
	}
	if data.Metadata.IsNull() {
		data.Metadata = types.MapNull(types.StringType)
	}

	tflog.Debug(ctx, "PolicyID state before API call", map[string]interface{}{
		"is_null":        data.PolicyID.IsNull(),
		"is_unknown":     data.PolicyID.IsUnknown(),
		"policy_id_type": fmt.Sprintf("%T", data.PolicyID),
	})

	metadata := make(map[string]string)
	if !data.Metadata.IsNull() {
		resp.Diagnostics.Append(data.Metadata.ElementsAs(ctx, &metadata, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var resourceFilter *map[string]interface{}
	if data.ResourceFilter != nil {
		// Process inline resource filter.
		DeepFixResourceFilter(data.ResourceFilter)
		data.ResourceFilter.InitConditions()
		if data.ResourceFilter.Type.IsNull() {
			resp.Diagnostics.AddError("Invalid Resource Filter", "Resource filter must have a type")
			return
		}
		filterMap, err := data.ResourceFilter.ToAPIFilter(ctx)
		if err != nil {
			resp.Diagnostics.AddError("ResourceFilter Conversion Error", fmt.Sprintf("Error converting inline resource filter: %v", err))
			return
		}
		// Validate and normalize the filter.
		validatedFilter := ValidateAPIFilter(filterMap)
		resourceFilter = &validatedFilter
		tflog.Info(ctx, "Converted inline resource filter to API format", map[string]interface{}{
			"filter_type":    validatedFilter["type"],
			"has_conditions": validatedFilter["conditions"] != nil,
			"filter_payload": fmt.Sprintf("%+v", validatedFilter),
		})
	} else if !data.ResourceFilterID.IsNull() {
		// Use referenced resource filter.
		filterID := data.ResourceFilterID.ValueString()
		tflog.Debug(ctx, "Using referenced resource filter", map[string]interface{}{
			"resource_filter_id": filterID,
		})
		refFilter, err := GetResourceFilterByID(ctx, filterID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to Retrieve ResourceFilter",
				fmt.Sprintf("Unable to retrieve filter with ID %s: %v\n\n"+
					"This may occur if the resource filter resource has not been created yet. "+
					"Consider adding a depends_on = [ctrlplane_resource_filter.your_filter] "+
					"to ensure the filter is created before the environment.",
					filterID, err),
			)
			return
		}
		apiFilter, err := refFilter.ToAPIFilter(ctx)
		if err != nil {
			resp.Diagnostics.AddError("ResourceFilter Conversion Error", fmt.Sprintf("Error converting referenced resource filter: %v", err))
			return
		}
		resourceFilter = &apiFilter
		tflog.Info(ctx, "Converted referenced resource filter to API format", map[string]interface{}{
			"filter_type":    apiFilter["type"],
			"has_conditions": apiFilter["conditions"] != nil,
			"filter_details": fmt.Sprintf("%+v", apiFilter),
		})
	}

	releaseChannels := make([]string, 0)
	for _, ch := range data.ReleaseChannels {
		if !ch.IsNull() && !ch.IsUnknown() {
			releaseChannels = append(releaseChannels, ch.ValueString())
		}
	}

	createReq := client.CreateEnvironmentJSONRequestBody{
		Name:            data.Name.ValueString(),
		Description:     stringToPtr(data.Description.ValueString()),
		SystemId:        data.SystemID.ValueString(),
		Metadata:        &metadata,
		ResourceFilter:  resourceFilter,
		ReleaseChannels: &releaseChannels,
	}

	tflog.Info(ctx, "API request details", map[string]interface{}{
		"name":             data.Name.ValueString(),
		"description":      data.Description.ValueString(),
		"system_id":        data.SystemID.ValueString(),
		"metadata":         metadata,
		"resource_filter":  fmt.Sprintf("%+v", resourceFilter),
		"release_channels": releaseChannels,
	})

	createResp, err := r.client.CreateEnvironmentWithResponse(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create environment: %v", err))
		return
	}
	if createResp.StatusCode() >= 400 {
		respBody := string(createResp.Body)
		tflog.Error(ctx, "API error response", map[string]interface{}{
			"status_code":             createResp.StatusCode(),
			"response_body":           respBody,
			"resource_filter_present": resourceFilter != nil,
		})
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Received error response: %d, body: %s", createResp.StatusCode(), respBody))
		return
	}

	data.ID = types.StringValue(createResp.JSON200.Id.String())
	data.Name = types.StringValue(createResp.JSON200.Name)
	data.Description = types.StringValue(*createResp.JSON200.Description)
	data.SystemID = types.StringValue(createResp.JSON200.SystemId.String())
	if createResp.JSON200.PolicyId != nil {
		data.PolicyID = types.StringValue(createResp.JSON200.PolicyId.String())
	} else {
		data.PolicyID = types.StringNull()
	}

	if createResp.JSON200.Metadata != nil {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, *createResp.JSON200.Metadata)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Metadata = metadataMap
		}
	}

	if createResp.JSON200 != nil && createResp.JSON200.ResourceFilter == nil {
		tflog.Debug(ctx, "API response has ResourceFilter as nil, setting Terraform state accordingly")
		data.ResourceFilter = nil
	} else if createResp.JSON200.ResourceFilter != nil {
		if !data.ResourceFilterID.IsNull() {
			tflog.Debug(ctx, "Using resource_filter_id, setting resource_filter to nil in state")
			data.ResourceFilter = nil
		} else {
			rf := *createResp.JSON200.ResourceFilter
			var filter ResourceFilterModel
			err = filter.FromAPIFilter(ctx, rf)
			if err != nil {
				resp.Diagnostics.AddWarning("ResourceFilter Conversion Error", fmt.Sprintf("Error converting resource filter: %v", err))
			} else {
				data.ResourceFilter = &filter
				PostProcessResourceFilter(ctx, data.ResourceFilter)
			}
		}
	}

	tflog.Debug(ctx, "Saving environment state", map[string]interface{}{
		"environment_id": data.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	defer func() {
		var data EnvironmentModel
		diags := resp.State.Get(ctx, &data)
		if diags.HasError() {
			return
		}
		// Additional processing if needed.
	}()

	var state EnvironmentModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("ID is missing", "Environment ID is required")
		return
	}

	getEnvResponse, err := r.client.GetEnvironmentWithResponse(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get environment", err.Error())
		return
	}
	if getEnvResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError("Failed to get environment", fmt.Sprintf("Status code: %d", getEnvResponse.StatusCode()))
		return
	}

	envData := getEnvResponse.JSON200
	state.ID = types.StringValue(envData.Id.String())
	state.Name = types.StringValue(envData.Name)
	if envData.Description != nil {
		state.Description = types.StringValue(*envData.Description)
	} else {
		state.Description = types.StringValue("")
	}
	state.SystemID = types.StringValue(envData.SystemId.String())
	if envData.PolicyId != nil {
		state.PolicyID = types.StringValue(envData.PolicyId.String())
	} else {
		state.PolicyID = types.StringNull()
	}

	if envData.Metadata != nil {
		metadataMap := make(map[string]attr.Value)
		for k, v := range *envData.Metadata {
			metadataMap[k] = types.StringValue(v)
		}
		metadata, metadataDiags := types.MapValueFrom(ctx, types.StringType, metadataMap)
		resp.Diagnostics.Append(metadataDiags...)
		if !resp.Diagnostics.HasError() {
			state.Metadata = metadata
		}
	} else {
		state.Metadata = types.MapNull(types.StringType)
	}

	if envData.ResourceFilter == nil {
		tflog.Debug(ctx, "ResourceFilter is nil in API response, setting Terraform state to null")
		state.ResourceFilter = nil
	} else {
		if !state.ResourceFilterID.IsNull() {
			tflog.Debug(ctx, "Using resource_filter_id, setting resource_filter to nil in state")
			state.ResourceFilter = nil
		} else {
			filter, err := CreateResourceFilterModel(ctx, *envData.ResourceFilter)
			if err != nil {
				resp.Diagnostics.AddError("Failed to convert resource filter", err.Error())
				return
			}
			state.ResourceFilter = filter
			PostProcessResourceFilter(ctx, state.ResourceFilter)
		}
	}

	if state.ResourceFilter != nil && state.ResourceFilter.Conditions != nil && len(state.ResourceFilter.Conditions) > 0 {
		DeepFixResourceFilter(state.ResourceFilter)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state EnvironmentModel
	diags := req.Plan.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	defer func() {
		if resp.Diagnostics.HasError() {
			return
		}
		var data EnvironmentModel
		diags := resp.State.Get(ctx, &data)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		modified := false
		if data.ResourceFilter != nil {
			for i, condition := range data.ResourceFilter.Conditions {
				if condition.Type.ValueString() == string(FilterTypeComparison) && len(condition.Conditions) > 0 {
					data.ResourceFilter.Conditions[i].Value = types.StringValue("")
					modified = true
					tflog.Debug(ctx, "Deferred update: Set nested comparison condition value to empty string", map[string]interface{}{
						"index": i,
					})
				}
			}
			if modified {
				diags = resp.State.Set(ctx, &data)
				resp.Diagnostics.Append(diags...)
			}
		}
	}()

	if !state.ResourceFilterID.IsNull() && state.ResourceFilter != nil {
		resp.Diagnostics.AddError(
			"Conflicting Resource Filter Configuration",
			"Both resource_filter and resource_filter_id are specified. Only one can be used at a time.",
		)
		return
	}

	tflog.Debug(ctx, "Updating environment (non-filter attributes only)", map[string]interface{}{
		"environment_id":         state.ID.ValueString(),
		"has_resource_filter":    state.ResourceFilter != nil,
		"has_resource_filter_id": !state.ResourceFilterID.IsNull(),
	})

	currentState, err := r.client.GetEnvironmentWithResponse(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get environment: %v", err))
		return
	}
	if currentState.StatusCode() != 200 {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Received error response: %d, body: %s", currentState.StatusCode(), string(currentState.Body)))
		return
	}

	envData := currentState.JSON200
	state.ID = types.StringValue(envData.Id.String())
	state.Name = types.StringValue(envData.Name)
	if envData.Description != nil {
		state.Description = types.StringValue(*envData.Description)
	} else {
		state.Description = types.StringValue("")
	}
	state.SystemID = types.StringValue(envData.SystemId.String())
	if envData.PolicyId != nil {
		state.PolicyID = types.StringValue(envData.PolicyId.String())
	} else {
		state.PolicyID = types.StringNull()
	}

	if envData.Metadata != nil {
		metadataMap := make(map[string]attr.Value)
		for k, v := range *envData.Metadata {
			metadataMap[k] = types.StringValue(v)
		}
		metadata, metadataDiags := types.MapValueFrom(ctx, types.StringType, metadataMap)
		resp.Diagnostics.Append(metadataDiags...)
		if !resp.Diagnostics.HasError() {
			state.Metadata = metadata
		}
	} else {
		state.Metadata = types.MapNull(types.StringType)
	}

	if envData.ResourceFilter == nil {
		tflog.Debug(ctx, "ResourceFilter is nil in API response, setting Terraform state to null")
		state.ResourceFilter = nil
	} else {
		if !state.ResourceFilterID.IsNull() {
			tflog.Debug(ctx, "Using resource_filter_id, setting resource_filter to nil in state")
			state.ResourceFilter = nil
		} else {
			filter, err := CreateResourceFilterModel(ctx, *envData.ResourceFilter)
			if err != nil {
				resp.Diagnostics.AddError("Failed to convert resource filter", err.Error())
				return
			}
			state.ResourceFilter = filter
			PostProcessResourceFilter(ctx, state.ResourceFilter)
		}
	}

	if state.ResourceFilter != nil && state.ResourceFilter.Conditions != nil && len(state.ResourceFilter.Conditions) > 0 {
		DeepFixResourceFilter(state.ResourceFilter)
	}

	tflog.Debug(ctx, "Saving updated environment state", map[string]interface{}{
		"environment_id": state.ID.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *EnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.DeleteEnvironmentWithResponse(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete environment, got error: %s", err))
		return
	}
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func getFilterTypeForLog(filter *ResourceFilterModel) string {
	if filter != nil && !filter.Type.IsNull() {
		return filter.Type.ValueString()
	}
	return "null"
}

func getConditionsCountForLog(filter *ResourceFilterModel) int {
	if filter != nil {
		return len(filter.Conditions)
	}
	return 0
}

func CreateResourceFilterModel(ctx context.Context, rf map[string]interface{}) (*ResourceFilterModel, error) {
	filter := &ResourceFilterModel{}
	err := filter.FromAPIFilter(ctx, rf)
	if err != nil {
		return nil, err
	}
	return filter, nil
}
