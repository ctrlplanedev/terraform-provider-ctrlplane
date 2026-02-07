resource "ctrlplane_relationship_rule" "cluster_to_namespace" {
  name              = "Cluster to Namespace"
  reference         = "cluster-to-namespace"
  description       = "Links Kubernetes clusters to their namespaces"
  relationship_type = "associated_with"

  matcher = "from.config.cluster_name == to.config.cluster_name"

  from {
    type     = "resource"
    selector = "resource.kind == 'kubernetes/cluster'"
  }

  to {
    type     = "resource"
    selector = "resource.kind == 'kubernetes/namespace'"
  }

  metadata = {
    managed_by = "terraform"
  }
}

resource "ctrlplane_relationship_rule" "env_to_deployment" {
  name              = "Environment to Deployment"
  reference         = "env-to-deployment"
  relationship_type = "depends_on"

  matcher = "from.system_id == to.system_id"

  from {
    type = "environment"
  }

  to {
    type = "deployment"
  }
}
