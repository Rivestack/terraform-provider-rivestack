resource "rivestack_cluster_extension" "example" {
  cluster_id = rivestack_cluster.example.id
  extension  = "vector"
  database   = rivestack_cluster_database.example.name
}
