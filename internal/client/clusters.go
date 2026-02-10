// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"time"
)

// ProvisionCluster creates a new HA cluster.
func (c *Client) ProvisionCluster(ctx context.Context, req ProvisionClusterRequest) (*ProvisionClusterResponse, error) {
	var resp ProvisionClusterResponse
	err := c.doRequest(ctx, "POST", "/api/ha/provision", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCluster retrieves a cluster by ID.
func (c *Client) GetCluster(ctx context.Context, id int) (*Cluster, error) {
	var cluster Cluster
	err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/ha/%d", id), nil, &cluster)
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// ListClusters lists all clusters for the authenticated user.
func (c *Client) ListClusters(ctx context.Context) ([]Cluster, error) {
	var resp ClusterListResponse
	err := c.doRequest(ctx, "GET", "/api/ha", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Clusters, nil
}

// DeleteCluster initiates deletion of a cluster.
func (c *Client) DeleteCluster(ctx context.Context, id int) error {
	return c.doRequest(ctx, "DELETE", fmt.Sprintf("/api/ha/%d", id), nil, nil)
}

// WaitForClusterActive polls the cluster until it reaches "active" or "failed" status.
func (c *Client) WaitForClusterActive(ctx context.Context, id int, timeout time.Duration) (*Cluster, error) {
	deadline := time.Now().Add(timeout)
	pollInterval := 15 * time.Second

	for time.Now().Before(deadline) {
		cluster, err := c.GetCluster(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("polling cluster status: %w", err)
		}

		switch cluster.Status {
		case "active":
			return cluster, nil
		case "failed":
			return nil, fmt.Errorf("cluster provisioning failed: %s", cluster.ErrorMessage)
		case "provisioning":
			// Continue polling.
		default:
			return nil, fmt.Errorf("unexpected cluster status: %s", cluster.Status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return nil, fmt.Errorf("timeout waiting for cluster to become active after %s", timeout)
}

// WaitForClusterDeleted polls the cluster until it is deleted or gone.
func (c *Client) WaitForClusterDeleted(ctx context.Context, id int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 10 * time.Second

	for time.Now().Before(deadline) {
		cluster, err := c.GetCluster(ctx, id)
		if err != nil {
			if IsNotFound(err) || IsGone(err) {
				return nil
			}
			return fmt.Errorf("polling cluster deletion status: %w", err)
		}

		if cluster.Status == "deleted" {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timeout waiting for cluster to be deleted after %s", timeout)
}
