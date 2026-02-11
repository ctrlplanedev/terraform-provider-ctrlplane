# Simple workflow template
resource "ctrlplane_workflow_template" "deploy" {
  name = "deploy-service"

  input {
    name           = "environment"
    type           = "string"
    default_string = "staging"
  }

  input {
    name            = "dry_run"
    type            = "boolean"
    default_boolean = false
  }

  job {
    name = "deploy"
    ref  = "github-deployer"

    config = {
      workflow = "deploy.yml"
      ref      = "main"
    }
  }
}

# Multi-step workflow with conditional job
resource "ctrlplane_workflow_template" "deploy_and_verify" {
  name = "deploy-and-verify"

  input {
    name           = "image_tag"
    type           = "string"
    default_string = "latest"
  }

  input {
    name           = "replicas"
    type           = "number"
    default_number = 3
  }

  job {
    name = "deploy"
    ref  = "github-deployer"

    config = {
      workflow = "deploy.yml"
    }
  }

  job {
    name = "run-smoke-tests"
    ref  = "github-deployer"
    if   = "job.deploy.status == 'successful'"

    config = {
      workflow = "smoke-tests.yml"
    }
  }
}
