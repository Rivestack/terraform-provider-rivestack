resource "rivestack_cluster_backup_config" "example" {
  cluster_id     = rivestack_cluster.example.id
  enabled        = true
  schedule       = "0 3 * * *"
  retention_full = 14
}
