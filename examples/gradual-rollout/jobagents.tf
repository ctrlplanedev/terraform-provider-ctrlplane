resource "ctrlplane_job_agent" "this" {
  name = "gradual-rollout-runner"
  test_runner {
    delay_seconds = 10
    status        = "successful"
    message       = "Test runner job agent"
  }
}
