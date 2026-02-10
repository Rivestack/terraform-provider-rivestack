// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
)

// AddNode adds a node to the cluster.
func (c *Client) AddNode(ctx context.Context, clusterID int) (*AddNodeResponse, error) {
	var resp AddNodeResponse
	err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/ha/%d/add-node", clusterID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// RemoveNode removes a node from the cluster.
func (c *Client) RemoveNode(ctx context.Context, clusterID int, nodeName string) (*RemoveNodeResponse, error) {
	req := RemoveNodeRequest{
		NodeName:           nodeName,
		DeleteServer:       true,
		RemovePostgresData: true,
	}
	var resp RemoveNodeResponse
	err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/ha/%d/remove-node", clusterID), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
