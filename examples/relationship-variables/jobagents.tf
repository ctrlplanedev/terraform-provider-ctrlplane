resource "ctrlplane_job_agent" "this" {
  name = "relationship-variables-runner"
  test_runner {
    delay_seconds = 5
    status        = "successful"
    message       = "Test runner for relationship variables example"
  }
}
