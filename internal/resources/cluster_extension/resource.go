// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package cluster_extension

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/rivestack/terraform-provider-rivestack/internal/client"
)

var (
	_ resource.Resource                = &clusterExtensionResource{}
	_ resource.ResourceWithImportState = &clusterExtensionResource{}
)

func NewResource() resource.Resource {
	return &clusterExtensionResource{}
}

type clusterExtensionResource struct {
	client *client.Client
}

type clusterExtensionResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Extension types.String `tfsdk:"extension"`
	Database  types.String `tfsdk:"database"`
}

func (r *clusterExtensionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_extension"
}

func (r *clusterExtensionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a PostgreSQL extension on a Rivestack HA cluster. Note: extensions cannot be removed from a running cluster; destroying this resource removes it from Terraform state only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier (cluster_id/extension/database).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Description: "ID of the cluster.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"extension": schema.StringAttribute{
				Description: "PostgreSQL extension name (e.g., vector, postgis, uuid-ossp).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database": schema.StringAttribute{
				Description: "Database to install the extension on. Defaults to the cluster's default database.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *clusterExtensionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *clusterExtensionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterExtensionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID, err := strconv.Atoi(plan.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster ID",
			fmt.Sprintf("Could not parse cluster ID %q: %s", plan.ClusterID.ValueString(), err))
		return
	}

	extReq := client.ConfigExtensionRequest{
		Extension: plan.Extension.ValueString(),
	}
	if !plan.Database.IsNull() && !plan.Database.IsUnknown() {
		extReq.Database = plan.Database.ValueString()
	}

	tflog.Info(ctx, "Creating cluster extension", map[string]interface{}{
		"cluster_id": clusterID,
		"extension":  plan.Extension.ValueString(),
	})

	configResp, err := r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		Extensions: []client.ConfigExtensionRequest{extReq},
	}, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster extension",
			fmt.Sprintf("Could not install extension %q on cluster %d: %s", plan.Extension.ValueString(), clusterID, err))
		return
	}

	if configResp.JobID > 0 {
		if err := r.client.WaitForJobComplete(ctx, clusterID, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError("Error waiting for extension installation",
				fmt.Sprintf("Configure job failed for cluster %d: %s", clusterID, err))
			return
		}
	}

	// Try to get database from configure response first.
	database := ""
	for _, ext := range configResp.Extensions {
		if ext.Extension == plan.Extension.ValueString() {
			database = ext.Database
			break
		}
	}

	// Fall back to reading from the cluster.
	if database == "" {
		cluster, err := r.client.GetCluster(ctx, clusterID)
		if err == nil {
			for _, ext := range cluster.Extensions {
				if ext.Extension == plan.Extension.ValueString() {
					database = ext.Database
					break
				}
			}
		}
	}

	// Final fallback: use the plan value or cluster default db.
	if database == "" {
		if !plan.Database.IsNull() && !plan.Database.IsUnknown() {
			database = plan.Database.ValueString()
		} else {
			// Read cluster to get default db_name.
			cluster, err := r.client.GetCluster(ctx, clusterID)
			if err == nil {
				database = cluster.DBName
			}
		}
	}

	plan.Database = types.StringValue(database)
	plan.ID = types.StringValue(fmt.Sprintf("%d/%s/%s", clusterID, plan.Extension.ValueString(), database))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterExtensionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterExtensionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID, extName, dbName, err := parseExtensionID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID",
			fmt.Sprintf("Could not parse resource ID %q: %s", state.ID.ValueString(), err))
		return
	}

	cluster, err := r.client.GetCluster(ctx, clusterID)
	if err != nil {
		if client.IsNotFound(err) || client.IsGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading cluster",
			fmt.Sprintf("Could not read cluster %d: %s", clusterID, err))
		return
	}

	found := false
	for _, ext := range cluster.Extensions {
		if ext.Extension == extName && ext.Database == dbName {
			found = true
			break
		}
	}

	if !found {
		tflog.Warn(ctx, "Cluster extension not found, removing from state", map[string]interface{}{
			"cluster_id": clusterID,
			"extension":  extName,
			"database":   dbName,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clusterExtensionResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Cluster extension attributes cannot be updated in-place.")
}

func (r *clusterExtensionResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Extensions cannot be removed from a running cluster via the API.
	// Removing from Terraform state only.
	tflog.Warn(ctx, "Extension removal is not supported by the Rivestack API. The extension remains installed on the cluster but is removed from Terraform state.")
}

func (r *clusterExtensionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Import ID must be in the format: cluster_id/extension/database")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("extension"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database"), parts[2])...)
}

func parseExtensionID(id string) (int, string, string, error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 {
		return 0, "", "", fmt.Errorf("expected format: cluster_id/extension/database")
	}
	clusterID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid cluster ID: %w", err)
	}
	return clusterID, parts[1], parts[2], nil
}
