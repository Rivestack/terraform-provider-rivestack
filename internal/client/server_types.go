// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package client

import "context"

// GetServerTypes retrieves available server types.
func (c *Client) GetServerTypes(ctx context.Context) (*ServerTypesResponse, error) {
	var resp ServerTypesResponse
	err := c.doRequest(ctx, "GET", "/api/ha/server-types", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
