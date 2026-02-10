data "rivestack_extensions" "available" {}

output "extensions" {
  value = data.rivestack_extensions.available.extensions
}
