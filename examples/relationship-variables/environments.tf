resource "ctrlplane_environment" "staging" {
  name              = "staging-relationship-vars"
  description       = "Staging environment"
  resource_selector = "resource.metadata['example'] == 'relationship-variables' && resource.metadata['environment'] == 'staging'"
}

resource "ctrlplane_environment" "prod" {
  name              = "prod-relationship-vars"
  description       = "Production environment"
  resource_selector = "resource.metadata['example'] == 'relationship-variables' && resource.metadata['environment'] == 'prod'"
}

resource "ctrlplane_environment_system_link" "staging" {
  environment_id = ctrlplane_environment.staging.id
  system_id      = ctrlplane_system.this.id
}

resource "ctrlplane_environment_system_link" "prod" {
  environment_id = ctrlplane_environment.prod.id
  system_id      = ctrlplane_system.this.id
}
