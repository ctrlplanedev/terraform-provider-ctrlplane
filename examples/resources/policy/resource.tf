# Basic policy with version cooldown
resource "ctrlplane_policy" "cooldown" {
  name     = "production-cooldown"
  priority = 10
  enabled  = true

  selector = "environment.name == 'production'"

  version_cooldown {
    duration = "1h"
  }
}

# Policy with a deployment window (maintenance window)
resource "ctrlplane_policy" "maintenance_window" {
  name        = "weekday-deploys-only"
  description = "Only allow deployments during business hours on weekdays"
  priority    = 5

  selector = "environment.metadata[\"tier\"] == \"critical\""

  deployment_window {
    rrule        = "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"
    duration     = "10h"
    timezone     = "America/New_York"
    allow_window = true
  }

  metadata = {
    managed_by = "terraform"
  }
}

# Policy with verification
resource "ctrlplane_policy" "canary_verification" {
  name     = "canary-check"
  priority = 20

  selector = "deployment.name == 'api-service'"

  verification {
    trigger_on = "new_version"

    metric {
      name     = "error-rate"
      interval = "1m"
      count    = 5

      datadog {
        site    = "datadoghq.com"
        api_key = var.datadog_api_key
        app_key = var.datadog_app_key

        queries = {
          errors = "sum:http.errors{service:api}.as_count()"
          total  = "sum:http.requests{service:api}.as_count()"
        }

        formula    = "errors / total * 100"
        aggregator = "avg"
      }

      success {
        condition = "result < 1.0"
      }

      failure {
        condition = "result > 5.0"
      }
    }
  }
}
