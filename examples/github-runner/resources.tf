resource "ctrlplane_resource_provider" "this" {
  name = "github-runner-test"

  resource {
    name       = "github-runner-test"
    identifier = "github-runner-test"
    kind       = "kubernetes/namespace"
    version    = "ctrlplane.dev/kubernetes/namespace/v1"
  }
}
