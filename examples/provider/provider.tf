terraform {
  required_providers {
    ctrlplane = {
      source = "ctrlplanedev/ctrlplane"
    }
  }
}

provider "ctrlplane" {
  base_url  = "https://app.ctrlplane.dev"
  workspace = "my-workspace-slug"
  token     = "my-ctrlplane-token"
}

provider "ctrlplane" {
  base_url  = "https://my-ctrlplane-instance.com"
  workspace = "00000000-0000-0000-0000-000000000000"
  token     = "my-ctrlplane-token"
}
