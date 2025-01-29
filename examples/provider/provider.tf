terraform {
  required_providers {
    ctrlplane = {
      source = "ctrlplane/ctrlplane"
    }
  }
}

provider "ctrlplane" {
  base_url  = "http://localhost:3000"
  workspace = "ctrlplane"
}

resource "ctrlplane_system" "example" {
  name = "tf_test_official"
  slug = "tf_test_official"
}
