resource "ctrlplane_environment" "this" {
  name              = var.environment_name
  description       = "${var.environment_name} environment"
  resource_selector = var.environment_resource_selector
}

resource "ctrlplane_environment_system_link" "this" {
  environment_id = ctrlplane_environment.this.id
  system_id      = ctrlplane_system.this.id
}
