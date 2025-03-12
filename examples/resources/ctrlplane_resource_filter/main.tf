terraform {
  required_providers {
    ctrlplane = {
      source = "registry.terraform.io/ctrlplanedev/ctrlplane"
    }
  }
}

provider "ctrlplane" {}

resource "ctrlplane_system" "example" {
  name        = "resource-filter-example-system"
  description = "Example system for resource filter"
  slug        = "resource-filter-example"
}

# Define a reusable resource filter
resource "ctrlplane_resource_filter" "statefulset_filter" {
  type     = "comparison"
  operator = "and"
  conditions = [
    {
      type     = "kind"
      operator = "equals"
      value    = "StatefulSet"
    },
    {
      type     = "name"
      operator = "contains"
      value    = "postgres"
    }
  ]
}

data "ctrlplane_resource_filter" "statefulset_filter" {
  type     = "comparison"
  operator = "and"
  conditions = [
    {
      type     = "kind"
      operator = "equals"
      value    = "StatefulSet"
    },
    {
      type     = "name"
      operator = "contains"
      value    = "postgres"
    }
  ]
}

# # Use the resource filter in an environment
# resource "ctrlplane_environment" "production_postgres" {
#   name        = "filter-reference-example"
#   description = "Example using a referenced resource filter"
#   system_id   = ctrlplane_system.example.id

#   metadata = {
#     env  = "production"
#     team = "platform"
#     app  = "postgres"
#   }

#   # Reference the resource filter by ID
#   resource_filter_id = data.ctrlplane_resource_filter.statefulset_filter.id

#   # Explicitly depend on the resource filter to ensure it's created first
#   depends_on = [data.ctrlplane_resource_filter.statefulset_filter]
# }

# Add a simple example using an inline filter
resource "ctrlplane_environment" "inline_filter" {
  name        = "inline-filter-example"
  description = "Example with inline filter"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "comparison"
    operator = "or"
    conditions = [
      {
        type     = "name"
        operator = "equals"
        value    = "payment-cluster"
      },
      {
        type     = "name"
        operator = "equals"
        value    = "auth-cluster"
      },
      {
        type     = "name"
        operator = "contains"
        value    = "critical"
      }
    ]
  }
}

# output "system" {
#   value = {
#     id   = ctrlplane_system.example.id
#     name = ctrlplane_system.example.name
#     slug = ctrlplane_system.example.slug
#   }
# }

# output "resource_filter" {
#   value = {
#     id         = ctrlplane_resource_filter.statefulset_filter.id
#     type       = ctrlplane_resource_filter.statefulset_filter.type
#     conditions = ctrlplane_resource_filter.statefulset_filter.conditions
#   }
# }

# output "referenced_environment" {
#   value = {
#     id                 = ctrlplane_environment.production_postgres.id
#     name               = ctrlplane_environment.production_postgres.name
#     resource_filter_id = ctrlplane_environment.production_postgres.resource_filter_id
#   }
# }

# output "inline_environment" {
#   value = {
#     id              = ctrlplane_environment.inline_filter.id
#     name            = ctrlplane_environment.inline_filter.name
#     resource_filter = ctrlplane_environment.inline_filter.resource_filter
#   }
# }
