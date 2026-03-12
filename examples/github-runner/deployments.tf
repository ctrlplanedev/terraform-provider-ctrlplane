resource "ctrlplane_deployment" "this" {
  name              = "github-runner-deployment"
  resource_selector = "resource.name == 'github-runner-test'"
  job_agent {
    id = ctrlplane_job_agent.this.id
    github {
      workflow_id = 106983480
    }
  }
}

resource "ctrlplane_deployment_system_link" "this" {
  deployment_id = ctrlplane_deployment.this.id
  system_id     = ctrlplane_system.this.id
}
