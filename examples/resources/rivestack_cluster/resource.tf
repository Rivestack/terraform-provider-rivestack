resource "rivestack_cluster" "example" {
  name              = "my-cluster"
  region            = "eu-central"
  server_type       = "starter"
  node_count        = 2
  postgresql_version = 17
}
