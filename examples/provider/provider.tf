terraform {
  required_providers {
    rivestack = {
      source  = "rivestack/rivestack"
      version = "~> 0.1"
    }
  }
}

provider "rivestack" {
  api_key = var.rivestack_api_key
  # base_url = "https://api.rivestack.io" # optional, this is the default
}

variable "rivestack_api_key" {
  type      = string
  sensitive = true
}
