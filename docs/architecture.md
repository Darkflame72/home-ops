# Cluster Architecture

This document describes the architecture of the home Kubernetes cluster, including hardware topology, networking, storage, and core components.

## Overview

This is a bare-metal Kubernetes cluster running on Intel NUC-style hardware, managed with Talos Linux as the operating system and ArgoCD for GitOps-based application deployment.

## Hardware Topology

### Node Configuration

| Node | Role | IP Address | Hardware | Install Disk | Extensions |
|------|------|------------|----------|--------------|------------|
| k8s-1 | Control Plane | 10.0.60.21 | Intel NUC | /dev/nvme1n1 | i915, intel-ucode, mei |
| k8s-2 | Worker | 10.0.60.22 | Intel NUC | /dev/nvme1n1 | i915, intel-ucode, mei |
| k8s-3 | Control Plane | 10.0.60.23 | Intel NUC | /dev/nvme1n1 | i915, intel-ucode, mei |
| k8s-5 | Control Plane | 10.0.60.25 | Intel NUC | /dev/nvme1n1 | i915, intel-ucode, mei |
| k8s-6 | Worker | 10.0.60.26 | Intel NUC | /dev/nvme0n1 | i915, intel-ucode, mei |
| k8s-7 | Worker | 10.0.60.27 | Intel NUC | /dev/nvme1n1 | i915, intel-ucode, mei |
| k8s-8 | Worker | 10.0.60.28 | Intel NUC | /dev/nvme0n1 | i915, intel-ucode, mei |
| k8s-9 | Worker | 10.0.60.29 | Intel NUC | /dev/nvme0n1 | i915, intel-ucode, mei |

**Total**: 8 nodes (3 control plane, 5 workers)

### Offline Nodes

- **k8s-4** (10.0.60.24): PSU not working
- **k8s-10** (10.0.60.30): Not starting

## Networking

### Network Configuration

- **Cluster Network**: 10.0.60.0/24
- **Gateway**: 10.0.60.1
- **Virtual IP (VIP)**: 10.0.60.10 (shared across control plane nodes)
- **MTU**: 1500
- **Pod CIDR**: 10.42.0.0/16
- **Service CIDR**: 10.43.0.0/16

### CNI

**Cilium** is used as the Container Network Interface, replacing the built-in Talos CNI. Cilium provides:
- Pod networking
- Network policies
- Load balancing
- Service mesh capabilities
- Network observability

### DNS

Multiple DNS providers are configured:
- **CoreDNS**: Internal cluster DNS
- **External-DNS (Cloudflare)**: Manages DNS records in Cloudflare
- **External-DNS (Pi-hole)**: Manages DNS records in local Pi-hole instance

### Ingress/Gateway

**Envoy Gateway** implements the Kubernetes Gateway API standard, providing:
- HTTP/HTTPS routing
- TLS termination
- gRPC routing
- Gateway-based traffic management

**Cloudflare Tunnel** exposes select services to the internet securely without opening firewall ports.

### Network Policies

Static network configuration is used across all nodes (no DHCP):
- Nameservers: 1.1.1.1, 1.0.0.1 (Cloudflare DNS)
- Search domain disabled for performance

## Storage

### Storage Providers

#### Rook-Ceph
Distributed block storage using Ceph, providing:
- Replicated persistent volumes
- High availability
- Dynamic volume provisioning
- Storage pools for different workload types

#### OpenEBS
Container-attached storage providing:
- Local PV provisioning
- Dynamic local volumes
- Storage classes for various use cases

### Backup and Recovery

**Volsync** handles backup and recovery of Persistent Volume Claims:
- Automated backups using Kopia
- Replication to remote storage
- Point-in-time recovery
- Integration with ArgoCD for declarative backup policies

## Core Components

### Operating System

**Talos Linux** provides:
- Immutable infrastructure
- API-driven configuration
- Secure by default (no SSH, no shell access)
- Minimal attack surface
- Automated updates

### GitOps

**ArgoCD** manages all cluster applications:
- Declarative application definitions
- Automated synchronization from Git
- Multi-source support (Helm, Kustomize, plain YAML)
- Progressive rollouts
- Self-healing capabilities

### Secrets Management

**External Secrets Operator** with **Infisical**:
- Centralized secret management
- Automatic secret synchronization
- Secret rotation support
- Universal Auth for authentication
- Project: `home-ops-iu-g0`, Environment: `prod`

### Certificate Management

**cert-manager** automates TLS certificate management:
- Automatic certificate issuance
- Certificate renewal
- Integration with Let's Encrypt
- Support for multiple issuers

### Monitoring and Observability

**Metrics Server** provides resource metrics for:
- Horizontal Pod Autoscaling
- kubectl top commands
- Resource-based scheduling decisions

**Reloader** automatically restarts pods when ConfigMaps or Secrets change, ensuring applications pick up configuration updates.

**Spegel** provides distributed container image caching:
- Peer-to-peer image distribution
- Reduced registry load
- Faster image pulls across the cluster

## Talos Configuration

### Global Patches

Applied to all nodes (control plane and workers):

1. **machine-files.yaml**: CRI configuration
   - Disables discarding unpacked layers for faster operations
   - Enables device ownership from security context

2. **machine-kubelet.yaml**: Kubelet settings
   - Disables serialized image pulls for parallel downloads
   - Restricts node IPs to cluster subnet (10.0.60.0/24)

3. **machine-network.yaml**: Network configuration
   - Disables search domain
   - Sets Cloudflare DNS nameservers

4. **machine-sysctls.yaml**: System tuning parameters

5. **machine-time.yaml**: Time synchronization settings

### Control Plane Patches

Applied only to control plane nodes:

1. **cluster.yaml**: Control plane configuration
   - Allows workload scheduling on control planes
   - Enables API aggregation layer
   - Disables built-in CoreDNS (using Cilium)
   - Disables kube-proxy (using Cilium)
   - Configures etcd metrics
   - Binds controller manager and scheduler to all interfaces

## High Availability

### Control Plane HA

Three control plane nodes provide:
- Etcd quorum (can tolerate 1 node failure)
- API server redundancy
- Scheduler and controller manager leader election

### Virtual IP

The Virtual IP (10.0.60.10) is managed by Talos across the three control plane nodes:
- Automatic failover
- Single endpoint for API access
- Transparent to clients during node failures

### Storage HA

Rook-Ceph provides:
- Replicated storage across multiple nodes
- Automatic data rebalancing
- Self-healing capabilities

## Security

### Secure Boot

Secure boot is disabled on all nodes to support custom system extensions.

### System Extensions

All nodes use Intel-specific Talos extensions:
- **i915**: Intel integrated GPU support
- **intel-ucode**: Intel microcode updates
- **mei**: Intel Management Engine Interface

### Network Security

- No DHCP (static IPs only)
- Cilium network policies
- TLS everywhere (cert-manager)
- API server mTLS

### Secret Management

- Secrets never stored in Git
- Infisical for centralized secret storage
- External Secrets Operator for K8s synchronization
- Universal Auth with client credentials

## Namespaces

Applications are organized into the following namespaces:

- `argocd`: ArgoCD and GitOps tooling
- `cert-manager`: Certificate management
- `comms`: Communication tools (Element)
- `default`: General applications (Atuin, Echo Server, Excalidraw, LittleLink, SMTP Relay)
- `external-secrets`: External Secrets Operator
- `kube-system`: Core Kubernetes components (Cilium, CoreDNS, CSI drivers, Metrics Server, Reloader, Snapshot Controller, Spegel)
- `network`: Network services (Cloudflare DNS/Tunnel, Envoy Gateway, Pi-hole DNS)
- `openebs-system`: OpenEBS storage
- `rook-ceph`: Rook-Ceph storage cluster and operator
- `volsync-system`: Backup and recovery (Kopia, Volsync)

## Maintenance and Updates

### Talos Updates

1. Modify configuration in `talos/talconfig.yaml`
2. Regenerate configs: `talhelper genconfig`
3. Apply updates: `talhelper gencommand upgrade | bash`

### Application Updates

**Renovate** automatically creates pull requests for:
- Helm chart updates
- Container image updates
- Dependency updates

Once merged, ArgoCD automatically syncs the changes to the cluster.

### Node Maintenance

To drain and reboot a node:

```bash
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data
talosctl -n <node-ip> reboot
kubectl uncordon <node-name>
```

## Disaster Recovery

### Backup Strategy

- **Volsync**: Automated PVC backups using Kopia
- **Git**: All configuration stored in version control
- **Talos**: Configuration files in `talos/clusterconfig/`
- **Secrets**: Stored in Infisical (external to cluster)

### Recovery Process

1. Rebuild nodes using Talos configuration
2. Bootstrap cluster with helmfile
3. Deploy ArgoCD
4. ArgoCD syncs all applications
5. Volsync restores PVC data from backups
6. External Secrets syncs secrets from Infisical

## Performance Optimizations

### Image Caching

- **Spegel**: P2P image distribution reduces registry load
- **Parallel Image Pulls**: Kubelet configured for non-serialized pulls

### Storage

- **OpenEBS**: Local storage for performance-critical workloads
- **Rook-Ceph**: Replicated storage for durability

### Scheduling

- Control plane nodes accept workloads to maximize resource utilization
- Appropriate node selectors and affinity rules per application

## External Dependencies

### Required External Services

1. **Infisical**: Secret management (eu.infisical.com)
2. **Cloudflare**: DNS and tunnel services
3. **GitHub**: Git repository and container registry
4. **Pi-hole**: Local DNS server

### Internet Connectivity

While many cluster operations work offline, the following require internet:
- Pulling container images
- Syncing from Git repository
- Accessing Infisical for secrets
- Cloudflare tunnel operation
- External DNS updates
- Certificate issuance from Let's Encrypt
