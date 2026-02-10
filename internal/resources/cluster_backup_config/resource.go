// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package cluster_backup_config

import (
	"context"
	"fmt"
	"strconv"
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
	_ resource.Resource                = &clusterBackupConfigResource{}
	_ resource.ResourceWithImportState = &clusterBackupConfigResource{}
)

func NewResource() resource.Resource {
	return &clusterBackupConfigResource{}
}

type clusterBackupConfigResource struct {
	client *client.Client
}

type clusterBackupConfigResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ClusterID     types.String `tfsdk:"cluster_id"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Schedule      types.String `tfsdk:"schedule"`
	RetentionFull types.Int64  `tfsdk:"retention_full"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *clusterBackupConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_backup_config"
}

func (r *clusterBackupConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages backup configuration for a Rivestack HA PostgreSQL cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier (cluster_id).",
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
			"enabled": schema.BoolAttribute{
				Description: "Whether automated backups are enabled.",
				Required:    true,
			},
			"schedule": schema.StringAttribute{
				Description: "Cron schedule for automated backups (e.g., \"0 3 * * *\" for daily at 3 AM).",
				Optional:    true,
				Computed:    true,
			},
			"retention_full": schema.Int64Attribute{
				Description: "Number of days to retain full backups.",
				Optional:    true,
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Last update timestamp.",
				Computed:    true,
			},
		},
	}
}

func (r *clusterBackupConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *clusterBackupConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterBackupConfigResourceModel
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

	updateReq := buildUpdateRequest(plan)

	tflog.Info(ctx, "Setting cluster backup config", map[string]interface{}{
		"cluster_id": clusterID,
	})

	config, err := r.client.UpdateBackupConfig(ctx, clusterID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error setting backup config",
			fmt.Sprintf("Could not set backup config on cluster %d: %s", clusterID, err))
		return
	}

	mapBackupConfigToState(config, &plan)
	plan.ID = types.StringValue(plan.ClusterID.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterBackupConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterBackupConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID, err := strconv.Atoi(state.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster ID",
			fmt.Sprintf("Could not parse cluster ID %q: %s", state.ClusterID.ValueString(), err))
		return
	}

	config, err := r.client.GetBackupConfig(ctx, clusterID)
	if err != nil {
		if client.IsNotFound(err) || client.IsGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading backup config",
			fmt.Sprintf("Could not read backup config for cluster %d: %s", clusterID, err))
		return
	}

	mapBackupConfigToState(config, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clusterBackupConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clusterBackupConfigResourceModel
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

	updateReq := buildUpdateRequest(plan)

	tflog.Info(ctx, "Updating cluster backup config", map[string]interface{}{
		"cluster_id": clusterID,
	})

	config, err := r.client.UpdateBackupConfig(ctx, clusterID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating backup config",
			fmt.Sprintf("Could not update backup config on cluster %d: %s", clusterID, err))
		return
	}

	mapBackupConfigToState(config, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterBackupConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clusterBackupConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID, err := strconv.Atoi(state.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cluster ID",
			fmt.Sprintf("Could not parse cluster ID %q: %s", state.ClusterID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "Disabling cluster backups", map[string]interface{}{
		"cluster_id": clusterID,
	})

	// Reset to disabled.
	enabled := false
	_, err = r.client.UpdateBackupConfig(ctx, clusterID, client.UpdateBackupConfigRequest{
		Enabled: &enabled,
	})
	if err != nil {
		if client.IsNotFound(err) || client.IsGone(err) {
			return
		}
		resp.Diagnostics.AddError("Error disabling backups",
			fmt.Sprintf("Could not disable backups on cluster %d: %s", clusterID, err))
		return
	}
}

func (r *clusterBackupConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), req.ID)...)
}

func buildUpdateRequest(plan clusterBackupConfigResourceModel) client.UpdateBackupConfigRequest {
	enabled := plan.Enabled.ValueBool()
	req := client.UpdateBackupConfigRequest{
		Enabled: &enabled,
	}
	if !plan.Schedule.IsNull() && !plan.Schedule.IsUnknown() {
		req.Schedule = plan.Schedule.ValueString()
	}
	if !plan.RetentionFull.IsNull() && !plan.RetentionFull.IsUnknown() {
		ret := int(plan.RetentionFull.ValueInt64())
		req.RetentionFull = &ret
	}
	return req
}

func mapBackupConfigToState(config *client.BackupConfig, state *clusterBackupConfigResourceModel) {
	state.Enabled = types.BoolValue(config.Enabled)
	state.Schedule = types.StringValue(config.Schedule)
	state.RetentionFull = types.Int64Value(int64(config.RetentionFull))
	state.UpdatedAt = types.StringValue(config.UpdatedAt.Format(time.RFC3339))
}
