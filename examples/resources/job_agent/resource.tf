# GitHub Actions job agent
resource "ctrlplane_job_agent" "github" {
  name = "github-deployer"

  github {
    installation_id = 12345678
    owner           = "my-org"
    repo            = "deployments"
    workflow_id     = 87654321
    ref             = "main"
  }

  metadata = {
    managed_by = "terraform"
  }
}

# Terraform Cloud job agent
resource "ctrlplane_job_agent" "terraform_cloud" {
  name = "tfc-deployer"

  terraform_cloud {
    address      = "https://app.terraform.io"
    organization = "my-org"
    template     = "default"
    token        = var.tfc_token
  }
}

# Custom job agent
resource "ctrlplane_job_agent" "custom" {
  name = "custom-agent"

  custom {
    type = "my-custom-agent"

    config = {
      endpoint = "https://agent.example.com"
      region   = "us-east-1"
    }
  }
}
