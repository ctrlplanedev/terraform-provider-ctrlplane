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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-ctrlplane/client"
)

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating deployment resource")

	var plan DeploymentModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Name.IsNull() || plan.SystemID.IsNull() || plan.Slug.IsNull() || plan.JobAgentConfig.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Fields",
			"One or more required fields are missing. Required fields are: name, system_id, slug, job_agent_config",
		)
		return
	}

	systemID, err := uuid.Parse(plan.SystemID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid System ID",
			fmt.Sprintf("Cannot parse system ID as UUID: %s", err),
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

	var resourceFilter map[string]interface{}
	if !plan.ResourceFilter.IsNull() {
		diags = plan.ResourceFilter.ElementsAs(ctx, &resourceFilter, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	createRequest := client.CreateDeploymentJSONRequestBody{
		Name:     plan.Name.ValueString(),
		SystemId: systemID,
		Slug:     plan.Slug.ValueString(),
	}

	if !plan.Description.IsNull() {
		desc := plan.Description.ValueString()
		createRequest.Description = &desc
	}

	createRequest.JobAgentConfig = &jobAgentConfig

	if !plan.JobAgentID.IsNull() {
		jobAgentID, err := uuid.Parse(plan.JobAgentID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Job Agent ID",
				fmt.Sprintf("Cannot parse job agent ID as UUID: %s", err),
			)
			return
		}
		createRequest.JobAgentId = &jobAgentID
	}

	if !plan.RetryCount.IsNull() {
		retryCountInt := int(plan.RetryCount.ValueInt64())
		retryCountFloat := float32(retryCountInt)
		createRequest.RetryCount = &retryCountFloat
	}

	if !plan.Timeout.IsNull() {
		timeoutInt := int(plan.Timeout.ValueInt64())
		timeoutFloat := float32(timeoutInt)
		createRequest.Timeout = &timeoutFloat
	}

	if resourceFilter != nil {
		createRequest.ResourceFilter = &resourceFilter
	}

	response, err := r.client.CreateDeploymentWithResponse(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to create deployment: %s", err),
		)
		return
	}

	if response.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to create deployment. Status: %d, Body: %s",
				response.StatusCode(), string(response.Body)),
		)
		return
	}

	var deployment client.Deployment
	if err := json.Unmarshal(response.Body, &deployment); err != nil {
		resp.Diagnostics.AddError(
			"API Response Error",
			fmt.Sprintf("Unable to unmarshal deployment creation response: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(deployment.Id.String())

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}
