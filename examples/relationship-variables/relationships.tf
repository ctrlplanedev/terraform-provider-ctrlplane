# Links KubernetesCluster resources to Database resources when they share the
# same "project" metadata value. The reference "cluster-database" is what
# deployment variable values use to look up the related entity.

resource "ctrlplane_relationship_rule" "cluster_to_database" {
  name      = "Cluster to Database"
  reference = "cluster-database"

  description = "Links clusters to databases in the same project"

  matcher = <<CEL
    from.kind == "KubernetesCluster" && to.kind == "Database" &&
    from.metadata["project"] == to.metadata["project"]
  CEL
}
