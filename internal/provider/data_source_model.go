// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DataSourceModel struct {
	Workspace        uuid.UUID                `tfsdk:"workspace"`
	Client           *api.ClientWithResponses `tfsdk:"client"`
	ID               types.String             `tfsdk:"id"`
	Name             types.String             `tfsdk:"name"`
	ResourceFilter   types.Object             `tfsdk:"resource_filter"`
	CustomAttributes []CustomAttribute        `tfsdk:"custom_attributes"`
}

type CustomAttribute struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}
