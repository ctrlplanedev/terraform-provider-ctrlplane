resource "ctrlplane_environment" "qa" {
  name              = "qa"
  description       = "QA environment"
  resource_selector = "resource.metadata['environment'] == 'qa'"
  metadata = {
    environment = "qa"
  }
}

resource "ctrlplane_environment" "prod" {
  name              = "prod"
  description       = "Production environment"
  resource_selector = "resource.metadata['environment'] == 'prod'"
  metadata = {
    environment = "prod"
  }
}

resource "ctrlplane_environment_system_link" "qa" {
  environment_id = ctrlplane_environment.qa.id
  system_id      = ctrlplane_system.simple-variables.id
}

resource "ctrlplane_environment_system_link" "prod" {
  environment_id = ctrlplane_environment.prod.id
  system_id      = ctrlplane_system.simple-variables.id
}
