resource "ctrlplane_environment" "this" {
  name              = "gradual-rollout"
  description       = "Gradual rollout environment"
  resource_selector = "resource.metadata['environment'] == 'gradual-rollout'"
  metadata = {
    environment = "gradual-rollout"
  }
}

resource "ctrlplane_environment_system_link" "this" {
  environment_id = ctrlplane_environment.this.id
  system_id      = ctrlplane_system.this.id
}

