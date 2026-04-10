resource "ctrlplane_deployment" "this" {
  name               = "simple-variable-deployment"
  resource_selector  = "resource.kind == 'testing' && resource.version == 'ctrlplane.dev/testing/v1'"
  job_agent_selector = "jobAgent.id == \"${ctrlplane_job_agent.this.id}\""

  test_runner {
    delay_seconds = 10
    status        = "successful"
    message       = "Test runner job agent"
  }
}

resource "ctrlplane_deployment_system_link" "this" {
  deployment_id = ctrlplane_deployment.this.id
  system_id     = ctrlplane_system.simple-variables.id
}

resource "ctrlplane_deployment_variable" "this" {
  deployment_id = ctrlplane_deployment.this.id
  key           = "simple-variable"
  description   = "A simple variable"
}

resource "ctrlplane_deployment_variable_value" "prod" {
  variable_id       = ctrlplane_deployment_variable.this.id
  priority          = 1
  literal_value     = "prod"
  resource_selector = "resource.metadata['environment'] == 'prod'"
}

resource "ctrlplane_deployment_variable_value" "qa" {
  variable_id       = ctrlplane_deployment_variable.this.id
  priority          = 2
  literal_value     = "qa"
  resource_selector = "resource.metadata['environment'] == 'qa'"
}
