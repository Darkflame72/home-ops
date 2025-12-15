# Kubernetes Cluster Setup Guide

This guide walks through setting up a Kubernetes cluster using Talos Linux and ArgoCD for GitOps management.

## Prerequisites

- [talhelper](https://github.com/budimanjojo/talhelper) installed
- [kubectl](https://kubernetes.io/docs/tasks/tools/) installed
- [helmfile](https://github.com/helmfile/helmfile) installed
- Access to the hardware nodes (8 nodes total: 3 control plane, 5 workers)
- Infisical account with Universal Auth credentials configured

## Cluster Architecture

- **Control Plane Nodes**: 3 nodes (k8s-1, k8s-3, k8s-5)
- **Worker Nodes**: 5 nodes (k8s-2, k8s-6, k8s-7, k8s-8, k8s-9)
- **VIP Address**: 10.0.60.10
- **Network**: 10.0.60.0/24

## Installation Steps

### 1. Setup Talos and Nodes

Ensure Talos is installed on your local machine by following the [official installation instructions](https://www.talos.dev/latest/introduction/getting-started/).

All commands are run from the `talos/` directory.

#### 1. Generate Talos Configuration

Generate the Talos machine configurations from the `talconfig.yaml`:

```bash
talhelper genconfig
```

This creates configuration files for all nodes in the `clusterconfig/` directory.

#### 2. Apply Talos Configuration to Nodes

Apply the generated configurations to each node. The `--insecure` flag is needed for initial installation:

```bash
talhelper gencommand apply --extra-flags "--insecure" | bash
```

#### 3. Bootstrap the Cluster

Bootstrap the Kubernetes control plane on the first control plane node:

```bash
talhelper gencommand bootstrap | bash
```

Wait for the cluster to initialize. You can check the status with:

```bash
kubectl get nodes
```

### 2. Bootstrap Cluster

#### 1. Configure External Secrets (only manual secret)

Create the `external-secrets` namespace (helm will also create it, but this ensures the manual secret can be applied first):

```bash
kubectl create namespace external-secrets --dry-run=client -o yaml | kubectl apply -f -
```

Create the Infisical Universal Auth credentials secret **in the `external-secrets` namespace**. This is the only manual secret required; all other secrets are provisioned by External Secrets from Infisical.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: universal-auth-credentials
  namespace: external-secrets
type: Opaque
stringData:
  clientId: <your-client-id>
  clientSecret: <your-client-secret>
```

Apply the credentials:

```bash
kubectl apply -f infisical-credentials.yaml
```

#### 2. Install Custom Resource Definitions (CRDs)

Extract and install CRDs for components like Envoy Gateway and monitoring:

```bash
cd bootstrap
helmfile -f bootstrap/crds.yaml template -q | kubectl apply --server-side --field-manager bootstrap --force-conflicts -f -
```

#### 3. Bootstrap Core Applications

Install core cluster components (Cilium, CoreDNS, Spegel, External Secrets):

```bash
helmfile -f bootstrap/helmfile.yaml sync
```

Wait for all core components to be ready:

```bash
kubectl get pods -A
```

#### 4. Setup external secrets

```bash
kubectl apply -f apps/external-secrets/external-secrets/config/infisical.yaml
```

#### 5. Install ArgoCD

```bash
helmfile -f bootstrap/argocd.yaml sync
kubectl apply -n argocd -k argocd/
```

## Post-Installation

### Managing Applications

All applications are managed through ArgoCD. To deploy a new application:

1. Add Helm values to `apps/<namespace>/<app-name>/`
2. Create an ArgoCD Application manifest in `argocd/apps/<namespace>/<app-name>.yaml`
3. Commit and push changes
4. ArgoCD will automatically sync the new application

### Updating Talos Configuration

To update Talos configuration:

1. Modify `talos/talconfig.yaml` or patch files in `talos/patches/`
2. Regenerate configurations: `talhelper genconfig`
3. Apply updates: `talhelper gencommand upgrade | bash`
