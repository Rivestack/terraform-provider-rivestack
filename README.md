# Terraform Provider for Rivestack

The official Terraform provider for [Rivestack](https://rivestack.io) — managed HA PostgreSQL clusters.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- Go >= 1.24 (to build the provider from source)

## Installation

```hcl
terraform {
  required_providers {
    rivestack = {
      source  = "rivestack/rivestack"
      version = "~> 0.1"
    }
  }
}
```

## Authentication

Get your API key from the [Rivestack Dashboard](https://app.rivestack.io). API keys use the `rsk_` prefix.

Set it via environment variable (recommended):

```sh
export RIVESTACK_API_KEY="rsk_your_api_key_here"
```

Or in the provider configuration:

```hcl
provider "rivestack" {
  api_key = "rsk_your_api_key_here"
}
```

## Quick Start

```hcl
provider "rivestack" {}

# Create an HA PostgreSQL cluster
resource "rivestack_cluster" "main" {
  name               = "my-cluster"
  region             = "eu-central"
  server_type        = "starter"
  node_count         = 2
  postgresql_version = 17
}

# Create a database
resource "rivestack_cluster_database" "app" {
  cluster_id = rivestack_cluster.main.id
  name       = "myapp"
}

# Create a user
resource "rivestack_cluster_user" "app" {
  cluster_id = rivestack_cluster.main.id
  username   = "app_user"
}

# Install an extension
resource "rivestack_cluster_extension" "vector" {
  cluster_id = rivestack_cluster.main.id
  extension  = "vector"
}

output "host" {
  value = rivestack_cluster.main.host
}

output "connection_string" {
  value     = rivestack_cluster.main.connection_string
  sensitive = true
}
```

## Resources

| Resource | Description |
|---|---|
| `rivestack_cluster` | HA PostgreSQL cluster lifecycle |
| `rivestack_cluster_database` | Database on a cluster |
| `rivestack_cluster_user` | Database user |
| `rivestack_cluster_extension` | PostgreSQL extension |
| `rivestack_cluster_grant` | User access grant |
| `rivestack_cluster_backup_config` | Backup schedule |

## Data Sources

| Data Source | Description |
|---|---|
| `rivestack_cluster` | Look up a cluster by ID |
| `rivestack_server_types` | List available server types |
| `rivestack_extensions` | List available PostgreSQL extensions |

## Provider Configuration

| Attribute | Description | Default |
|---|---|---|
| `api_key` | API key (`rsk_` prefix). Env: `RIVESTACK_API_KEY` | — |
| `base_url` | API base URL. Env: `RIVESTACK_BASE_URL` | `https://api.rivestack.io` |

## Import

All resources support `terraform import`:

```sh
terraform import rivestack_cluster.main 42
terraform import rivestack_cluster_database.app 42/myapp
terraform import rivestack_cluster_user.app 42/app_user
terraform import rivestack_cluster_extension.vector 42/vector/myapp
terraform import rivestack_cluster_grant.reader 42/reader/myapp
terraform import rivestack_cluster_backup_config.main 42
```

## Building from Source

```sh
git clone https://github.com/Rivestack/terraform-provider-rivestack.git
cd terraform-provider-rivestack
make install
```

## License

[MPL-2.0](LICENSE)
