resource "ctrlplane_policy" "progression" {
  name     = "progression"
  selector = "environment.name == 'prod'"

  environment_progression {
    depends_on_environment_selector = "environment.name == 'qa'"
    minimum_success_percentage      = 80
  }
}
