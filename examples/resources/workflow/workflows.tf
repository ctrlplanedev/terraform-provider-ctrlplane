resource "ctrlplane_job_agent" "runner_1" {
  name = "workflow-runner-1"

  test_runner {
    delay_seconds = 10
    status        = "successful"
    message       = "Test runner job agent"
  }
}

resource "ctrlplane_job_agent" "runner_2" {
  name = "workflow-runner-2"

  test_runner {
    delay_seconds = 10
    status        = "successful"
    message       = "Test runner job agent"
  }
}

resource "ctrlplane_workflow" "example" {
  name = "example-workflow"

  inputs = jsonencode([
    { key = "environment", type = "string", default = "staging" },
    { key = "retries", type = "number", default = 3 },
    { key = "dryRun", type = "boolean", default = true },
  ])

  job_agent {
    name     = "workflow-runner-1"
    ref      = ctrlplane_job_agent.runner_1.id
    config   = { delaySeconds = "10", message = "Test runner job agent", status = "successful" }
    selector = "true"
  }

  job_agent {
    name     = "workflow-runner-2"
    ref      = ctrlplane_job_agent.runner_2.id
    config   = { delaySeconds = "10", message = "Test runner job agent", status = "successful" }
    selector = "true"
  }
}
