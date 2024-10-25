// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"os"

	"terraform-provider-ctrlplane/client"

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
}

func (p *CtrlplaneProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ctrlplane"
	resp.Version = p.version
}

func (p *CtrlplaneProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the Ctrlplane API endpoint",
				Optional:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token to use for authentication",
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

func (p *CtrlplaneProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CtrlplaneProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.BaseURL.IsNull()  {
		envBaseURL := os.Getenv("CTRLPLANE_BASE_URL")
		if envBaseURL != "" {
			data.BaseURL = types.StringValue(envBaseURL)
		} else {
			data.BaseURL = types.StringValue("https://api.ctrlplane.dev")
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

	client, err := client.NewClient(
		data.BaseURL.ValueString(),
		client.WithRequestEditorFn(addAPIKey(data.Token.ValueString())),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *CtrlplaneProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTargetResource,
	}
}

func (p *CtrlplaneProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func (p *CtrlplaneProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewExampleFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CtrlplaneProvider{
			version: version,
		}
	}
}
