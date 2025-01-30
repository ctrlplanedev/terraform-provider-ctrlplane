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
