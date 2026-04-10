resource "ctrlplane_deployment" "this" {
  name               = "relationship-variables-app"
  resource_selector  = "resource.metadata['example'] == 'relationship-variables' && resource.kind == 'KubernetesCluster'"
  job_agent_selector = "jobAgent.id == \"${ctrlplane_job_agent.this.id}\""

  test_runner {
    delay_seconds = 5
    status        = "successful"
    message       = "Test runner for relationship variables example"
  }
}

resource "ctrlplane_deployment_system_link" "this" {
  deployment_id = ctrlplane_deployment.this.id
  system_id     = ctrlplane_system.this.id
}

# --- Deployment variable: database_url ---
# This variable resolves its value from a *related* resource (the database)
# rather than a hardcoded literal.

resource "ctrlplane_deployment_variable" "database_url" {
  deployment_id = ctrlplane_deployment.this.id
  key           = "database_url"
  description   = "Connection string to the database, resolved via relationship"
  default_value = "postgres://localhost:5432/myapp"
}

# The value uses reference_value to pull "config.connection_string" from the
# related database resource (matched via the "cluster-database" relationship).
# At release time, the engine:
#   1. Looks at the target resource (a KubernetesCluster)
#   2. Follows the "cluster-database" relationship to find the related Database
#   3. Reads config.connection_string from that Database resource
#   4. Sets that as the value of the "database_url" variable

resource "ctrlplane_deployment_variable_value" "database_url" {
  variable_id       = ctrlplane_deployment_variable.database_url.id
  priority          = 1
  resource_selector = "resource.metadata['example'] == 'relationship-variables' && resource.kind == 'KubernetesCluster'"

  reference_value = {
    reference = "cluster-database"
    path      = ["config", "connection_string"]
  }
}

# --- Deployment variable: database_host ---
# Another example pulling a different property from the same relationship.

resource "ctrlplane_deployment_variable" "database_host" {
  deployment_id = ctrlplane_deployment.this.id
  key           = "database_host"
  description   = "Database hostname, resolved via relationship"
}

resource "ctrlplane_deployment_variable_value" "database_host" {
  variable_id       = ctrlplane_deployment_variable.database_host.id
  priority          = 1
  resource_selector = "resource.metadata['example'] == 'relationship-variables' && resource.kind == 'KubernetesCluster'"

  reference_value = {
    reference = "cluster-database"
    path      = ["config", "host"]
  }
}
