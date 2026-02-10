// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package extensions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rivestack/terraform-provider-rivestack/internal/client"
)

var _ datasource.DataSource = &extensionsDataSource{}

func NewDataSource() datasource.DataSource {
	return &extensionsDataSource{}
}

type extensionsDataSource struct {
	client *client.Client
}

type extensionsDataSourceModel struct {
	Extensions []extensionModel `tfsdk:"extensions"`
}

type extensionModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Category    types.String `tfsdk:"category"`
	Default     types.Bool   `tfsdk:"default"`
}

func (d *extensionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_extensions"
}

func (d *extensionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists available PostgreSQL extensions for Rivestack HA clusters.",
		Attributes: map[string]schema.Attribute{
			"extensions": schema.ListNestedAttribute{
				Description: "Available PostgreSQL extensions.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Extension name.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Extension description.",
							Computed:    true,
						},
						"category": schema.StringAttribute{
							Description: "Extension category.",
							Computed:    true,
						},
						"default": schema.BoolAttribute{
							Description: "Whether this extension is installed by default.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *extensionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *extensionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apiResp, err := d.client.GetExtensions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading extensions",
			fmt.Sprintf("Could not read extensions: %s", err))
		return
	}

	state := extensionsDataSourceModel{}

	for _, ext := range apiResp.Extensions {
		state.Extensions = append(state.Extensions, extensionModel{
			Name:        types.StringValue(ext.Name),
			Description: types.StringValue(ext.Description),
			Category:    types.StringValue(ext.Category),
			Default:     types.BoolValue(ext.Default),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
