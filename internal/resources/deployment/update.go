// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-ctrlplane/client"
)

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DeploymentModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating deployment", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	if state.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Deployment ID",
			"Cannot update deployment without ID. This is a bug in the provider.",
		)
		return
	}

	deploymentID, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Deployment ID",
			fmt.Sprintf("Cannot parse deployment ID as UUID: %s", err),
		)
		return
	}

	var jobAgentConfigMap map[string]string
	diags = plan.JobAgentConfig.ElementsAs(ctx, &jobAgentConfigMap, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobAgentConfig := make(map[string]interface{})
	for k, v := range jobAgentConfigMap {
		jobAgentConfig[k] = v
	}

	// var resourceFilter map[string]interface{}
	// if !plan.ResourceFilter.IsNull() {
	// 	diags = plan.ResourceFilter.ElementsAs(ctx, &resourceFilter, false)
	// 	resp.Diagnostics.Append(diags...)
	// 	if resp.Diagnostics.HasError() {
	// 		return
	// 	}
	// }

	systemID, err := uuid.Parse(plan.SystemID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid System ID",
			fmt.Sprintf("Cannot parse system ID as UUID: %s", err),
		)
		return
	}

	updateRequest := client.UpdateDeploymentJSONRequestBody{
		Id:       deploymentID,
		Name:     plan.Name.ValueString(),
		SystemId: systemID,
		Slug:     plan.Slug.ValueString(),
	}

	if !plan.Description.IsNull() {
		desc := plan.Description.ValueString()
		updateRequest.Description = desc
	}

	updateRequest.JobAgentConfig = jobAgentConfig

	if !plan.JobAgentID.IsNull() {
		agentID, err := uuid.Parse(plan.JobAgentID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Job Agent ID",
				fmt.Sprintf("Cannot parse job agent ID as UUID: %s", err),
			)
			return
		}
		updateRequest.JobAgentId = &agentID
	}

	if !plan.RetryCount.IsNull() {
		retryCount := int(plan.RetryCount.ValueInt64())
		updateRequest.RetryCount = &retryCount
	}

	if !plan.Timeout.IsNull() {
		timeout := int(plan.Timeout.ValueInt64())
		updateRequest.Timeout = &timeout
	}

	// TODO: ResourceFilter will be migrated to ResourceSelector
	// if resourceFilter != nil {
	// 	updateRequest.Set("resourceFilter", &resourceFilter)
	// }

	response, err := r.client.UpdateDeploymentWithResponse(ctx, deploymentID, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to update deployment: %s", err),
		)
		return
	}

	if response.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to update deployment. Status: %d, Body: %s",
				response.StatusCode(), string(response.Body)),
		)
		return
	}

	var deployment client.Deployment
	if err := json.Unmarshal(response.Body, &deployment); err != nil {
		resp.Diagnostics.AddError(
			"API Response Error",
			fmt.Sprintf("Unable to unmarshal deployment update response: %s", err),
		)
		return
	}

	plan.ID = state.ID

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}
