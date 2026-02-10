resource "rivestack_cluster_database" "example" {
  cluster_id = rivestack_cluster.example.id
  name       = "analytics"
}
