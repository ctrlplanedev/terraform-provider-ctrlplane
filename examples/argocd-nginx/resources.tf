resource "ctrlplane_resource_provider" "this" {
  name = "argocd-example"

  resource {
    name       = var.resource_name
    identifier = var.resource_identifier
    kind       = var.resource_kind
    version    = var.resource_version
    metadata   = merge({ "environment" = var.environment_name }, var.resource_metadata)
  }
}
