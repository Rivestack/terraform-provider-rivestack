// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
)

// GetBackupConfig retrieves the backup configuration for a cluster.
func (c *Client) GetBackupConfig(ctx context.Context, clusterID int) (*BackupConfig, error) {
	var config BackupConfig
	err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/ha/%d/backup-config", clusterID), nil, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateBackupConfig updates the backup configuration for a cluster.
func (c *Client) UpdateBackupConfig(ctx context.Context, clusterID int, req UpdateBackupConfigRequest) (*BackupConfig, error) {
	var config BackupConfig
	err := c.doRequest(ctx, "PUT", fmt.Sprintf("/api/ha/%d/backup-config", clusterID), req, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
