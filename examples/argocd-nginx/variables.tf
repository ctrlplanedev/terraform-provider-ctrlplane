variable "workspace" {
  type        = string
  description = "The workspace to use"
}

variable "url" {
  type        = string
  description = "The URL of the Ctrlplane API"
}

variable "api_key" {
  type        = string
  description = "The API key for the Ctrlplane API"
  sensitive   = true
}

variable "argocd_server_url" {
  type        = string
  description = "ArgoCD server address (host[:port] or URL)"
}

variable "argocd_api_key" {
  type        = string
  sensitive   = true
  description = "ArgoCD API token"
}

variable "resource_name" {
  type        = string
  description = "Display name of the Kubernetes cluster resource"
}

variable "resource_identifier" {
  type        = string
  description = "Unique identifier for the resource (must match how the cluster is registered in ArgoCD)"
}

variable "resource_kind" {
  type        = string
  default     = "KubernetesCluster"
  description = "Kind of the resource (e.g. GoogleKubernetesEngine, KubernetesCluster)"
}

variable "resource_version" {
  type        = string
  default     = "ctrlplane.dev/kubernetes/cluster/v1"
  description = "Version string of the resource schema"
}

variable "resource_metadata" {
  type        = map(string)
  default     = {}
  description = "Metadata key-value pairs for the resource"
}

variable "environment_name" {
  type        = string
  description = "Name of the environment"
}

variable "environment_resource_selector" {
  type        = string
  description = "CEL expression to select resources for this environment"
}

variable "deployment_resource_selector" {
  type        = string
  default     = "resource.kind == 'KubernetesCluster'"
  description = "CEL expression to select resources for this deployment"
}
