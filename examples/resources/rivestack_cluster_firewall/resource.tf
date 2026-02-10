resource "rivestack_cluster_firewall" "example" {
  cluster_id = rivestack_cluster.example.id
  source_ips = [
    "203.0.113.0/24",
    "198.51.100.10/32",
  ]
}
