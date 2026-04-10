resource "ctrlplane_deployment" "this" {
  name               = "argocd-guestbook"
  resource_selector  = var.deployment_resource_selector
  job_agent_selector = "jobAgent.id == \"${ctrlplane_job_agent.this.id}\""
}

resource "ctrlplane_deployment_system_link" "this" {
  deployment_id = ctrlplane_deployment.this.id
  system_id     = ctrlplane_system.this.id
}
