terraform {
  required_providers {
    ctrlplane = {
      source  = "ctrlplanedev/ctrlplane"
      version = ">= 1.10.1"
    }
  }
}

provider "ctrlplane" {
  workspace = var.workspace
  url       = var.url
  api_key   = var.api_key
}
