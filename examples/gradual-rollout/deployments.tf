resource "ctrlplane_deployment" "this" {
  name               = "gradual-rollout-deployment"
  resource_selector  = "resource.kind == 'gradual-rollout-testing' && resource.version == 'ctrlplane.dev/gradual-rollout-testing/v1'"
  job_agent_selector = "jobAgent.id == \"${ctrlplane_job_agent.this.id}\""

  test_runner {
    delay_seconds = 10
    status        = "successful"
    message       = "Test runner job agent"
  }
}

resource "ctrlplane_deployment_system_link" "this" {
  deployment_id = ctrlplane_deployment.this.id
  system_id     = ctrlplane_system.this.id
}
