<div align="center">

<img width="144px" height="144px" src="https://raw.githubusercontent.com/mchestr/home-cluster/main/docs/src/assets/logo.png"/>

## My Home Kubernetes Cluster

... managed with ArgoCD and Renovate

</div>

### <img src="https://fonts.gstatic.com/s/e/notoemoji/latest/1f4a1/512.gif" alt="💡" width="16" height="16"> Core Components

Core components that form the foundation of the cluster:

- [argoproj/argo-cd](https://argo-cd.readthedocs.io/en/stable/): GitOps continuous delivery tool for Kubernetes.
- [backube/volsync](https://github.com/backube/volsync): Backup and recovery of persistent volume claims.
- [cilium/cilium](https://github.com/cilium/cilium): Kubernetes CNI providing networking, security, and observability.
- [envoyproxy/gateway](https://github.com/envoyproxy/gateway): Kubernetes-based application gateway using [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/).
- [external-secrets/external-secrets](https://github.com/external-secrets/external-secrets): Manages Kubernetes secrets using [Infisical](https://infisical.com).
- [jetstack/cert-manager](https://cert-manager.io/docs/): Creates SSL certificates for services in my Kubernetes cluster.
- [kubernetes-sigs/external-dns](https://github.com/kubernetes-sigs/external-dns): Automatically manages DNS records from my cluster in Cloudflare and Pi-hole.
- [openebs/openebs](https://openebs.io/): Container attached storage for Kubernetes.
- [rook/rook](https://github.com/rook/rook): Distributed block storage for persistent storage using Ceph.
- [siderolabs/talos](https://www.talos.dev/): Secure, immutable, and minimal Linux distribution for Kubernetes.

### <img src="https://fonts.gstatic.com/s/e/notoemoji/latest/1f916/512.gif" alt="🤖" width="16" height="16"> Automation

- [Github Actions](https://docs.github.com/en/actions) for checking code formatting and running periodic jobs
- [Renovate](https://github.com/renovatebot/renovate) keeps the application charts and container images up-to-date

### <img src="https://fonts.gstatic.com/s/e/notoemoji/latest/1f32a_fe0f/512.gif" alt="🌪" width="16" height="16"> Cloud Dependencies

- [Cloudflare](https://cloudflare.com): Tunnels for exposing services, DNS management, and domain management.
- [Infisical](https://infisical.com): Secrets management platform providing secure secret storage and synchronization to Kubernetes.

### <img src="https://fonts.gstatic.com/s/e/notoemoji/latest/1f35d/512.gif" alt="🍝" width="16" height="16"> Directories

This Git repository contains the following directories.

```sh
📁 apps            # Application Helm values and configurations grouped by namespace
📁 argocd          # ArgoCD Application manifests and configuration
📁 bootstrap       # Helmfile for bootstrapping core cluster components and CRDs
📁 docs            # Documentation
📁 talos           # Talos Linux configuration files and patches
```

## 📚 Documentation

Detailed documentation is available in the `docs/` directory:

- **[Setup Guide](./docs/setup.md)**: Step-by-step installation and bootstrapping instructions
- **[Architecture](./docs/architecture.md)**: Comprehensive cluster architecture, networking, and design decisions

## <img src="https://fonts.gstatic.com/s/e/notoemoji/latest/1f64f/512.gif" alt="🙏" width="16" height="16"> Gratitude and Thanks

Thanks to all the people who donate their time to the [Kubernetes @Home](https://github.com/k8s-at-home/) community.

See [LICENSE](./LICENSE)
