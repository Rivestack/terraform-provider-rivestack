// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package cluster_grant

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/rivestack/terraform-provider-rivestack/internal/client"
)

var (
	_ resource.Resource                = &clusterGrantResource{}
	_ resource.ResourceWithImportState = &clusterGrantResource{}
)

func NewResource() resource.Resource {
	return &clusterGrantResource{}
}

type clusterGrantResource struct {
	client *client.Client
}

type clusterGrantResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Username  types.String `tfsdk:"username"`
	Database  types.String `tfsdk:"database"`
	Access    types.String `tfsdk:"access"`
}

func (r *clusterGrantResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_grant"
}

func (r *clusterGrantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an access grant on a Rivestack HA PostgreSQL cluster. Note: grant revocation is not currently supported by the API; destroying this resource removes it from Terraform state only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier (cluster_id/username/database).",
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
			"username": schema.StringAttribute{
				Description: "Username to grant access to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database": schema.StringAttribute{
				Description: "Database to grant access on.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access": schema.StringAttribute{
				Description: "Access level: read (SELECT only) or write (SELECT, INSERT, UPDATE, DELETE).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("write"),
				Validators: []validator.String{
					stringvalidator.OneOf("read", "write"),
				},
			},
		},
	}
}

func (r *clusterGrantResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *clusterGrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterGrantResourceModel
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

	grantReq := client.ConfigGrantRequest{
		Username: plan.Username.ValueString(),
		Database: plan.Database.ValueString(),
		Access:   plan.Access.ValueString(),
	}

	tflog.Info(ctx, "Creating cluster grant", map[string]interface{}{
		"cluster_id": clusterID,
		"username":   plan.Username.ValueString(),
		"database":   plan.Database.ValueString(),
		"access":     plan.Access.ValueString(),
	})

	configResp, err := r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		Grants: []client.ConfigGrantRequest{grantReq},
	}, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster grant",
			fmt.Sprintf("Could not create grant on cluster %d: %s", clusterID, err))
		return
	}

	if configResp.JobID > 0 {
		if err := r.client.WaitForJobComplete(ctx, clusterID, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError("Error waiting for grant creation",
				fmt.Sprintf("Configure job failed for cluster %d: %s", clusterID, err))
			return
		}
	}

	plan.ID = types.StringValue(fmt.Sprintf("%d/%s/%s", clusterID, plan.Username.ValueString(), plan.Database.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterGrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterGrantResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID, username, database, err := parseGrantID(state.ID.ValueString())
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
	for _, g := range cluster.Grants {
		if g.Username == username && g.Database == database {
			state.Access = types.StringValue(g.Access)
			found = true
			break
		}
	}

	if !found {
		tflog.Warn(ctx, "Cluster grant not found, removing from state", map[string]interface{}{
			"cluster_id": clusterID,
			"username":   username,
			"database":   database,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clusterGrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clusterGrantResourceModel
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

	// Re-apply grant with updated access level (ON CONFLICT DO UPDATE).
	grantReq := client.ConfigGrantRequest{
		Username: plan.Username.ValueString(),
		Database: plan.Database.ValueString(),
		Access:   plan.Access.ValueString(),
	}

	configResp, err := r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		Grants: []client.ConfigGrantRequest{grantReq},
	}, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error updating cluster grant",
			fmt.Sprintf("Could not update grant on cluster %d: %s", clusterID, err))
		return
	}

	if configResp.JobID > 0 {
		if err := r.client.WaitForJobComplete(ctx, clusterID, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError("Error waiting for grant update",
				fmt.Sprintf("Configure job failed for cluster %d: %s", clusterID, err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterGrantResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Grant revocation is not currently supported by the Rivestack API.
	// Removing from Terraform state only.
	tflog.Warn(ctx, "Grant revocation is not supported by the Rivestack API. The grant remains on the cluster but is removed from Terraform state.")
}

func (r *clusterGrantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Import ID must be in the format: cluster_id/username/database")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database"), parts[2])...)
}

func parseGrantID(id string) (int, string, string, error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 {
		return 0, "", "", fmt.Errorf("expected format: cluster_id/username/database")
	}
	clusterID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid cluster ID: %w", err)
	}
	return clusterID, parts[1], parts[2], nil
}
