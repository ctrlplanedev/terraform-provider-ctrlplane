// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentModel struct {
	ID               types.String         `tfsdk:"id"`
	Name             types.String         `tfsdk:"name"`
	Description      types.String         `tfsdk:"description"`
	SystemID         types.String         `tfsdk:"system_id"`
	PolicyID         types.String         `tfsdk:"policy_id"`
	Metadata         types.Map            `tfsdk:"metadata"`
	ResourceFilter   *ResourceFilterModel `tfsdk:"resource_filter"`
	ResourceFilterID types.String         `tfsdk:"resource_filter_id"`
	ReleaseChannels  []types.String       `tfsdk:"release_channels"`
}

type EnvironmentDataSourceModel struct {
	ID              types.String         `tfsdk:"id"`
	Name            types.String         `tfsdk:"name"`
	Description     types.String         `tfsdk:"description"`
	SystemID        types.String         `tfsdk:"system_id"`
	PolicyID        types.String         `tfsdk:"policy_id"`
	Metadata        types.Map            `tfsdk:"metadata"`
	ResourceFilter  *ResourceFilterModel `tfsdk:"resource_filter"`
	ReleaseChannels []types.String       `tfsdk:"release_channels"`
}

func (e *EnvironmentModel) SetDefaults() {
	if e.Description.IsNull() {
		e.Description = types.StringValue("")
	}
	if e.Metadata.IsNull() {
		e.Metadata = types.MapNull(types.StringType)
	}
	if e.ReleaseChannels == nil {
		e.ReleaseChannels = []types.String{}
	}
}

func (e *EnvironmentDataSourceModel) SetDefaults() {
	if e.Description.IsNull() {
		e.Description = types.StringValue("")
	}
	if e.Metadata.IsNull() {
		e.Metadata = types.MapNull(types.StringType)
	}
	if e.ReleaseChannels == nil {
		e.ReleaseChannels = []types.String{}
	}
}
