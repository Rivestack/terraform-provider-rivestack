data "rivestack_server_types" "available" {}

output "server_types" {
  value = data.rivestack_server_types.available.server_types
}
