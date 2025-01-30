// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure CtrlplaneProvider satisfies various provider interfaces.
var _ provider.Provider = &CtrlplaneProvider{}
var _ provider.ProviderWithFunctions = &CtrlplaneProvider{}

// CtrlplaneProvider defines the provider implementation.
type CtrlplaneProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// CtrlplaneProviderModel describes the provider data model.
type CtrlplaneProviderModel struct {
	BaseURL   types.String `tfsdk:"base_url"`
	Token     types.String `tfsdk:"token"`
	Workspace types.String `tfsdk:"workspace"`
}

func (p *CtrlplaneProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ctrlplane"
	resp.Version = p.version
}

func (p *CtrlplaneProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "The Ctrlplane provider is used to manage the lifecycle of your Ctrlplane constructs, including systems, policies, resources, and more. A provider is scoped to a workspace, and can be configured with a base URL and token.",
		MarkdownDescription: "The Ctrlplane provider is used to manage the lifecycle of your Ctrlplane constructs, including systems, policies, resources, and more. A provider is scoped to a workspace, and can be configured with a base URL and token.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Description:         "The URL of the Ctrlplane API endpoint. Can be set in the CTRLPLANE_BASE_URL environment variable. Defaults to `https://app.ctrlplane.com` if not set.",
				MarkdownDescription: "The URL of the Ctrlplane API endpoint. Can be set in the CTRLPLANE_BASE_URL environment variable. Defaults to `https://app.ctrlplane.com` if not set.",
				Optional:            true,
			},
			"token": schema.StringAttribute{
				Description:         "The token to use for authentication. Can be set in the CTRLPLANE_TOKEN environment variable.",
				MarkdownDescription: "The token to use for authentication. Can be set in the CTRLPLANE_TOKEN environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"workspace": schema.StringAttribute{
				Description:         "The workspace to use. Can be set in the CTRLPLANE_WORKSPACE environment variable. Can be a workspace ID or slug.",
				MarkdownDescription: "The workspace to use. Can be set in the CTRLPLANE_WORKSPACE environment variable. Can be a workspace ID or slug.",
				Optional:            true,
			},
		},
	}
}

func addAPIKey(apiKey string) client.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Add("x-api-key", apiKey)
		return nil
	}
}

func getWorkspaceById(ctx context.Context, workspaceID uuid.UUID, client *client.ClientWithResponses) (uuid.UUID, error) {
	validatedWorkspace, err := client.GetWorkspaceWithResponse(ctx, workspaceID)
	if err != nil {
		return uuid.Nil, err
	}

	if validatedWorkspace.JSON200 != nil {
		return validatedWorkspace.JSON200.Id, nil
	}

	if validatedWorkspace.JSON404 != nil {
		return uuid.Nil, errors.New("workspace not found")
	}

	return uuid.Nil, errors.New("failed to get workspace by id")
}

func getWorkspaceBySlug(ctx context.Context, slug string, client *client.ClientWithResponses) (uuid.UUID, error) {
	validatedWorkspace, err := client.GetWorkspaceBySlugWithResponse(ctx, slug)
	if err != nil {
		return uuid.Nil, err
	}

	if validatedWorkspace.JSON200 != nil {
		return validatedWorkspace.JSON200.Id, nil
	}

	if validatedWorkspace.JSON404 != nil {
		return uuid.Nil, errors.New("workspace not found")
	}

	return uuid.Nil, fmt.Errorf("failed to get workspace by slug: %s", slug)
}

func getWorkspace(ctx context.Context, workspace string, client *client.ClientWithResponses) (uuid.UUID, error) {
	if workspace == "" {
		return uuid.Nil, errors.New("workspace is required")
	}

	if workspaceID, err := uuid.Parse(workspace); err == nil {
		return getWorkspaceById(ctx, workspaceID, client)
	}
	return getWorkspaceBySlug(ctx, workspace, client)
}

func (p *CtrlplaneProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CtrlplaneProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.BaseURL.IsNull() {
		envBaseURL := os.Getenv("CTRLPLANE_BASE_URL")
		if envBaseURL != "" {
			data.BaseURL = types.StringValue(envBaseURL)
		} else {
			data.BaseURL = types.StringValue("https://app.ctrlplane.dev")
		}
	}

	if data.Token.ValueString() == "" {
		envToken := os.Getenv("CTRLPLANE_TOKEN")
		if envToken == "" {
			resp.Diagnostics.AddError("Missing API key", "The API key must be set either in the provider configuration or in the CTRLPLANE_TOKEN environment variable")
			return
		}
		data.Token = types.StringValue(envToken)
	}

	if data.Workspace.IsNull() {
		envWorkspace := os.Getenv("CTRLPLANE_WORKSPACE")
		if envWorkspace == "" {
			resp.Diagnostics.AddError("Missing workspace", "The workspace must be set either in the provider configuration or in the CTRLPLANE_WORKSPACE environment variable")
			return
		}
		data.Workspace = types.StringValue(envWorkspace)
	}

	server := data.BaseURL.ValueString()
	server = strings.TrimSuffix(server, "/")
	server = strings.TrimSuffix(server, "/api")
	server = server + "/api"

	client, err := client.NewClientWithResponses(
		server,
		client.WithRequestEditorFn(addAPIKey(data.Token.ValueString())),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create client", err.Error())
		return
	}

	configuredWorkspace := data.Workspace.ValueString()
	workspaceID, err := getWorkspace(ctx, configuredWorkspace, client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get workspace", err.Error())
		return
	}

	dataSourceModel := &DataSourceModel{
		Workspace: workspaceID,
		Client:    client,
	}

	resp.DataSourceData = dataSourceModel
	resp.ResourceData = dataSourceModel
}

func (p *CtrlplaneProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSystemResource,
	}
}

func (p *CtrlplaneProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *CtrlplaneProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CtrlplaneProvider{
			version: version,
		}
	}
}
