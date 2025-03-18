// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"terraform-provider-ctrlplane/client"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &EnvironmentDataSource{}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

type EnvironmentDataSource struct {
	client    *client.ClientWithResponses
	workspace uuid.UUID
}

func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = GetEnvironmentDataSourceSchema()
}

func (d *EnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	dataSourceModel, ok := req.ProviderData.(*DataSourceModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *DataSourceModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = dataSourceModel.Client
	d.workspace = dataSourceModel.Workspace
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.SetDefaults()

	if data.SystemID.IsNull() || data.Name.IsNull() {
		resp.Diagnostics.AddError("Missing Required Parameters", "Both system_id and name are required to identify an environment")
		return
	}

	systemID, err := uuid.Parse(data.SystemID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid System ID", fmt.Sprintf("Unable to parse system_id as UUID: %s", err))
		return
	}

	cfg := RetryConfig{
		MaxRetries:    5,
		TotalWaitTime: 10 * time.Second,
		RetryDelay:    2 * time.Second,
	}

	var sysResp *client.GetSystemResponse
	err = Retry(ctx, cfg, func() (bool, error) {
		var err error
		sysResp, err = d.client.GetSystemWithResponse(ctx, systemID)
		if err != nil || sysResp.JSON200 == nil || sysResp.JSON200.Environments == nil || len(*sysResp.JSON200.Environments) == 0 {
			return false, err
		}
		return true, nil
	})
	if err != nil || sysResp == nil {
		tflog.Warn(ctx, "Could not retrieve system after multiple attempts", map[string]interface{}{
			"system_id": systemID.String(),
		})
		resp.Diagnostics.AddError("System Retrieval Error", fmt.Sprintf("Could not retrieve system with ID %s", systemID.String()))
		return
	}

	var environmentID string
	for _, env := range *sysResp.JSON200.Environments {
		if env.Name == data.Name.ValueString() {
			environmentID = env.Id.String()
			break
		}
	}
	if environmentID == "" {
		resp.Diagnostics.AddError("Environment Not Found",
			fmt.Sprintf("Could not find environment with name %s in system %s after multiple attempts",
				data.Name.ValueString(), data.SystemID.ValueString()))
		return
	}

	var envResp *client.GetEnvironmentResponse
	err = Retry(ctx, cfg, func() (bool, error) {
		var err error
		envResp, err = d.client.GetEnvironmentWithResponse(ctx, environmentID)
		if err != nil || envResp.JSON200 == nil {
			return false, err
		}
		if envResp.JSON200.PolicyId != nil && envResp.JSON200.Description != nil && envResp.JSON200.Metadata != nil {
			return true, nil
		}
		return false, nil
	})
	if err != nil || envResp == nil || envResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Could not retrieve environment details for ID %s after multiple attempts", environmentID))
		return
	}

	if !(envResp.JSON200.PolicyId != nil && envResp.JSON200.Description != nil && envResp.JSON200.Metadata != nil) {
		tflog.Warn(ctx, "Environment data incomplete", map[string]interface{}{
			"environment_id": environmentID,
		})
		resp.Diagnostics.AddWarning(
			"Environment Data Incomplete",
			"Not all required environment data is available yet. This is normal during creation and will be resolved on subsequent applies.",
		)
	}

	data.ID = types.StringValue(envResp.JSON200.Id.String())
	data.Name = types.StringValue(envResp.JSON200.Name)
	if envResp.JSON200.Description != nil {
		data.Description = types.StringValue(*envResp.JSON200.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.SystemID = types.StringValue(envResp.JSON200.SystemId.String())

	if envResp.JSON200.PolicyId != nil && *envResp.JSON200.PolicyId != uuid.Nil && envResp.JSON200.PolicyId.String() != "00000000-0000-0000-0000-000000000000" {
		data.PolicyID = types.StringValue(envResp.JSON200.PolicyId.String())
	} else {
		data.PolicyID = types.StringNull()
	}

	if envResp.JSON200.Metadata != nil {
		metadata := make(map[string]attr.Value)
		for k, v := range *envResp.JSON200.Metadata {
			metadata[k] = types.StringValue(v)
		}
		data.Metadata = types.MapValueMust(types.StringType, metadata)
	} else {
		data.Metadata = types.MapNull(types.StringType)
	}

	if envResp.JSON200.ResourceFilter != nil {
		rf := *envResp.JSON200.ResourceFilter
		var filter ResourceFilterModel
		err = filter.FromAPIFilter(ctx, rf)
		if err != nil {
			resp.Diagnostics.AddWarning("Resource Filter Conversion", fmt.Sprintf("Error converting resource filter from API: %v", err))
		} else {
			data.ResourceFilter = &filter
		}
	}

	data.DeploymentVersionChannels = []types.String{}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
