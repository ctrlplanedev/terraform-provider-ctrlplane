terraform {
  required_providers {
    ctrlplane = {
      source = "registry.terraform.io/ctrlplanedev/ctrlplane"
    }
  }
}

provider "ctrlplane" {}

# Example: Retrieve a deployment using its ID
data "ctrlplane_deployment" "example" {
  id = "9d1a9d15-6faf-426e-9657-5e7115588226"
}

# Usage example: Use retrieved deployment data
output "deployment_name" {
  value = data.ctrlplane_deployment.example.name
}

output "deployment_system_id" {
  value = data.ctrlplane_deployment.example.system_id
}

# Example: Retrieve a deployment that was created in this configuration
resource "ctrlplane_system" "example" {
  name        = "example-system"
  slug        = "example-system"
  description = "Example system for deployment"
}

resource "ctrlplane_deployment" "example" {
  name        = "example-deployment"
  slug        = "example-deployment"
  description = "Example deployment for Terraform provider"
  system_id   = ctrlplane_system.example.id

  job_agent_config = {
    "agent_type" = "kubernetes"
    "namespace"  = "default"
  }
}

# Reference the created deployment
data "ctrlplane_deployment" "created_example" {
  id = ctrlplane_deployment.example.id
}

output "created_deployment_name" {
  value = data.ctrlplane_deployment.created_example.name
}
