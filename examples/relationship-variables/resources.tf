# Two kinds of resources: clusters and databases.
# They share the same project, which is what the relationship rule matches on.
# All resources have "example" = "relationship-variables" to scope selectors.

resource "ctrlplane_resource_provider" "this" {
  name = "relationship-variables-testing"

  # A Kubernetes cluster in project "acme-prod"
  resource {
    name       = "prod-cluster"
    identifier = "relationship-variables/prod-cluster"
    kind       = "KubernetesCluster"
    version    = "ctrlplane.dev/kubernetes/cluster/v1"
    config = jsonencode({
      server = "https://10.0.0.1"
    })
    metadata = {
      "example"     = "relationship-variables"
      "project"     = "acme-prod"
      "environment" = "prod"
    }
  }

  # A database in the same project "acme-prod"
  resource {
    name       = "prod-database"
    identifier = "relationship-variables/prod-database"
    kind       = "Database"
    version    = "ctrlplane.dev/database/v1"
    config = jsonencode({
      connection_string = "postgres://db.acme-prod.internal:5432/myapp"
      host              = "db.acme-prod.internal"
      port              = 5432
    })
    metadata = {
      "example" = "relationship-variables"
      "project" = "acme-prod"
    }
  }

  # A Kubernetes cluster in project "acme-staging"
  resource {
    name       = "staging-cluster"
    identifier = "relationship-variables/staging-cluster"
    kind       = "KubernetesCluster"
    version    = "ctrlplane.dev/kubernetes/cluster/v1"
    config = jsonencode({
      server = "https://10.0.1.1"
    })
    metadata = {
      "example"     = "relationship-variables"
      "project"     = "acme-staging"
      "environment" = "staging"
    }
  }

  # A database in the same project "acme-staging"
  resource {
    name       = "staging-database"
    identifier = "relationship-variables/staging-database"
    kind       = "Database"
    version    = "ctrlplane.dev/database/v1"
    config = jsonencode({
      connection_string = "postgres://db.acme-staging.internal:5432/myapp"
      host              = "db.acme-staging.internal"
      port              = 5432
    })
    metadata = {
      "example" = "relationship-variables"
      "project" = "acme-staging"
    }
  }
}
