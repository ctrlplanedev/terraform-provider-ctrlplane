# Basic deployment
resource "ctrlplane_deployment" "api" {
  name      = "api-service"
  system_id = ctrlplane_system.example.id

  resource_selector = "resource.kind == \"kubernetes/namespace\""

  metadata = {
    service = "api"
    team    = "backend"
  }
}

# Deployment with a GitHub Actions job agent
resource "ctrlplane_deployment" "web" {
  name      = "web-frontend"
  system_id = ctrlplane_system.example.id

  resource_selector = "resource.kind == \"kubernetes/namespace\""

  job_agent {
    id = ctrlplane_job_agent.github.id

    github {
      owner       = "my-org"
      repo        = "web-frontend"
      workflow_id = 12345678
    }
  }
}
