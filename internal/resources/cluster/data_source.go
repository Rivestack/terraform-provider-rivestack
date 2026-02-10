// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package cluster

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rivestack/terraform-provider-rivestack/internal/client"
)

var _ datasource.DataSource = &clusterDataSource{}

// NewDataSource returns a new cluster data source.
func NewDataSource() datasource.DataSource {
	return &clusterDataSource{}
}

type clusterDataSource struct {
	client *client.Client
}

type clusterDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	TenantID          types.String `tfsdk:"tenant_id"`
	Region            types.String `tfsdk:"region"`
	DBType            types.String `tfsdk:"db_type"`
	ServerType        types.String `tfsdk:"server_type"`
	NodeCount         types.Int64  `tfsdk:"node_count"`
	PostgreSQLVersion types.Int64  `tfsdk:"postgresql_version"`
	DBName            types.String `tfsdk:"db_name"`
	DBUser            types.String `tfsdk:"db_user"`
	DBPassword        types.String `tfsdk:"db_password"`
	Host              types.String `tfsdk:"host"`
	ConnectionString  types.String `tfsdk:"connection_string"`
	Status            types.String `tfsdk:"status"`
	HealthStatus      types.String `tfsdk:"health_status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (d *clusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (d *clusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to read information about an existing Rivestack HA PostgreSQL cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Cluster ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Display name.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "Unique tenant identifier.",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "Cluster region.",
				Computed:    true,
			},
			"db_type": schema.StringAttribute{
				Description: "Cluster type (ha or core_solo).",
				Computed:    true,
			},
			"server_type": schema.StringAttribute{
				Description: "Server size.",
				Computed:    true,
			},
			"node_count": schema.Int64Attribute{
				Description: "Number of nodes.",
				Computed:    true,
			},
			"postgresql_version": schema.Int64Attribute{
				Description: "PostgreSQL major version.",
				Computed:    true,
			},
			"db_name": schema.StringAttribute{
				Description: "Default database name.",
				Computed:    true,
			},
			"db_user": schema.StringAttribute{
				Description: "Default database user.",
				Computed:    true,
			},
			"db_password": schema.StringAttribute{
				Description: "Default database user password.",
				Computed:    true,
				Sensitive:   true,
			},
			"host": schema.StringAttribute{
				Description: "Cluster hostname.",
				Computed:    true,
			},
			"connection_string": schema.StringAttribute{
				Description: "Full PostgreSQL connection string.",
				Computed:    true,
				Sensitive:   true,
			},
			"status": schema.StringAttribute{
				Description: "Cluster status.",
				Computed:    true,
			},
			"health_status": schema.StringAttribute{
				Description: "Cluster health status.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Last update timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *clusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *clusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state clusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster ID",
			fmt.Sprintf("Could not parse cluster ID %q: %s", state.ID.ValueString(), err))
		return
	}

	cluster, err := d.client.GetCluster(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading cluster",
			fmt.Sprintf("Could not read cluster %d: %s", id, err))
		return
	}

	state.ID = types.StringValue(strconv.Itoa(cluster.ID))
	state.Name = types.StringValue(cluster.Name)
	state.TenantID = types.StringValue(cluster.TenantID)
	state.Region = types.StringValue(cluster.Region)
	state.DBType = types.StringValue(cluster.DBType)
	state.ServerType = types.StringValue(cluster.ServerType)
	state.NodeCount = types.Int64Value(int64(cluster.NodeCount))
	state.PostgreSQLVersion = types.Int64Value(int64(cluster.PostgreSQLVersion))
	state.DBName = types.StringValue(cluster.DBName)
	state.DBUser = types.StringValue(cluster.DBUser)
	state.DBPassword = types.StringValue(cluster.DBPassword)
	state.Host = types.StringValue(cluster.Host)
	state.ConnectionString = types.StringValue(cluster.ConnectionString)
	state.Status = types.StringValue(cluster.Status)
	state.HealthStatus = types.StringValue(cluster.HealthStatus)
	state.CreatedAt = types.StringValue(cluster.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(cluster.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
