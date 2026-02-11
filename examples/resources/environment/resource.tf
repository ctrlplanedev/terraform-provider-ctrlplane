resource "ctrlplane_environment" "production" {
  name        = "production"
  description = "Production environment"
  system_id   = ctrlplane_system.example.id

  resource_selector = "resource.metadata[\"environment\"] == \"production\""

  metadata = {
    tier   = "critical"
    region = "us-east-1"
  }
}

resource "ctrlplane_environment" "staging" {
  name        = "staging"
  description = "Staging environment for pre-production testing"
  system_id   = ctrlplane_system.example.id

  resource_selector = "resource.metadata[\"environment\"] == \"staging\""
}
