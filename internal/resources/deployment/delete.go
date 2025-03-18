// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package deployment

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Delete deletes the deployment resource.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting deployment resource")

	// Get the current state
	var state DeploymentModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if ID is set
	if state.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Deployment ID",
			"Cannot delete deployment without ID. This is a bug in the provider.",
		)
		return
	}

	// Parse ID to UUID
	deploymentID, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Deployment ID",
			fmt.Sprintf("Cannot parse deployment ID as UUID: %s", err),
		)
		return
	}

	tflog.Debug(ctx, "Deleting deployment", map[string]interface{}{
		"id": deploymentID.String(),
	})

	// Call API to delete deployment
	response, err := r.client.DeleteDeploymentWithResponse(ctx, deploymentID)
	if err != nil {
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Unable to delete deployment: %s", err),
		)
		return
	}

	// Check the response status
	if response.StatusCode() != http.StatusNoContent && response.StatusCode() != http.StatusOK {
		// If resource does not exist, consider it "deleted"
		if response.StatusCode() == http.StatusNotFound {
			tflog.Warn(ctx, "Deployment already deleted or does not exist", map[string]interface{}{
				"id": deploymentID.String(),
			})
			return
		}

		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to delete deployment. Status: %d, Body: %s",
				response.StatusCode(), string(response.Body)),
		)
		return
	}

	// Successfully deleted
	tflog.Info(ctx, "Deployment deleted successfully", map[string]interface{}{
		"id": deploymentID.String(),
	})
}
