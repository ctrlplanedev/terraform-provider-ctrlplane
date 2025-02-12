terraform {
  required_providers {
    ctrlplane = {
      source = "registry.terraform.io/ctrlplanedev/ctrlplane"
    }
  }
}

provider "ctrlplane" {
}

resource "ctrlplane_system" "example" {
  name        = "example-system"
  description = "Example system"
  slug        = "example-system"
}

resource "ctrlplane_environment" "example" {
  name        = "example-env"
  description = "Example environment"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    filter_type = "kubernetes"
    namespace   = "production"
  }
}

# Data source example
data "ctrlplane_environment" "example" {
  id = ctrlplane_environment.example.id
}

output "environment_name" {
  value = data.ctrlplane_environment.example.name
}

output "environment_metadata" {
  value = data.ctrlplane_environment.example.metadata
}

output "environment_resource_filter" {
  value = data.ctrlplane_environment.example.resource_filter
}
