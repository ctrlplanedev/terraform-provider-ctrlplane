resource "ctrlplane_job_agent" "this" {
  name = "simple-runner-variables-example"
  test_runner {
    delay_seconds = 10
    status        = "successful"
    message       = "Test runner job agent for variables example"
  }
}
