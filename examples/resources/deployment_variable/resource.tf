# Variable with a string default value
resource "ctrlplane_deployment_variable" "image_tag" {
  deployment_id = ctrlplane_deployment.example.id
  key           = "IMAGE_TAG"
  description   = "Docker image tag to deploy"
  default_value = "latest"
}

# Variable with a numeric default value
resource "ctrlplane_deployment_variable" "replica_count" {
  deployment_id = ctrlplane_deployment.example.id
  key           = "REPLICA_COUNT"
  description   = "Number of pod replicas"
  default_value = 3
}

# Variable with no default value
resource "ctrlplane_deployment_variable" "api_key" {
  deployment_id = ctrlplane_deployment.example.id
  key           = "API_KEY"
  description   = "External API key (provided per environment)"
}
