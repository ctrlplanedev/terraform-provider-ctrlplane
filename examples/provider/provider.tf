terraform {
  required_providers {
    ctrlplane = {
      source = "ctrlplane/ctrlplane"
    }
  }
}

provider "ctrlplane" {
  base_url = "http://localhost:3000"
}

resource "ctrlplane_system" "example" {
  name         = "tf_test_official"
  slug         = "tf_test_official"
  workspace_id = "5316df47-1f1c-4a5e-85e6-645e6b821616"
}
