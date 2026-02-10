// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"time"
)

// ConfigureCluster sends a configuration request to the cluster.
func (c *Client) ConfigureCluster(ctx context.Context, clusterID int, req ConfigureRequest) (*ConfigureResponse, error) {
	var resp ConfigureResponse
	err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/ha/%d/configure", clusterID), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ConfigureWithRetry sends a configuration request, retrying on 409 Conflict
// (cluster has an active job). Retries with 10s backoff for up to maxWait.
func (c *Client) ConfigureWithRetry(ctx context.Context, clusterID int, req ConfigureRequest, maxWait time.Duration) (*ConfigureResponse, error) {
	deadline := time.Now().Add(maxWait)
	retryInterval := 10 * time.Second

	for {
		resp, err := c.ConfigureCluster(ctx, clusterID, req)
		if err == nil {
			return resp, nil
		}

		if !IsConflict(err) {
			return nil, err
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for cluster to be available for configuration: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryInterval):
		}
	}
}

// WaitForJobComplete polls the cluster's active jobs until none are active.
func (c *Client) WaitForJobComplete(ctx context.Context, clusterID int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 10 * time.Second

	for time.Now().Before(deadline) {
		var resp JobsResponse
		err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/ha/%d/jobs?active=true", clusterID), nil, &resp)
		if err != nil {
			return fmt.Errorf("polling job status: %w", err)
		}

		if len(resp.Jobs) == 0 {
			return nil
		}

		// Check if any job has failed.
		for _, job := range resp.Jobs {
			if job.Status == "failed" {
				return fmt.Errorf("cluster job %d (%s) failed: %s", job.ID, job.JobType, job.ErrorMessage)
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timeout waiting for cluster jobs to complete after %s", timeout)
}
