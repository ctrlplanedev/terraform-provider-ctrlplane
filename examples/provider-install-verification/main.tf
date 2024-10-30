terraform {
  required_providers {
    ctrlplane = {
      source = "registry.terraform.io/ctrlplanedev/ctrlplane"
    }
  }
}

provider "ctrlplane" {
  base_url = "http://localhost:3000"
  token    = "23a2c39540820915.a0720874eb01c409f5c3c29f0b7a63cc1cf7f8f5abdbba0cf72e2536b2963c9a"
}

data "ctrlplane_target" "example" {
  id = "01959d69-6bf1-400a-8e99-ace7a5144b0a"
}
