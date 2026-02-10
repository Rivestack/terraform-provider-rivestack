// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rivestack/terraform-provider-rivestack/internal/client"
	"github.com/rivestack/terraform-provider-rivestack/internal/resources/cluster"
	"github.com/rivestack/terraform-provider-rivestack/internal/resources/cluster_backup_config"
	"github.com/rivestack/terraform-provider-rivestack/internal/resources/cluster_database"
	"github.com/rivestack/terraform-provider-rivestack/internal/resources/cluster_extension"

	"github.com/rivestack/terraform-provider-rivestack/internal/resources/cluster_grant"
	"github.com/rivestack/terraform-provider-rivestack/internal/resources/cluster_user"
	"github.com/rivestack/terraform-provider-rivestack/internal/resources/extensions"
	"github.com/rivestack/terraform-provider-rivestack/internal/resources/server_types"
)

var _ provider.Provider = &RivestackProvider{}

// RivestackProvider defines the Rivestack Terraform provider.
type RivestackProvider struct {
	version string
}

// RivestackProviderModel describes the provider configuration data model.
type RivestackProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

// New returns a new provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RivestackProvider{
			version: version,
		}
	}
}

func (p *RivestackProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "rivestack"
	resp.Version = p.version
}

func (p *RivestackProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Rivestack provider is used to manage Rivestack HA PostgreSQL clusters.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "Rivestack API key (rsk_ prefix). Can also be set via the RIVESTACK_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "Rivestack API base URL. Defaults to https://api.rivestack.io. Can also be set via the RIVESTACK_BASE_URL environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *RivestackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config RivestackProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve API key: config > env var.
	apiKey := config.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("RIVESTACK_API_KEY")
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The Rivestack API key must be set in the provider configuration or via the RIVESTACK_API_KEY environment variable.",
		)
		return
	}

	// Resolve base URL: config > env var > default.
	baseURL := config.BaseURL.ValueString()
	if baseURL == "" {
		baseURL = os.Getenv("RIVESTACK_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.rivestack.io"
	}

	c := client.NewClient(baseURL, apiKey, p.version)

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *RivestackProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		cluster.NewResource,
		cluster_user.NewResource,
		cluster_database.NewResource,
		cluster_extension.NewResource,
		cluster_grant.NewResource,
		cluster_backup_config.NewResource,
	}
}

func (p *RivestackProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		cluster.NewDataSource,
		server_types.NewDataSource,
		extensions.NewDataSource,
	}
}
