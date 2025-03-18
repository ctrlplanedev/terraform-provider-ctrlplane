// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ctrlplane/client"
)

var (
	_ datasource.DataSource = &DataSource{}
)

func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

type DataSource struct {
	client *client.ClientWithResponses
}

func (d *DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (d *DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = GetDeploymentDataSourceSchema()
}

func (d *DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	// Use reflection to safely extract the Client field from the provider data
	// without needing to import the provider package directly
	providerValue := reflect.ValueOf(req.ProviderData).Elem()
	clientField := providerValue.FieldByName("Client")

	if !clientField.IsValid() {
		resp.Diagnostics.AddError(
			"Invalid Provider Data",
			"Provider data does not contain a Client field",
		)
		return
	}

	client, ok := clientField.Interface().(*client.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Client Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T", clientField.Interface()),
		)
		return
	}

	d.client = client
}

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DeploymentModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ID.IsNull() && config.Name.IsNull() && config.SystemID.IsNull() && config.Slug.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"At least one of id, name, system_id, or slug must be provided to find a deployment",
		)
		return
	}

	if !config.ID.IsNull() {
		deploymentID, err := uuid.Parse(config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Deployment ID",
				fmt.Sprintf("Cannot parse deployment ID as UUID: %s", err),
			)
			return
		}

		response, err := d.client.GetDeploymentWithResponse(ctx, deploymentID)
		if err != nil {
			resp.Diagnostics.AddError(
				"API Error",
				fmt.Sprintf("Unable to read deployment by ID: %s", err),
			)
			return
		}

		if response.StatusCode() == http.StatusNotFound {
			resp.Diagnostics.AddError(
				"Deployment Not Found",
				fmt.Sprintf("No deployment found with ID %s", deploymentID.String()),
			)
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

		config = mapDeploymentToModel(ctx, deployment, resp, config)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"ID Required",
			"Deployment lookup requires an ID. Please specify the 'id' attribute.",
		)
		return
	}

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

func mapDeploymentToModel(ctx context.Context, deployment client.Deployment, resp *datasource.ReadResponse, model DeploymentModel) DeploymentModel {
	model.ID = types.StringValue(deployment.Id.String())
	model.Name = types.StringValue(deployment.Name)
	model.Description = types.StringValue(deployment.Description)
	model.SystemID = types.StringValue(deployment.SystemId.String())
	model.Slug = types.StringValue(deployment.Slug)

	if deployment.JobAgentId != nil {
		model.JobAgentID = types.StringValue(deployment.JobAgentId.String())
	} else {
		model.JobAgentID = types.StringNull()
	}

	jobAgentConfigMap, diags := types.MapValueFrom(ctx, types.StringType, deployment.JobAgentConfig)
	resp.Diagnostics.Append(diags...)
	model.JobAgentConfig = jobAgentConfigMap

	if deployment.RetryCount != nil {
		model.RetryCount = types.Int64Value(int64(*deployment.RetryCount))
	} else {
		model.RetryCount = types.Int64Null()
	}

	if deployment.Timeout != nil {
		model.Timeout = types.Int64Value(int64(*deployment.Timeout))
	} else {
		model.Timeout = types.Int64Null()
	}

	model.ResourceFilter = types.MapNull(types.StringType)

	return model
}
