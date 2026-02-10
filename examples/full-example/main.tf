terraform {
  required_providers {
    rivestack = {
      source  = "rivestack/rivestack"
      version = "~> 0.1"
    }
  }
}

# Configure the provider using an API key.
# Set RIVESTACK_API_KEY environment variable or use the api_key argument.
provider "rivestack" {}

# Look up available server types and extensions.
data "rivestack_server_types" "available" {}
data "rivestack_extensions" "available" {}

# Create an HA PostgreSQL cluster.
resource "rivestack_cluster" "production" {
  name               = "production-cluster"
  region             = "eu-central"
  server_type        = "growth"
  node_count         = 2
  postgresql_version = 17
}

# Create an additional database on the cluster.
resource "rivestack_cluster_database" "analytics" {
  cluster_id = rivestack_cluster.production.id
  name       = "analytics"
  owner      = rivestack_cluster.production.db_user
}

# Create users.
resource "rivestack_cluster_user" "app_user" {
  cluster_id = rivestack_cluster.production.id
  username   = "app_user"
}

resource "rivestack_cluster_user" "readonly_user" {
  cluster_id = rivestack_cluster.production.id
  username   = "readonly_user"
}

# Grant access to users on the analytics database.
resource "rivestack_cluster_grant" "app_write" {
  cluster_id = rivestack_cluster.production.id
  username   = rivestack_cluster_user.app_user.username
  database   = rivestack_cluster_database.analytics.name
  access     = "write"
}

resource "rivestack_cluster_grant" "reader_read" {
  cluster_id = rivestack_cluster.production.id
  username   = rivestack_cluster_user.readonly_user.username
  database   = rivestack_cluster_database.analytics.name
  access     = "read"
}

# Install pgvector extension on the analytics database.
resource "rivestack_cluster_extension" "vector" {
  cluster_id = rivestack_cluster.production.id
  extension  = "vector"
  database   = rivestack_cluster_database.analytics.name
}

# Install PostGIS extension on the default database.
resource "rivestack_cluster_extension" "postgis" {
  cluster_id = rivestack_cluster.production.id
  extension  = "postgis"
}

# Restrict firewall to specific IPs.
resource "rivestack_cluster_firewall" "production" {
  cluster_id = rivestack_cluster.production.id
  source_ips = [
    "203.0.113.0/24",   # Office network
    "198.51.100.10/32", # CI/CD server
  ]
}

# Configure automated backups.
resource "rivestack_cluster_backup_config" "production" {
  cluster_id     = rivestack_cluster.production.id
  enabled        = true
  schedule       = "0 3 * * *" # Daily at 3 AM
  retention_full = 14           # Keep 14 backups
}

# Outputs
output "cluster_host" {
  value = rivestack_cluster.production.host
}

output "cluster_connection_string" {
  value     = rivestack_cluster.production.connection_string
  sensitive = true
}

output "app_user_password" {
  value     = rivestack_cluster_user.app_user.password
  sensitive = true
}
