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

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeploymentModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Deployment ID",
			"Cannot read deployment without ID. This is a bug in the provider.",
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

	tflog.Debug(ctx, "Reading deployment", map[string]interface{}{
		"id": deploymentID.String(),
	})

	response, err := r.client.GetDeploymentWithResponse(ctx, deploymentID)
	if err != nil {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to read deployment: %s", err),
		)
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if response.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Received status %d: %s", response.StatusCode(), string(response.Body)),
		)
		return
	}

	var deployment client.Deployment
	if err := json.Unmarshal(response.Body, &deployment); err != nil {
		resp.Diagnostics.AddError(
			"API Response Error",
			fmt.Sprintf("Unable to unmarshal deployment response: %s", err),
		)
		return
	}

	state.ID = types.StringValue(deployment.Id.String())
	state.Name = types.StringValue(deployment.Name)
	state.Description = types.StringValue(deployment.Description)
	state.SystemID = types.StringValue(deployment.SystemId.String())
	state.Slug = types.StringValue(deployment.Slug)

	if deployment.JobAgentId != nil {
		state.JobAgentID = types.StringValue(deployment.JobAgentId.String())
	} else {
		state.JobAgentID = types.StringNull()
	}

	jobAgentConfigMap, diags := types.MapValueFrom(ctx, types.StringType, deployment.JobAgentConfig)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.JobAgentConfig = jobAgentConfigMap

	if deployment.RetryCount != nil {
		state.RetryCount = types.Int64Value(int64(*deployment.RetryCount))
	} else {
		state.RetryCount = types.Int64Null()
	}

	if deployment.Timeout != nil {
		state.Timeout = types.Int64Value(int64(*deployment.Timeout))
	} else {
		state.Timeout = types.Int64Null()
	}

	// The ResourceFilter field is not returned by the API's Get operation
	// We preserve the value from the existing state if it's a Read after Apply
	// We only set it to null if it doesn't exist in the current state (like during import)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
