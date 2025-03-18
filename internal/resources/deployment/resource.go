// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package deployment

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-ctrlplane/client"
)

var (
	_ resource.Resource                = &Resource{}
	_ resource.ResourceWithImportState = &Resource{}
)

type Resource struct {
	client *client.ClientWithResponses
}

func NewResource() resource.Resource {
	return &Resource{}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

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

	r.client = client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a deployment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Deployment identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the deployment",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description of the deployment",
			},
			"system_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "System ID this deployment belongs to",
			},
			"slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Slug identifier for the deployment",
			},
			"job_agent_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Job agent ID to use for this deployment",
			},
			"job_agent_config": schema.MapAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Job agent configuration",
			},
			"retry_count": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Number of retry attempts",
			},
			"timeout": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Timeout in seconds",
			},
			"resource_filter": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Resource filter configuration",
			},
		},
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing deployment resource", map[string]interface{}{
		"id": req.ID,
	})

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
