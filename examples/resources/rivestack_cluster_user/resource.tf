resource "rivestack_cluster_user" "example" {
  cluster_id = rivestack_cluster.example.id
  username   = "app_user"
}
