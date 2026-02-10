data "rivestack_cluster" "example" {
  id = "42"
}

output "cluster_host" {
  value = data.rivestack_cluster.example.host
}
