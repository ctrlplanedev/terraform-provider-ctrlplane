resource "ctrlplane_resource" "example" {
  name        = "my-k8s-cluster"
  identifier  = "k8s/my-cluster-prod"
  kind        = "kubernetes/cluster"
  version     = "1.28"
  provider_id = "your-resource-provider-id"

  config = {
    host        = "https://k8s.example.com"
    cluster_arn = "arn:aws:eks:us-east-1:123456789:cluster/my-cluster"
  }

  metadata = {
    environment = "production"
    region      = "us-east-1"
    team        = "platform"
  }
}
