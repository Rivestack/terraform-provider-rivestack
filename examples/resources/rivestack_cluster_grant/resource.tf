resource "rivestack_cluster_grant" "example" {
  cluster_id = rivestack_cluster.example.id
  username   = rivestack_cluster_user.example.username
  database   = rivestack_cluster_database.example.name
  access     = "read"
}
