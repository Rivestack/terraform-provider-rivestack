// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package client

import "context"

// GetExtensions retrieves available PostgreSQL extensions.
func (c *Client) GetExtensions(ctx context.Context) (*ExtensionsResponse, error) {
	var resp ExtensionsResponse
	err := c.doRequest(ctx, "GET", "/api/ha/extensions", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
