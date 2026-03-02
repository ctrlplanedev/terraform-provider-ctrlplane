resource "ctrlplane_job_agent" "this" {
  name = "simple-runner"
  test_runner {
    delay_seconds = 10
    status        = "successful"
    message       = "Test runner job agent"
  }
}
