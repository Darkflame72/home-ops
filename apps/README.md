# Applications

This directory contains the Helm values and configuration for applications running in the cluster.

## Structure

Applications are grouped by namespace:

```text
apps/
  <namespace>/
    <application>/
      values.yaml        # Helm values
      kustomization.yaml # Kustomize configuration (if used)
      external-secret.yaml # Secret definitions
```

## Adding a new application

1. Create a directory for the namespace if it doesn't exist.
2. Create a directory for the application.
3. Add `values.yaml` for the Helm chart.
4. Add `external-secret.yaml` if the application requires secrets.
5. Add the application to `bootstrap/helmfile.yaml` or create an ArgoCD Application in `argocd/apps/`.
