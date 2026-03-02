resource "ctrlplane_resource" "qa" {
  name        = "qa"
  identifier  = "qa"
  kind        = "testing"
  version     = "ctrlplane.dev/testing/v1"
  provider_id = "72eb83d8-b825-47c1-960d-405e96137282"

  config = {
    host = "https://qa.example.com"
  }

  metadata = {
    environment = "qa"
  }
}

resource "ctrlplane_resource" "prod" {
  name        = "prod"
  identifier  = "prod"
  kind        = "testing"
  version     = "ctrlplane.dev/testing/v1"
  provider_id = "72eb83d8-b825-47c1-960d-405e96137282"

  config = {
    host = "https://prod.example.com"
  }

  metadata = {
    environment = "prod"
  }
}
