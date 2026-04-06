# Basic variable set with literal string values
resource "ctrlplane_variable_set" "production" {
  name        = "production-vars"
  description = "Variables for production release targets"
  selector    = "resource.metadata[\"environment\"] == \"production\""
  priority    = 10

  variables = [
    {
      key   = "API_URL"
      value = "https://api.example.com"
    },
    {
      key   = "LOG_LEVEL"
      value = "warn"
    },
    {
      key   = "MAX_RETRIES"
      value = "3"
    },
  ]
}

# Variable set with a reference value
resource "ctrlplane_variable_set" "cluster_info" {
  name        = "cluster-info"
  description = "Variables derived from resource metadata"
  selector    = "resource.kind == \"kubernetes/cluster\""
  priority    = 5

  variables = [
    {
      key = "CLUSTER_NAME"
      reference_value = {
        reference = "resource"
        path      = ["metadata", "cluster_name"]
      }
    },
    {
      key   = "DEFAULT_NAMESPACE"
      value = "default"
    },
  ]
}

# Variable set with no variables (selector-only)
resource "ctrlplane_variable_set" "empty" {
  name        = "placeholder"
  description = "Placeholder variable set"
  selector    = "true"
}
