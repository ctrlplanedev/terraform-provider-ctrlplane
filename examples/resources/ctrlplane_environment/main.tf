terraform {
  required_providers {
    ctrlplane = {
      source = "registry.terraform.io/ctrlplanedev/ctrlplane"
    }
  }
}

provider "ctrlplane" {
  base_url  = "http://localhost:3000"
  token     = "ec2dcd404a4a53c4.41c876a1055cb2e636721fdd394be83dbdc901ab57aeccb14b0ca57eb687e26a"
  workspace = "zacharyblasczyk"
}

resource "ctrlplane_system" "example" {
  name        = "example-system"
  description = "Example system"
  slug        = "example-system"
}

# Comparison Type Examples - Real-world Use Cases

# AND operator - Find all production EKS nodes
resource "ctrlplane_environment" "comparison_and" {
  name        = "comparison-and"
  description = "Find all EKS nodes in production environment"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "comparison"
    operator = "and"
    conditions = [
      {
        type     = "name"
        operator = "contains"
        value    = "eks-node"
      },
      # {
      #   type     = "provider"
      #   operator = "equals"
      #   value    = "aws"
      # },
      {
        type     = "name"
        operator = "contains"
        value    = "production"
      }
    ]
  }

  lifecycle {
    create_before_destroy = true
  }
}

output "comparison_and" {
  value = {
    id              = ctrlplane_environment.comparison_and.id
    name            = ctrlplane_environment.comparison_and.name
    description     = ctrlplane_environment.comparison_and.description
    resource_filter = ctrlplane_environment.comparison_and.resource_filter
  }
}

data "ctrlplane_environment" "comparison_and" {
  name       = ctrlplane_environment.comparison_and.name
  system_id  = ctrlplane_environment.comparison_and.system_id
  depends_on = [ctrlplane_environment.comparison_and]
}

# OR operator - Find resources in critical clusters
resource "ctrlplane_environment" "comparison_or" {
  name        = "comparison-or"
  description = "Find resources in any critical clusters"
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

output "comparison_or" {
  value = {
    id              = ctrlplane_environment.comparison_or.id
    name            = ctrlplane_environment.comparison_or.name
    description     = ctrlplane_environment.comparison_or.description
    resource_filter = ctrlplane_environment.comparison_or.resource_filter
  }
}

data "ctrlplane_environment" "comparison_or" {
  name       = ctrlplane_environment.comparison_or.name
  system_id  = ctrlplane_environment.comparison_or.system_id
  depends_on = [ctrlplane_environment.comparison_or]
}

# Metadata Type Examples

# starts-with - AWS resources in us-east
resource "ctrlplane_environment" "metadata_starts_with" {
  name        = "metadata-starts-with"
  description = "Find all AWS resources in us-east regions"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team   = "platform"
    env    = "production"
    region = "us-east-1"
  }

  resource_filter = {
    type     = "name"
    operator = "starts-with"
    value    = "us-east"
  }
}

output "metadata_starts_with" {
  value = {
    id              = ctrlplane_environment.metadata_starts_with.id
    name            = ctrlplane_environment.metadata_starts_with.name
    description     = ctrlplane_environment.metadata_starts_with.description
    resource_filter = ctrlplane_environment.metadata_starts_with.resource_filter
  }
}

data "ctrlplane_environment" "metadata_starts_with" {
  name       = ctrlplane_environment.metadata_starts_with.name
  system_id  = ctrlplane_environment.metadata_starts_with.system_id
  depends_on = [ctrlplane_environment.metadata_starts_with]
}

# ends-with - Find x86_64 architecture resources
resource "ctrlplane_environment" "metadata_ends_with" {
  name        = "metadata-ends-with"
  description = "Find all resources with x86_64 architecture"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
    arch = "x86_64"
  }

  resource_filter = {
    type     = "name"
    operator = "ends-with"
    value    = "x86_64"
  }
}

output "metadata_ends_with" {
  value = {
    id              = ctrlplane_environment.metadata_ends_with.id
    name            = ctrlplane_environment.metadata_ends_with.name
    description     = ctrlplane_environment.metadata_ends_with.description
    resource_filter = ctrlplane_environment.metadata_ends_with.resource_filter
  }
}

data "ctrlplane_environment" "metadata_ends_with" {
  name       = ctrlplane_environment.metadata_ends_with.name
  system_id  = ctrlplane_environment.metadata_ends_with.system_id
  depends_on = [ctrlplane_environment.metadata_ends_with]
}

# contains - Find GPU-enabled resources
resource "ctrlplane_environment" "metadata_contains" {
  name        = "metadata-contains"
  description = "Find all GPU-enabled resources"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team          = "platform"
    env           = "production"
    instance_type = "p3.2xlarge"
  }

  resource_filter = {
    type     = "name"
    operator = "contains"
    value    = "gpu"
  }
}

output "metadata_contains" {
  value = {
    id              = ctrlplane_environment.metadata_contains.id
    name            = ctrlplane_environment.metadata_contains.name
    description     = ctrlplane_environment.metadata_contains.description
    resource_filter = ctrlplane_environment.metadata_contains.resource_filter
  }
}

data "ctrlplane_environment" "metadata_contains" {
  name       = ctrlplane_environment.metadata_contains.name
  system_id  = ctrlplane_environment.metadata_contains.system_id
  depends_on = [ctrlplane_environment.metadata_contains]
}

# regex - Find specific instance family
resource "ctrlplane_environment" "metadata_regex" {
  name        = "metadata-regex"
  description = "Find all M5 or M6 instance types"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "name"
    operator = "regex"
    value    = "^m[56]\\."
  }
}

output "metadata_regex" {
  value = {
    id              = ctrlplane_environment.metadata_regex.id
    name            = ctrlplane_environment.metadata_regex.name
    description     = ctrlplane_environment.metadata_regex.description
    resource_filter = ctrlplane_environment.metadata_regex.resource_filter
  }
}

data "ctrlplane_environment" "metadata_regex" {
  name       = ctrlplane_environment.metadata_regex.name
  system_id  = ctrlplane_environment.metadata_regex.system_id
  depends_on = [ctrlplane_environment.metadata_regex]
}

# equals - Find resources with specific label
resource "ctrlplane_environment" "metadata_null" {
  name        = "metadata-null"
  description = "Find resources missing owner label"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "comparison"
    operator = "and"
    conditions = [
      {
        type     = "name"
        operator = "equals"
        value    = "unlabeled-resource"
      }
    ]
  }
}

output "metadata_null" {
  value = {
    id              = ctrlplane_environment.metadata_null.id
    name            = ctrlplane_environment.metadata_null.name
    description     = ctrlplane_environment.metadata_null.description
    resource_filter = ctrlplane_environment.metadata_null.resource_filter
  }
}

data "ctrlplane_environment" "metadata_null" {
  name       = ctrlplane_environment.metadata_null.name
  system_id  = ctrlplane_environment.metadata_null.system_id
  depends_on = [ctrlplane_environment.metadata_null]
}

# Kind Type Example

# equals - Find all StatefulSet resources
resource "ctrlplane_environment" "kind_equals" {
  name        = "kind-equals"
  description = "Find all StatefulSet resources in the cluster"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "kind"
    operator = "equals"
    value    = "StatefulSet"
  }
}

output "kind_equals" {
  value = {
    id              = ctrlplane_environment.kind_equals.id
    name            = ctrlplane_environment.kind_equals.name
    description     = ctrlplane_environment.kind_equals.description
    resource_filter = ctrlplane_environment.kind_equals.resource_filter
  }
}

data "ctrlplane_environment" "kind_equals" {
  name       = ctrlplane_environment.kind_equals.name
  system_id  = ctrlplane_environment.kind_equals.system_id
  depends_on = [ctrlplane_environment.kind_equals]
}

# Provider Type Example

# equals - Find GCP resources
resource "ctrlplane_environment" "provider_equals" {
  name        = "provider-equals"
  description = "Find all GCP resources"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "name"
    operator = "contains"
    value    = "gcp"
  }
}

output "provider_equals" {
  value = {
    id              = ctrlplane_environment.provider_equals.id
    name            = ctrlplane_environment.provider_equals.name
    description     = ctrlplane_environment.provider_equals.description
    resource_filter = ctrlplane_environment.provider_equals.resource_filter
  }
}

data "ctrlplane_environment" "provider_equals" {
  name       = ctrlplane_environment.provider_equals.name
  system_id  = ctrlplane_environment.provider_equals.system_id
  depends_on = [ctrlplane_environment.provider_equals]
}

# Identifier Type Examples

# equals - Find specific pod
resource "ctrlplane_environment" "identifier_equals" {
  name        = "identifier-equals"
  description = "Find the main database pod"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "identifier"
    operator = "equals"
    value    = "postgres-main-0"
  }
}

output "identifier_equals" {
  value = {
    id              = ctrlplane_environment.identifier_equals.id
    name            = ctrlplane_environment.identifier_equals.name
    description     = ctrlplane_environment.identifier_equals.description
    resource_filter = ctrlplane_environment.identifier_equals.resource_filter
  }
}

data "ctrlplane_environment" "identifier_equals" {
  name       = ctrlplane_environment.identifier_equals.name
  system_id  = ctrlplane_environment.identifier_equals.system_id
  depends_on = [ctrlplane_environment.identifier_equals]
}

# regex - Find pods with numeric suffixes
resource "ctrlplane_environment" "identifier_regex" {
  name        = "identifier-regex"
  description = "Find all replicated pods with numeric suffixes"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "identifier"
    operator = "regex"
    value    = "-[0-9]+$"
  }
}

output "identifier_regex" {
  value = {
    id              = ctrlplane_environment.identifier_regex.id
    name            = ctrlplane_environment.identifier_regex.name
    description     = ctrlplane_environment.identifier_regex.description
    resource_filter = ctrlplane_environment.identifier_regex.resource_filter
  }
}

data "ctrlplane_environment" "identifier_regex" {
  name       = ctrlplane_environment.identifier_regex.name
  system_id  = ctrlplane_environment.identifier_regex.system_id
  depends_on = [ctrlplane_environment.identifier_regex]
}

# starts-with - Find frontend services
resource "ctrlplane_environment" "identifier_starts_with" {
  name        = "identifier-starts-with"
  description = "Find all frontend services"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "identifier"
    operator = "starts-with"
    value    = "frontend-"
  }
}

output "identifier_starts_with" {
  value = {
    id              = ctrlplane_environment.identifier_starts_with.id
    name            = ctrlplane_environment.identifier_starts_with.name
    description     = ctrlplane_environment.identifier_starts_with.description
    resource_filter = ctrlplane_environment.identifier_starts_with.resource_filter
  }
}

data "ctrlplane_environment" "identifier_starts_with" {
  name       = ctrlplane_environment.identifier_starts_with.name
  system_id  = ctrlplane_environment.identifier_starts_with.system_id
  depends_on = [ctrlplane_environment.identifier_starts_with]
}

# ends-with - Find sidecar containers
resource "ctrlplane_environment" "identifier_ends_with" {
  name        = "identifier-ends-with"
  description = "Find all sidecar containers"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "identifier"
    operator = "ends-with"
    value    = "-sidecar"
  }
}

output "identifier_ends_with" {
  value = {
    id              = ctrlplane_environment.identifier_ends_with.id
    name            = ctrlplane_environment.identifier_ends_with.name
    description     = ctrlplane_environment.identifier_ends_with.description
    resource_filter = ctrlplane_environment.identifier_ends_with.resource_filter
  }
}

data "ctrlplane_environment" "identifier_ends_with" {
  name       = ctrlplane_environment.identifier_ends_with.name
  system_id  = ctrlplane_environment.identifier_ends_with.system_id
  depends_on = [ctrlplane_environment.identifier_ends_with]
}

# contains - Find cache resources
resource "ctrlplane_environment" "identifier_contains" {
  name        = "identifier-contains"
  description = "Find all cache-related resources"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "identifier"
    operator = "contains"
    value    = "cache"
  }
}

output "identifier_contains" {
  value = {
    id              = ctrlplane_environment.identifier_contains.id
    name            = ctrlplane_environment.identifier_contains.name
    description     = ctrlplane_environment.identifier_contains.description
    resource_filter = ctrlplane_environment.identifier_contains.resource_filter
  }
}

data "ctrlplane_environment" "identifier_contains" {
  name       = ctrlplane_environment.identifier_contains.name
  system_id  = ctrlplane_environment.identifier_contains.system_id
  depends_on = [ctrlplane_environment.identifier_contains]
}

# Created-at Type Examples

# before - Find resources created before migration
resource "ctrlplane_environment" "created_at_before" {
  name        = "created-at-before"
  description = "Find resources created before the platform migration"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "created-at"
    operator = "before"
    value    = "2023-06-01T00:00:00Z"
  }
}

output "created_at_before" {
  value = {
    id              = ctrlplane_environment.created_at_before.id
    name            = ctrlplane_environment.created_at_before.name
    description     = ctrlplane_environment.created_at_before.description
    resource_filter = ctrlplane_environment.created_at_before.resource_filter
  }
}

data "ctrlplane_environment" "created_at_before" {
  name       = ctrlplane_environment.created_at_before.name
  system_id  = ctrlplane_environment.created_at_before.system_id
  depends_on = [ctrlplane_environment.created_at_before]
}

# after - Find recently created resources
resource "ctrlplane_environment" "created_at_after" {
  name        = "created-at-after"
  description = "Find resources created after recent deployment"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "created-at"
    operator = "after"
    value    = "2023-08-15T00:00:00Z"
  }
}

output "created_at_after" {
  value = {
    id              = ctrlplane_environment.created_at_after.id
    name            = ctrlplane_environment.created_at_after.name
    description     = ctrlplane_environment.created_at_after.description
    resource_filter = ctrlplane_environment.created_at_after.resource_filter
  }
}

data "ctrlplane_environment" "created_at_after" {
  name       = ctrlplane_environment.created_at_after.name
  system_id  = ctrlplane_environment.created_at_after.system_id
  depends_on = [ctrlplane_environment.created_at_after]
}

# before-or-on - Find resources on or before patching
resource "ctrlplane_environment" "created_at_before_or_on" {
  name        = "created-at-before-or-on"
  description = "Find resources created on or before the security patching"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "created-at"
    operator = "before-or-on"
    value    = "2023-07-01T00:00:00Z"
  }
}

output "created_at_before_or_on" {
  value = {
    id              = ctrlplane_environment.created_at_before_or_on.id
    name            = ctrlplane_environment.created_at_before_or_on.name
    description     = ctrlplane_environment.created_at_before_or_on.description
    resource_filter = ctrlplane_environment.created_at_before_or_on.resource_filter
  }
}

data "ctrlplane_environment" "created_at_before_or_on" {
  name       = ctrlplane_environment.created_at_before_or_on.name
  system_id  = ctrlplane_environment.created_at_before_or_on.system_id
  depends_on = [ctrlplane_environment.created_at_before_or_on]
}

# after-or-on - Find resources since a specific date
resource "ctrlplane_environment" "created_at_after_or_on" {
  name        = "created-at-after-or-on"
  description = "Find resources created since the start of Q3"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "created-at"
    operator = "after-or-on"
    value    = "2023-07-01T00:00:00Z"
  }
}

output "created_at_after_or_on" {
  value = {
    id              = ctrlplane_environment.created_at_after_or_on.id
    name            = ctrlplane_environment.created_at_after_or_on.name
    description     = ctrlplane_environment.created_at_after_or_on.description
    resource_filter = ctrlplane_environment.created_at_after_or_on.resource_filter
  }
}

data "ctrlplane_environment" "created_at_after_or_on" {
  name       = ctrlplane_environment.created_at_after_or_on.name
  system_id  = ctrlplane_environment.created_at_after_or_on.system_id
  depends_on = [ctrlplane_environment.created_at_after_or_on]
}

# Last-sync Type Examples

# before - Find resources not synced recently
resource "ctrlplane_environment" "last_sync_before" {
  name        = "last-sync-before"
  description = "Find resources not synced in the last week"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "last-sync"
    operator = "before"
    value    = "2023-08-14T00:00:00Z"
  }
}

output "last_sync_before" {
  value = {
    id              = ctrlplane_environment.last_sync_before.id
    name            = ctrlplane_environment.last_sync_before.name
    description     = ctrlplane_environment.last_sync_before.description
    resource_filter = ctrlplane_environment.last_sync_before.resource_filter
  }
}

data "ctrlplane_environment" "last_sync_before" {
  name       = ctrlplane_environment.last_sync_before.name
  system_id  = ctrlplane_environment.last_sync_before.system_id
  depends_on = [ctrlplane_environment.last_sync_before]
}

# after - Find recently synced resources
resource "ctrlplane_environment" "last_sync_after" {
  name        = "last-sync-after"
  description = "Find resources synced after latest update"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "last-sync"
    operator = "after"
    value    = "2023-08-20T00:00:00Z"
  }
}

output "last_sync_after" {
  value = {
    id              = ctrlplane_environment.last_sync_after.id
    name            = ctrlplane_environment.last_sync_after.name
    description     = ctrlplane_environment.last_sync_after.description
    resource_filter = ctrlplane_environment.last_sync_after.resource_filter
  }
}

data "ctrlplane_environment" "last_sync_after" {
  name       = ctrlplane_environment.last_sync_after.name
  system_id  = ctrlplane_environment.last_sync_after.system_id
  depends_on = [ctrlplane_environment.last_sync_after]
}

# before-or-on - Find resources not synced since a date
resource "ctrlplane_environment" "last_sync_before_or_on" {
  name        = "last-sync-before-or-on"
  description = "Find resources not synced since Q3 start"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "last-sync"
    operator = "before-or-on"
    value    = "2023-07-01T00:00:00Z"
  }
}

output "last_sync_before_or_on" {
  value = {
    id              = ctrlplane_environment.last_sync_before_or_on.id
    name            = ctrlplane_environment.last_sync_before_or_on.name
    description     = ctrlplane_environment.last_sync_before_or_on.description
    resource_filter = ctrlplane_environment.last_sync_before_or_on.resource_filter
  }
}

data "ctrlplane_environment" "last_sync_before_or_on" {
  name       = ctrlplane_environment.last_sync_before_or_on.name
  system_id  = ctrlplane_environment.last_sync_before_or_on.system_id
  depends_on = [ctrlplane_environment.last_sync_before_or_on]
}

# after-or-on - Find resources synced since a specific date
resource "ctrlplane_environment" "last_sync_after_or_on" {
  name        = "last-sync-after-or-on"
  description = "Find resources synced since monthly maintenance"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "last-sync"
    operator = "after-or-on"
    value    = "2023-08-01T00:00:00Z"
  }
}

output "last_sync_after_or_on" {
  value = {
    id              = ctrlplane_environment.last_sync_after_or_on.id
    name            = ctrlplane_environment.last_sync_after_or_on.name
    description     = ctrlplane_environment.last_sync_after_or_on.description
    resource_filter = ctrlplane_environment.last_sync_after_or_on.resource_filter
  }
}

data "ctrlplane_environment" "last_sync_after_or_on" {
  name       = ctrlplane_environment.last_sync_after_or_on.name
  system_id  = ctrlplane_environment.last_sync_after_or_on.system_id
  depends_on = [ctrlplane_environment.last_sync_after_or_on]
}

# Real-world Combined Examples

# Find all production PostgreSQL StatefulSets
resource "ctrlplane_environment" "production_postgres" {
  name        = "production-postgres"
  description = "Find all production PostgreSQL StatefulSets"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
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
}

output "production_postgres" {
  value = {
    id              = ctrlplane_environment.production_postgres.id
    name            = ctrlplane_environment.production_postgres.name
    description     = ctrlplane_environment.production_postgres.description
    resource_filter = ctrlplane_environment.production_postgres.resource_filter
  }
}

data "ctrlplane_environment" "production_postgres" {
  name       = ctrlplane_environment.production_postgres.name
  system_id  = ctrlplane_environment.production_postgres.system_id
  depends_on = [ctrlplane_environment.production_postgres]
}

# Find ARM-based nodes in non-production environments
resource "ctrlplane_environment" "arm_nonprod" {
  name        = "arm-nonprod"
  description = "Find all ARM-based nodes in non-production environments"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "comparison"
    operator = "and"
    conditions = [
      {
        type     = "kind"
        operator = "equals"
        value    = "Node"
      },
      {
        type     = "name"
        operator = "contains"
        value    = "arm64"
      },
      {
        type     = "comparison"
        operator = "or"
        conditions = [
          {
            type     = "name"
            operator = "equals"
            value    = "staging-node"
          },
          {
            type     = "name"
            operator = "equals"
            value    = "development-node"
          }
        ]
      }
    ]
  }
}

output "arm_nonprod" {
  value = {
    id              = ctrlplane_environment.arm_nonprod.id
    name            = ctrlplane_environment.arm_nonprod.name
    description     = ctrlplane_environment.arm_nonprod.description
    resource_filter = ctrlplane_environment.arm_nonprod.resource_filter
  }
}

data "ctrlplane_environment" "arm_nonprod" {
  name       = ctrlplane_environment.arm_nonprod.name
  system_id  = ctrlplane_environment.arm_nonprod.system_id
  depends_on = [ctrlplane_environment.arm_nonprod]
}

# Find recently created high-memory instances
resource "ctrlplane_environment" "new_high_memory" {
  name        = "new-high-memory"
  description = "Find all recently created high-memory instances"
  system_id   = ctrlplane_system.example.id

  metadata = {
    team = "platform"
    env  = "production"
  }

  resource_filter = {
    type     = "comparison"
    operator = "and"
    conditions = [
      {
        type     = "name"
        operator = "contains"
        value    = "xlarge"
      },
      {
        type     = "name"
        operator = "contains"
        value    = "aws"
      }
    ]
  }
}

output "new_high_memory" {
  value = {
    id              = ctrlplane_environment.new_high_memory.id
    name            = ctrlplane_environment.new_high_memory.name
    description     = ctrlplane_environment.new_high_memory.description
    resource_filter = ctrlplane_environment.new_high_memory.resource_filter
  }
}

data "ctrlplane_environment" "new_high_memory" {
  name       = ctrlplane_environment.new_high_memory.name
  system_id  = ctrlplane_environment.new_high_memory.system_id
  depends_on = [ctrlplane_environment.new_high_memory]
}
