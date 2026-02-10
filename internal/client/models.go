// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package client

import "time"

// ProvisionClusterRequest is the request body for provisioning a new HA cluster.
type ProvisionClusterRequest struct {
	Name              string   `json:"name"`
	Region            string   `json:"region"`
	DBName            string   `json:"db_name,omitempty"`
	DBType            string   `json:"db_type,omitempty"`
	ServerType        string   `json:"server_type,omitempty"`
	NodeCount         int      `json:"node_count,omitempty"`
	PostgreSQLVersion int      `json:"postgresql_version,omitempty"`
	Extensions        []string `json:"extensions,omitempty"`
	SubscriptionID    *int     `json:"subscription_id,omitempty"`
}

// ProvisionClusterResponse is the response from provisioning a cluster.
type ProvisionClusterResponse struct {
	ID             int       `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name"`
	Region         string    `json:"region"`
	Status         string    `json:"status"`
	StreamURL      string    `json:"stream_url"`
	SubscriptionID *int      `json:"subscription_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// Cluster represents a full HA cluster with all its details.
type Cluster struct {
	ID                int                `json:"id"`
	TenantID          string             `json:"tenant_id"`
	Name              string             `json:"name"`
	Region            string             `json:"region"`
	DBType            string             `json:"db_type"`
	ServerType        string             `json:"server_type"`
	NodeCount         int                `json:"node_count"`
	PostgreSQLVersion int                `json:"postgresql_version"`
	DBName            string             `json:"db_name"`
	DBUser            string             `json:"db_user"`
	DBPassword        string             `json:"db_password"`
	Host              string             `json:"host"`
	ConnectionString  string             `json:"connection_string"`
	Status            string             `json:"status"`
	HealthStatus      string             `json:"health_status"`
	SourceIPs         string             `json:"source_ips"`
	ErrorMessage      string             `json:"error_message"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	Users             []ClusterUser      `json:"users"`
	Databases         []ClusterDatabase  `json:"databases"`
	Extensions        []ClusterExtension `json:"extensions"`
	Grants            []ClusterGrant     `json:"grants"`
	BackupConfig      *BackupConfig      `json:"backup_config"`
}

// ClusterUser represents a database user on a cluster.
type ClusterUser struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
}

// ClusterDatabase represents a database on a cluster.
type ClusterDatabase struct {
	DBName string `json:"db_name"`
	Owner  string `json:"owner"`
}

// ClusterExtension represents a PostgreSQL extension installed on a cluster database.
type ClusterExtension struct {
	Extension string `json:"extension"`
	Database  string `json:"database"`
}

// ClusterGrant represents an access grant on a cluster.
type ClusterGrant struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Database  string    `json:"database"`
	Access    string    `json:"access"`
	CreatedAt time.Time `json:"created_at"`
}

// ClusterListResponse is the response from listing clusters.
type ClusterListResponse struct {
	Clusters []Cluster `json:"clusters"`
}

// ConfigureRequest is the request body for the unified configure endpoint.
type ConfigureRequest struct {
	Users           []ConfigUserRequest      `json:"users,omitempty"`
	DeleteUsers     []string                 `json:"delete_users,omitempty"`
	Databases       []ConfigDatabaseRequest  `json:"databases,omitempty"`
	DeleteDatabases []string                 `json:"delete_databases,omitempty"`
	Extensions      []ConfigExtensionRequest `json:"extensions,omitempty"`
	Grants          []ConfigGrantRequest     `json:"grants,omitempty"`
	SourceIPs       []string                 `json:"source_ips,omitempty"`
	DeleteIPs       []string                 `json:"delete_ips,omitempty"`
	ReplaceIPs      bool                     `json:"replace_ips,omitempty"`
}

// ConfigUserRequest is a user creation request within ConfigureRequest.
type ConfigUserRequest struct {
	Username string `json:"username"`
}

// ConfigDatabaseRequest is a database creation request within ConfigureRequest.
type ConfigDatabaseRequest struct {
	Name  string `json:"name"`
	Owner string `json:"owner,omitempty"`
}

// ConfigExtensionRequest is an extension installation request within ConfigureRequest.
type ConfigExtensionRequest struct {
	Extension string `json:"extension"`
	Database  string `json:"database,omitempty"`
}

// ConfigGrantRequest is a grant creation request within ConfigureRequest.
type ConfigGrantRequest struct {
	Username string `json:"username"`
	Database string `json:"database"`
	Access   string `json:"access,omitempty"`
}

// ConfigureResponse is the response from the configure endpoint.
type ConfigureResponse struct {
	Message          string               `json:"message"`
	JobID            int                  `json:"job_id"`
	StreamURL        string               `json:"stream_url"`
	Users            []ConfigUserResponse `json:"users,omitempty"`
	DeletedUsers     []string             `json:"deleted_users,omitempty"`
	Databases        []ConfigDBResponse   `json:"databases,omitempty"`
	DeletedDatabases []string             `json:"deleted_databases,omitempty"`
	Extensions       []ConfigExtResponse  `json:"extensions,omitempty"`
	Grants           []ConfigGrantRequest `json:"grants,omitempty"`
	SourceIPs        []string             `json:"source_ips,omitempty"`
	DeletedIPs       []string             `json:"deleted_ips,omitempty"`
}

// ConfigUserResponse is a user in the configure response, includes generated password.
type ConfigUserResponse struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ConfigDBResponse is a database in the configure response.
type ConfigDBResponse struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

// ConfigExtResponse is an extension in the configure response.
type ConfigExtResponse struct {
	Extension string `json:"extension"`
	Database  string `json:"database"`
}

// AddNodeResponse is the response from adding a node.
type AddNodeResponse struct {
	Message      string `json:"message"`
	JobID        int    `json:"job_id"`
	StreamURL    string `json:"stream_url"`
	NewNodeCount int    `json:"new_node_count"`
	NewNodeName  string `json:"new_node_name"`
}

// RemoveNodeRequest is the request body for removing a node.
type RemoveNodeRequest struct {
	NodeName           string `json:"node_name"`
	DeleteServer       bool   `json:"delete_server"`
	RemovePostgresData bool   `json:"remove_postgres_data"`
}

// RemoveNodeResponse is the response from removing a node.
type RemoveNodeResponse struct {
	Message      string `json:"message"`
	JobID        int    `json:"job_id"`
	StreamURL    string `json:"stream_url"`
	NewNodeCount int    `json:"new_node_count"`
	RemovedNode  string `json:"removed_node"`
}

// BackupConfig represents the backup configuration for a cluster.
type BackupConfig struct {
	ID            int       `json:"id"`
	ClusterID     int       `json:"cluster_id"`
	Enabled       bool      `json:"enabled"`
	Schedule      string    `json:"schedule"`
	RetentionFull int       `json:"retention_full"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UpdateBackupConfigRequest is the request body for updating backup config.
type UpdateBackupConfigRequest struct {
	Enabled       *bool  `json:"enabled,omitempty"`
	Schedule      string `json:"schedule,omitempty"`
	RetentionFull *int   `json:"retention_full,omitempty"`
}

// ServerType represents an available server type.
type ServerType struct {
	Type           string  `json:"type"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	CPUs           int     `json:"cpus"`
	MemoryGB       int     `json:"memory_gb"`
	StorageGB      int     `json:"storage_gb"`
	StorageAvailGB int     `json:"storage_avail_gb"`
	PricePerNode   float64 `json:"price_per_node"`
}

// ServerTypesResponse is the response from listing server types.
type ServerTypesResponse struct {
	ServerTypes []ServerType `json:"server_types"`
	Default     string       `json:"default"`
}

// Extension represents an available PostgreSQL extension.
type Extension struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Default     bool   `json:"default"`
}

// ExtensionsResponse is the response from listing extensions.
type ExtensionsResponse struct {
	Extensions []Extension `json:"extensions"`
	TotalCount int         `json:"total_count"`
}

// JobsResponse is the response from listing cluster jobs.
type JobsResponse struct {
	Jobs  []Job `json:"jobs"`
	Count int   `json:"count"`
}

// Job represents a cluster job.
type Job struct {
	ID              int       `json:"id"`
	JobType         string    `json:"job_type"`
	JenkinsJob      string    `json:"jenkins_job"`
	QueueItemID     int       `json:"queue_item_id"`
	BuildNumber     int       `json:"build_number"`
	Status          string    `json:"status"`
	ExpectedChanges string    `json:"expected_changes"`
	ErrorMessage    string    `json:"error_message"`
	StreamURL       string    `json:"stream_url"`
	Progress        int       `json:"progress"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
