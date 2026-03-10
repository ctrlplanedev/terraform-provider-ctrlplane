resource "ctrlplane_policy" "progression" {
  name     = "progression"
  selector = "environment.name == 'prod'"

  environment_progression {
    depends_on_environment_selector = "environment.name == 'qa'"
    minimum_success_percentage      = 80
  }

  verification {
    trigger_on = "jobSuccess"

    metric {
      name     = "sleep-check"
      interval = "10s"
      count    = 1

      success {
        condition = "true"
        threshold = 1
      }

      sleep {
        duration_seconds = 30
      }
    }
  }
}
