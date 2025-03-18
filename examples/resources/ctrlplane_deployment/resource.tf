terraform {
  required_providers {
    ctrlplane = {
      source = "registry.terraform.io/ctrlplanedev/ctrlplane"
    }
  }
}

provider "ctrlplane" {}


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

  # Required job agent configuration
  job_agent_config = {
    "agent_type" = "kubernetes"
    "namespace"  = "default"
  }

  # Optional fields
  retry_count = 3
  timeout     = 600

  # Optional resource filter configuration
  resource_filter = {
    "environment" = "dev"
    "region"      = "us-west"
  }
}
