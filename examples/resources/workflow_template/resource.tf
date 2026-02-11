# Simple workflow template using GitHub job agent
resource "ctrlplane_workflow_template" "deploy" {
  name = "deploy-service"

  input {
    key = "environment"
    string {
      default = "staging"
    }
  }

  input {
    key = "dry_run"
    boolean {
      default = false
    }
  }

  job {
    key = "deploy"

    agent {
      ref = ctrlplane_job_agent.github.id

      github {
        owner       = "my-org"
        repo        = "my-repo"
        workflow_id = 12345
        ref         = "main"
      }
    }
  }
}

# Multi-step workflow with conditional job
resource "ctrlplane_workflow_template" "deploy_and_verify" {
  name = "deploy-and-verify"

  input {
    key = "image_tag"
    string {
      default = "latest"
    }
  }

  input {
    key = "replicas"
    number {
      default = 3
    }
  }

  job {
    key = "deploy"

    agent {
      ref = ctrlplane_job_agent.github.id

      github {
        owner       = "my-org"
        repo        = "my-repo"
        workflow_id = 12345
      }
    }
  }

  job {
    key = "run-smoke-tests"
    if  = "job.deploy.status == 'successful'"

    agent {
      ref = ctrlplane_job_agent.github.id

      github {
        owner       = "my-org"
        repo        = "my-repo"
        workflow_id = 67890
      }
    }
  }
}

# Workflow using generic config for custom job agents
resource "ctrlplane_workflow_template" "custom" {
  name = "custom-workflow"

  job {
    key = "run"

    agent {
      ref = ctrlplane_job_agent.custom.id

      config = {
        image = "deploy:latest"
      }
    }
  }
}
