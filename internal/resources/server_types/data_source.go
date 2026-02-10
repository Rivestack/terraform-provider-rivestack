// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package server_types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rivestack/terraform-provider-rivestack/internal/client"
)

var _ datasource.DataSource = &serverTypesDataSource{}

func NewDataSource() datasource.DataSource {
	return &serverTypesDataSource{}
}

type serverTypesDataSource struct {
	client *client.Client
}

type serverTypesDataSourceModel struct {
	ServerTypes []serverTypeModel `tfsdk:"server_types"`
	Default     types.String      `tfsdk:"default"`
}

type serverTypeModel struct {
	Type           types.String  `tfsdk:"type"`
	Name           types.String  `tfsdk:"name"`
	Description    types.String  `tfsdk:"description"`
	CPUs           types.Int64   `tfsdk:"cpus"`
	MemoryGB       types.Int64   `tfsdk:"memory_gb"`
	StorageGB      types.Int64   `tfsdk:"storage_gb"`
	StorageAvailGB types.Int64   `tfsdk:"storage_avail_gb"`
	PricePerNode   types.Float64 `tfsdk:"price_per_node"`
}

func (d *serverTypesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_types"
}

func (d *serverTypesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists available server types for Rivestack HA PostgreSQL clusters.",
		Attributes: map[string]schema.Attribute{
			"default": schema.StringAttribute{
				Description: "The default server type.",
				Computed:    true,
			},
			"server_types": schema.ListNestedAttribute{
				Description: "Available server types.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Server type identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Display name.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Server type description.",
							Computed:    true,
						},
						"cpus": schema.Int64Attribute{
							Description: "Number of CPUs.",
							Computed:    true,
						},
						"memory_gb": schema.Int64Attribute{
							Description: "Memory in GB.",
							Computed:    true,
						},
						"storage_gb": schema.Int64Attribute{
							Description: "Total storage in GB.",
							Computed:    true,
						},
						"storage_avail_gb": schema.Int64Attribute{
							Description: "Available storage in GB.",
							Computed:    true,
						},
						"price_per_node": schema.Float64Attribute{
							Description: "Price per node per month.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *serverTypesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serverTypesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apiResp, err := d.client.GetServerTypes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading server types",
			fmt.Sprintf("Could not read server types: %s", err))
		return
	}

	state := serverTypesDataSourceModel{
		Default: types.StringValue(apiResp.Default),
	}

	for _, st := range apiResp.ServerTypes {
		state.ServerTypes = append(state.ServerTypes, serverTypeModel{
			Type:           types.StringValue(st.Type),
			Name:           types.StringValue(st.Name),
			Description:    types.StringValue(st.Description),
			CPUs:           types.Int64Value(int64(st.CPUs)),
			MemoryGB:       types.Int64Value(int64(st.MemoryGB)),
			StorageGB:      types.Int64Value(int64(st.StorageGB)),
			StorageAvailGB: types.Int64Value(int64(st.StorageAvailGB)),
			PricePerNode:   types.Float64Value(st.PricePerNode),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
