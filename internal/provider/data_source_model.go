// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
)

type DataSourceModel struct {
	Workspace uuid.UUID                   `tfsdk:"workspace"`
	Client    *client.ClientWithResponses `tfsdk:"client"`
}
