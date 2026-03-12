resource "ctrlplane_job_agent" "this" {
  name = "github-runner"

  github {
    installation_id = 69350482
    owner           = "ctrlplanedev"
    repo            = "ctrlplane-old"
  }
}
