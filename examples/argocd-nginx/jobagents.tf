resource "ctrlplane_job_agent" "this" {
  name = "argocd-nginx-runner"

  argocd {
    server_url = var.argocd_server_url
    api_key    = var.argocd_api_key
    template   = <<-EOT
      {{- $resourceName := .resource.name -}}
      {{- $environmentName := .environment.name -}}
      ---
      apiVersion: argoproj.io/v1alpha1
      kind: Application
      metadata:
        name: "{{$resourceName}}-guestbook"
        namespace: argocd
        labels:
          app.kubernetes.io/name: guestbook
          environment: "{{$environmentName}}"
          resource: "{{$resourceName}}"
      spec:
        project: default
        source:
          repoURL: https://github.com/argoproj/argocd-example-apps
          targetRevision: master
          path: guestbook
        destination:
          name: "{{.resource.identifier}}"
          namespace: guestbook
        syncPolicy:
          automated:
            prune: true
            selfHeal: true
          syncOptions:
            - CreateNamespace=true
    EOT
  }
}
