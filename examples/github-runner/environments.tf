resource "ctrlplane_environment" "this" {
  name              = "github-runner-test"
  description       = "GitHub runner test environment"
  resource_selector = "resource.name == 'github-runner-test'"
}

resource "ctrlplane_environment_system_link" "this" {
  environment_id = ctrlplane_environment.this.id
  system_id      = ctrlplane_system.this.id
}

