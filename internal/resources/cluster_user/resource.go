// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package cluster_user

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
	_ resource.Resource                = &clusterUserResource{}
	_ resource.ResourceWithImportState = &clusterUserResource{}
)

func NewResource() resource.Resource {
	return &clusterUserResource{}
}

type clusterUserResource struct {
	client *client.Client
}

type clusterUserResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
}

func (r *clusterUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_user"
}

func (r *clusterUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a database user on a Rivestack HA PostgreSQL cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource identifier (cluster_id/username).",
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
				Description: "PostgreSQL username. Must start with a letter or underscore and contain only letters, numbers, and underscores (max 63 characters).",
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
			"password": schema.StringAttribute{
				Description: "Auto-generated password for the user.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *clusterUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *clusterUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterUserResourceModel
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

	username := plan.Username.ValueString()

	tflog.Info(ctx, "Creating cluster user", map[string]interface{}{
		"cluster_id": clusterID,
		"username":   username,
	})

	configResp, err := r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		Users: []client.ConfigUserRequest{{Username: username}},
	}, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster user",
			fmt.Sprintf("Could not create user %q on cluster %d: %s", username, clusterID, err))
		return
	}

	// Wait for the configure job to complete.
	if configResp.JobID > 0 {
		if err := r.client.WaitForJobComplete(ctx, clusterID, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError("Error waiting for user creation",
				fmt.Sprintf("Configure job failed for cluster %d: %s", clusterID, err))
			return
		}
	}

	plan.ID = types.StringValue(fmt.Sprintf("%d/%s", clusterID, username))

	// Extract password from response.
	for _, u := range configResp.Users {
		if u.Username == username {
			plan.Password = types.StringValue(u.Password)
			break
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clusterUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID, username, err := parseUserID(state.ID.ValueString())
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
	for _, u := range cluster.Users {
		if u.Username == username {
			found = true
			break
		}
	}

	if !found {
		tflog.Warn(ctx, "Cluster user not found, removing from state", map[string]interface{}{
			"cluster_id": clusterID,
			"username":   username,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	// Username and cluster_id don't change; password is only returned at creation.
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *clusterUserResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes are ForceNew, so Update is never called.
	resp.Diagnostics.AddError("Update not supported", "Cluster user attributes cannot be updated in-place.")
}

func (r *clusterUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clusterUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID, username, err := parseUserID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID",
			fmt.Sprintf("Could not parse resource ID %q: %s", state.ID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "Deleting cluster user", map[string]interface{}{
		"cluster_id": clusterID,
		"username":   username,
	})

	configResp, err := r.client.ConfigureWithRetry(ctx, clusterID, client.ConfigureRequest{
		DeleteUsers: []string{username},
	}, 2*time.Minute)
	if err != nil {
		if client.IsNotFound(err) || client.IsGone(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting cluster user",
			fmt.Sprintf("Could not delete user %q from cluster %d: %s", username, clusterID, err))
		return
	}

	if configResp.JobID > 0 {
		if err := r.client.WaitForJobComplete(ctx, clusterID, 5*time.Minute); err != nil {
			resp.Diagnostics.AddError("Error waiting for user deletion",
				fmt.Sprintf("Configure job failed for cluster %d: %s", clusterID, err))
			return
		}
	}
}

func (r *clusterUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: cluster_id/username
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Import ID must be in the format: cluster_id/username")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), parts[1])...)
	// Password cannot be imported; it will be unknown.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), types.StringValue(""))...)
}

func parseUserID(id string) (int, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("expected format: cluster_id/username")
	}
	clusterID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid cluster ID: %w", err)
	}
	return clusterID, parts[1], nil
}
