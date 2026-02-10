// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package cluster

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/rivestack/terraform-provider-rivestack/internal/client"
)

var (
	_ resource.Resource                = &clusterResource{}
	_ resource.ResourceWithImportState = &clusterResource{}
)

// NewResource returns a new cluster resource.
func NewResource() resource.Resource {
	return &clusterResource{}
}

type clusterResource struct {
	client *client.Client
}

type clusterResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Region            types.String `tfsdk:"region"`
	ServerType        types.String `tfsdk:"server_type"`
	NodeCount         types.Int64  `tfsdk:"node_count"`
	DBName            types.String `tfsdk:"db_name"`
	DBType            types.String `tfsdk:"db_type"`
	PostgreSQLVersion types.Int64  `tfsdk:"postgresql_version"`
	Extensions        types.List   `tfsdk:"extensions"`
	SubscriptionID    types.Int64  `tfsdk:"subscription_id"`
	TenantID          types.String `tfsdk:"tenant_id"`
	Status            types.String `tfsdk:"status"`
	Host              types.String `tfsdk:"host"`
	ConnectionString  types.String `tfsdk:"connection_string"`
	DBUser            types.String `tfsdk:"db_user"`
	DBPassword        types.String `tfsdk:"db_password"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *clusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rivestack HA PostgreSQL cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Cluster ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Display name for the cluster.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: "Region for the cluster (e.g., eu-central, us-east).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("eu-central", "us-east"),
				},
			},
			"server_type": schema.StringAttribute{
				Description: "Server size: starter, growth, or scale.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("starter"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("starter", "growth", "scale"),
				},
			},
			"node_count": schema.Int64Attribute{
				Description: "Number of nodes (1-3).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
				Validators: []validator.Int64{
					int64validator.Between(1, 3),
				},
			},
			"db_name": schema.StringAttribute{
				Description: "Name of the default database.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("appdb"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"db_type": schema.StringAttribute{
				Description: "Cluster type: ha or core_solo.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("ha"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("ha", "core_solo"),
				},
			},
			"postgresql_version": schema.Int64Attribute{
				Description: "PostgreSQL major version.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(17),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"extensions": schema.ListAttribute{
				Description: "Additional PostgreSQL extensions to install at creation time.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"subscription_id": schema.Int64Attribute{
				Description: "Pool subscription ID to draw nodes from.",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Description: "Unique tenant identifier (rs-* prefix).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Cluster status.",
				Computed:    true,
			},
			"host": schema.StringAttribute{
				Description: "Cluster hostname for connections.",
				Computed:    true,
			},
			"connection_string": schema.StringAttribute{
				Description: "Full PostgreSQL connection string.",
				Computed:    true,
				Sensitive:   true,
			},
			"db_user": schema.StringAttribute{
				Description: "Default database user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"db_password": schema.StringAttribute{
				Description: "Default database user password.",
				Computed:    true,
				Sensitive:   true,
			},
			"created_at": schema.StringAttribute{
				Description: "Cluster creation timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Cluster last update timestamp.",
				Computed:    true,
			},
		},
	}
}

func (r *clusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	provisionReq := client.ProvisionClusterRequest{
		Name:              plan.Name.ValueString(),
		Region:            plan.Region.ValueString(),
		DBName:            plan.DBName.ValueString(),
		DBType:            plan.DBType.ValueString(),
		ServerType:        plan.ServerType.ValueString(),
		NodeCount:         int(plan.NodeCount.ValueInt64()),
		PostgreSQLVersion: int(plan.PostgreSQLVersion.ValueInt64()),
	}

	if !plan.SubscriptionID.IsNull() {
		subID := int(plan.SubscriptionID.ValueInt64())
		provisionReq.SubscriptionID = &subID
	}

	if !plan.Extensions.IsNull() {
		var exts []string
		resp.Diagnostics.Append(plan.Extensions.ElementsAs(ctx, &exts, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		provisionReq.Extensions = exts
	}

	tflog.Info(ctx, "Creating cluster", map[string]interface{}{
		"name":   provisionReq.Name,
		"region": provisionReq.Region,
	})

	provisionResp, err := r.client.ProvisionCluster(ctx, provisionReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster",
			fmt.Sprintf("Could not create cluster %q: %s", plan.Name.ValueString(), err))
		return
	}

	tflog.Info(ctx, "Waiting for cluster to become active", map[string]interface{}{
		"cluster_id": provisionResp.ID,
	})

	cluster, err := r.client.WaitForClusterActive(ctx, provisionResp.ID, 25*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for cluster",
			fmt.Sprintf("Cluster %d failed to become active: %s", provisionResp.ID, err))
		return
	}

	mapClusterToState(cluster, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster ID",
			fmt.Sprintf("Could not parse cluster ID %q: %s", state.ID.ValueString(), err))
		return
	}

	cluster, err := r.client.GetCluster(ctx, id)
	if err != nil {
		if client.IsNotFound(err) || client.IsGone(err) {
			tflog.Warn(ctx, "Cluster not found, removing from state", map[string]interface{}{"id": id})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading cluster",
			fmt.Sprintf("Could not read cluster %d: %s", id, err))
		return
	}

	// Preserve extensions from state since they are only used at creation.
	extensions := state.Extensions
	mapClusterToState(cluster, &state)
	state.Extensions = extensions

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster ID",
			fmt.Sprintf("Could not parse cluster ID %q: %s", state.ID.ValueString(), err))
		return
	}

	// Only node_count can change in-place.
	oldCount := state.NodeCount.ValueInt64()
	newCount := plan.NodeCount.ValueInt64()

	if newCount != oldCount {
		tflog.Info(ctx, "Scaling cluster nodes", map[string]interface{}{
			"cluster_id": id,
			"from":       oldCount,
			"to":         newCount,
		})

		if newCount > oldCount {
			for i := oldCount; i < newCount; i++ {
				_, err := r.client.AddNode(ctx, id)
				if err != nil {
					resp.Diagnostics.AddError("Error adding node",
						fmt.Sprintf("Could not add node to cluster %d: %s", id, err))
					return
				}
				if err := r.client.WaitForJobComplete(ctx, id, 10*time.Minute); err != nil {
					resp.Diagnostics.AddError("Error waiting for add-node job",
						fmt.Sprintf("Add-node job failed for cluster %d: %s", id, err))
					return
				}
			}
		} else {
			// Get cluster details to find node names for removal.
			cluster, err := r.client.GetCluster(ctx, id)
			if err != nil {
				resp.Diagnostics.AddError("Error reading cluster",
					fmt.Sprintf("Could not read cluster %d for node removal: %s", id, err))
				return
			}
			// Remove nodes from highest number down.
			for i := oldCount; i > newCount; i-- {
				nodeName := fmt.Sprintf("%s-db-%d", cluster.TenantID, i)
				_, err := r.client.RemoveNode(ctx, id, nodeName)
				if err != nil {
					resp.Diagnostics.AddError("Error removing node",
						fmt.Sprintf("Could not remove node %s from cluster %d: %s", nodeName, id, err))
					return
				}
				if err := r.client.WaitForJobComplete(ctx, id, 10*time.Minute); err != nil {
					resp.Diagnostics.AddError("Error waiting for remove-node job",
						fmt.Sprintf("Remove-node job failed for cluster %d: %s", id, err))
					return
				}
			}
		}
	}

	// Refresh state from API.
	cluster, err := r.client.GetCluster(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading cluster after update",
			fmt.Sprintf("Could not read cluster %d: %s", id, err))
		return
	}

	extensions := plan.Extensions
	subscriptionID := plan.SubscriptionID
	mapClusterToState(cluster, &plan)
	plan.Extensions = extensions
	plan.SubscriptionID = subscriptionID

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster ID",
			fmt.Sprintf("Could not parse cluster ID %q: %s", state.ID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "Deleting cluster", map[string]interface{}{"id": id})

	err = r.client.DeleteCluster(ctx, id)
	if err != nil {
		if client.IsNotFound(err) || client.IsGone(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting cluster",
			fmt.Sprintf("Could not delete cluster %d: %s", id, err))
		return
	}

	err = r.client.WaitForClusterDeleted(ctx, id, 10*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for cluster deletion",
			fmt.Sprintf("Cluster %d did not finish deleting: %s", id, err))
		return
	}
}

func (r *clusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapClusterToState(c *client.Cluster, state *clusterResourceModel) {
	state.ID = types.StringValue(strconv.Itoa(c.ID))
	state.Name = types.StringValue(c.Name)
	state.Region = types.StringValue(c.Region)
	state.ServerType = types.StringValue(c.ServerType)
	state.NodeCount = types.Int64Value(int64(c.NodeCount))
	state.DBName = types.StringValue(c.DBName)
	state.DBType = types.StringValue(c.DBType)
	state.PostgreSQLVersion = types.Int64Value(int64(c.PostgreSQLVersion))
	state.TenantID = types.StringValue(c.TenantID)
	state.Status = types.StringValue(c.Status)
	state.Host = types.StringValue(c.Host)
	state.ConnectionString = types.StringValue(c.ConnectionString)
	state.DBUser = types.StringValue(c.DBUser)
	state.DBPassword = types.StringValue(c.DBPassword)
	state.CreatedAt = types.StringValue(c.CreatedAt.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(c.UpdatedAt.Format(time.RFC3339))
}
