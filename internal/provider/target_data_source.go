// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"terraform-provider-ctrlplane/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &TargetDataSource{}

func NewTargetDataSource() datasource.DataSource {
	return &TargetDataSource{}
}

type TargetDataSource struct {
	client *client.Client
}

type TargetDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func (d *TargetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target"
}

func (d *TargetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Target data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Target identifier",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Target name",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Target type",
				Computed:            true,
			},
		},
	}
}

func (d *TargetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *TargetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TargetDataSourceModel

	tflog.Trace(ctx, "reading target data source")

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := d.client.GetTarget(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Target",
			err.Error(),
		)
		return
	}

	data.Name = types.StringValue(target.Name)
	data.Type = types.StringValue(target.Type)

	tflog.Trace(ctx, "finished reading target data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
