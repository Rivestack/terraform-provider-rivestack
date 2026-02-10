// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package cluster_database

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/rivestack/terraform-provider-rivestack/internal/client"
)

func pgIdentifierRegex() *regexp.Regexp {
	return regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
}

var (
	_ resource.Resource                = &clusterDatabaseResource{}
	_ resource.ResourceWithImportState = &clusterDatabaseResource{}
)

func NewResource() resource.Resource {
	return &clusterDatabaseResource{}
}

type clusterDatabaseResource struct {
	client *client.Client
}

type clusterDatabaseResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Name      types.String `tfsdk:"name"`
	Owner     types.String `tfsdk:"owner"`
}

func (r *clusterDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_database"
}

func (r *clusterDatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a database on a Rivestack HA PostgreSQL cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier (cluster_id/name).",
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
			"name": schema.StringAttribute{
				Description: "Database name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 63),
					stringvalidator.RegexMatches(
						pgIdentifierRegex(),
						"must start with a letter or underscore and contain only letters, numbers, and underscores",
					),
				},
			},
			"owner": schema.StringAttribute{
				Description: "Database owner username. Defaults to the cluster's default user.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *clusterDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *clusterDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterDatabaseResourceModel
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

	dbName := plan.Name.ValueString()
	dbReq := client.ConfigDatabaseRequest{Name: dbName}
	if !plan.Owner.IsNull() && !plan.Owner.IsUnknown() {
		dbReq.Owner = plan.Owner.ValueString()
	}

	tflog.Info(ctx, "Creating cluster database", map[string]interface{}{
		"cluster_id": clusterID,
		"name":       dbName,
	})

	configResp, err := r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		Databases: []client.ConfigDatabaseRequest{dbReq},
	}, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster database",
			fmt.Sprintf("Could not create database %q on cluster %d: %s", dbName, clusterID, err))
		return
	}

	if configResp.JobID > 0 {
		if err := r.client.WaitForJobComplete(ctx, clusterID, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError("Error waiting for database creation",
				fmt.Sprintf("Configure job failed for cluster %d: %s", clusterID, err))
			return
		}
	}

	plan.ID = types.StringValue(fmt.Sprintf("%d/%s", clusterID, dbName))

	// Extract owner from configure response first.
	for _, db := range configResp.Databases {
		if db.Name == dbName {
			plan.Owner = types.StringValue(db.Owner)
			break
		}
	}

	// If owner is still unknown, read back from the cluster.
	if plan.Owner.IsNull() || plan.Owner.IsUnknown() {
		cluster, err := r.client.GetCluster(ctx, clusterID)
		if err == nil {
			for _, db := range cluster.Databases {
				if db.DBName == dbName {
					plan.Owner = types.StringValue(db.Owner)
					break
				}
			}
		}
	}

	// Final fallback: set owner to empty string rather than leaving it unknown.
	if plan.Owner.IsNull() || plan.Owner.IsUnknown() {
		plan.Owner = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID, dbName, err := parseDatabaseID(state.ID.ValueString())
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
	for _, db := range cluster.Databases {
		if db.DBName == dbName {
			state.Owner = types.StringValue(db.Owner)
			found = true
			break
		}
	}

	if !found {
		tflog.Warn(ctx, "Cluster database not found, removing from state", map[string]interface{}{
			"cluster_id": clusterID,
			"name":       dbName,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clusterDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clusterDatabaseResourceModel
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

	// Owner can be updated via the configure endpoint (ON CONFLICT DO UPDATE).
	dbReq := client.ConfigDatabaseRequest{
		Name:  plan.Name.ValueString(),
		Owner: plan.Owner.ValueString(),
	}

	configResp, err := r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		Databases: []client.ConfigDatabaseRequest{dbReq},
	}, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error updating cluster database",
			fmt.Sprintf("Could not update database %q on cluster %d: %s", plan.Name.ValueString(), clusterID, err))
		return
	}

	if configResp.JobID > 0 {
		if err := r.client.WaitForJobComplete(ctx, clusterID, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError("Error waiting for database update",
				fmt.Sprintf("Configure job failed for cluster %d: %s", clusterID, err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clusterDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID, dbName, err := parseDatabaseID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID",
			fmt.Sprintf("Could not parse resource ID %q: %s", state.ID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "Deleting cluster database", map[string]interface{}{
		"cluster_id": clusterID,
		"name":       dbName,
	})

	configResp, err := r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		DeleteDatabases: []string{dbName},
	}, 2*time.Minute)
	if err != nil {
		if client.IsNotFound(err) || client.IsGone(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting cluster database",
			fmt.Sprintf("Could not delete database %q from cluster %d: %s", dbName, clusterID, err))
		return
	}

	if configResp.JobID > 0 {
		if err := r.client.WaitForJobComplete(ctx, clusterID, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError("Error waiting for database deletion",
				fmt.Sprintf("Configure job failed for cluster %d: %s", clusterID, err))
			return
		}
	}
}

func (r *clusterDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Import ID must be in the format: cluster_id/database_name")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

func parseDatabaseID(id string) (int, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("expected format: cluster_id/database_name")
	}
	clusterID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid cluster ID: %w", err)
	}
	return clusterID, parts[1], nil
}
