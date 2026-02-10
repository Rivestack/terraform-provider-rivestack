// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package cluster_firewall

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
	_ resource.Resource                = &clusterFirewallResource{}
	_ resource.ResourceWithImportState = &clusterFirewallResource{}
)

func NewResource() resource.Resource {
	return &clusterFirewallResource{}
}

type clusterFirewallResource struct {
	client *client.Client
}

type clusterFirewallResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ClusterID types.String `tfsdk:"cluster_id"`
	SourceIPs types.Set    `tfsdk:"source_ips"`
}

func (r *clusterFirewallResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_firewall"
}

func (r *clusterFirewallResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages firewall rules (IP allowlist) for a Rivestack HA PostgreSQL cluster.",
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
			"source_ips": schema.SetAttribute{
				Description: "Set of IP addresses or CIDR ranges allowed to connect. Use [\"0.0.0.0/0\"] for unrestricted access.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *clusterFirewallResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *clusterFirewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterFirewallResourceModel
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

	var sourceIPs []string
	resp.Diagnostics.Append(plan.SourceIPs.ElementsAs(ctx, &sourceIPs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Setting cluster firewall rules", map[string]interface{}{
		"cluster_id": clusterID,
		"source_ips": sourceIPs,
	})

	_, err = r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		SourceIPs:  sourceIPs,
		ReplaceIPs: true,
	}, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error setting cluster firewall",
			fmt.Sprintf("Could not set firewall rules on cluster %d: %s", clusterID, err))
		return
	}

	plan.ID = types.StringValue(plan.ClusterID.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterFirewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterFirewallResourceModel
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

	// Parse source_ips from comma-separated string.
	var ips []string
	if cluster.SourceIPs != "" {
		for _, ip := range strings.Split(cluster.SourceIPs, ",") {
			trimmed := strings.TrimSpace(ip)
			if trimmed != "" {
				ips = append(ips, trimmed)
			}
		}
	}

	ipSet, diags := types.SetValueFrom(ctx, types.StringType, ips)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.SourceIPs = ipSet

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clusterFirewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clusterFirewallResourceModel
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

	var sourceIPs []string
	resp.Diagnostics.Append(plan.SourceIPs.ElementsAs(ctx, &sourceIPs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating cluster firewall rules", map[string]interface{}{
		"cluster_id": clusterID,
		"source_ips": sourceIPs,
	})

	_, err = r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		SourceIPs:  sourceIPs,
		ReplaceIPs: true,
	}, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error updating cluster firewall",
			fmt.Sprintf("Could not update firewall rules on cluster %d: %s", clusterID, err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterFirewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clusterFirewallResourceModel
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

	tflog.Info(ctx, "Resetting cluster firewall to allow all", map[string]interface{}{
		"cluster_id": clusterID,
	})

	// Reset to allow all traffic.
	_, err = r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		SourceIPs:  []string{"0.0.0.0/0"},
		ReplaceIPs: true,
	}, 2*time.Minute)
	if err != nil {
		if client.IsNotFound(err) || client.IsGone(err) {
			return
		}
		resp.Diagnostics.AddError("Error resetting cluster firewall",
			fmt.Sprintf("Could not reset firewall on cluster %d: %s", clusterID, err))
		return
	}
}

func (r *clusterFirewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), req.ID)...)
}
