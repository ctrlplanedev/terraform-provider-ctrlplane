resource "ctrlplane_policy" "this" {
  name     = "gradual-rollout"
  selector = "environment.name == 'gradual-rollout'"

  gradual_rollout {
    rollout_type        = "linear-normalized"
    time_scale_interval = 600
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
