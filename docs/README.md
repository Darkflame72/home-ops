# Home Operations Documentation

Welcome to the documentation for the home Kubernetes cluster. This directory contains comprehensive guides covering setup, architecture, and application management.

## Getting Started

If you're new to this cluster or setting it up for the first time, start here:

1. **[Setup Guide](./setup.md)** - Complete installation and bootstrap instructions
2. **[Architecture](./architecture.md)** - Understand how the cluster is designed
3. **[Applications](./applications.md)** - Explore what's deployed

## Documentation Overview

### [Setup Guide](./setup.md)

**Purpose**: Step-by-step instructions for deploying the cluster from scratch

**Contents**:
- Prerequisites and required tools
- Cluster architecture overview (nodes, IPs, roles)
- Talos Linux installation and configuration
- External Secrets setup with Infisical
- CRD installation
- Core application bootstrapping with Helmfile
- ArgoCD deployment
- Post-installation verification
- Troubleshooting common issues

**When to use**: Setting up a new cluster, rebuilding after disaster, or onboarding new team members

### [Architecture Documentation](./architecture.md)

**Purpose**: In-depth technical documentation of cluster design and components

**Contents**:
- Hardware topology and node specifications
- Network architecture (CNI, DNS, gateways, tunnels)
- Storage systems (Rook-Ceph, OpenEBS, Volsync)
- Core components and their configurations
- Talos configuration patches explained
- High availability setup
- Security architecture
- Disaster recovery strategy
- Performance optimizations
- External dependencies

**When to use**: Understanding design decisions, troubleshooting complex issues, planning changes, or contributing to the cluster

### [Applications Catalog](./applications.md)

**Purpose**: Comprehensive reference of all deployed applications

**Contents**:
- Complete list of applications by namespace
- Application descriptions and purposes
- Chart sources and versions
- Configuration highlights
- Access information
- Status tracking
- Maintenance procedures
- Adding new applications

**When to use**: Finding what's deployed, understanding application purposes, checking application status, or deploying new services

## Directory Structure

```
docs/
├── README.md           # This file - documentation index
├── setup.md            # Installation and setup guide
├── architecture.md     # Technical architecture documentation
└── applications.md     # Application catalog and reference
```

## Additional Documentation

### Talos Patches

Detailed documentation about Talos Linux configuration patches is available in the main repository:

- **[Talos Patches](../talos/patches/README.md)** - Comprehensive guide to all Talos configuration patches

This document explains:
- All global patches (applied to all nodes)
- Control plane-specific patches
- Purpose and benefits of each patch
- How to modify or add new patches
- Troubleshooting patch issues

## Quick Links

### Common Tasks

- **Deploy a new application**: See [Applications - Adding New Applications](./applications.md#adding-new-applications)
- **Update Talos configuration**: See [Architecture - Talos Configuration](./architecture.md#talos-configuration)
- **Check application status**: See [Applications - Monitoring Application Health](./applications.md#monitoring-application-health)
- **Backup and recovery**: See [Architecture - Disaster Recovery](./architecture.md#disaster-recovery)
- **Node maintenance**: See [Architecture - Maintenance and Updates](./architecture.md#maintenance-and-updates)

### External Resources

- [Talos Linux Documentation](https://www.talos.dev/)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [Cilium Documentation](https://docs.cilium.io/)
- [Rook-Ceph Documentation](https://rook.io/docs/rook/latest/)
- [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/)
- [External Secrets Operator](https://external-secrets.io/)

## Contributing to Documentation

### Updating Documentation

When making changes to the cluster configuration:

1. Update relevant documentation files to reflect changes
2. Ensure accuracy by cross-referencing with actual config files
3. Update version numbers and links as needed
4. Test commands and procedures when possible

### Documentation Standards

- Use clear, descriptive headings
- Include code examples where appropriate
- Link between related documentation sections
- Keep configuration examples up-to-date
- Explain the "why" not just the "what"

### Documentation Locations

- **Cluster-wide docs**: `docs/` directory (this location)
- **Component-specific docs**: Within component directories (e.g., `talos/patches/README.md`)
- **Application docs**: Refer to upstream documentation, provide links in `applications.md`

## Getting Help

### Troubleshooting

Each documentation file includes troubleshooting sections:

- [Setup - Troubleshooting](./setup.md#troubleshooting)
- [Architecture - Disaster Recovery](./architecture.md#disaster-recovery)
- [Applications - Monitoring Application Health](./applications.md#monitoring-application-health)
- [Talos Patches - Troubleshooting](../talos/patches/README.md#troubleshooting)

### Useful Commands

Check cluster status:
```bash
kubectl get nodes
kubectl get pods -A
kubectl get applications -n argocd
```

Check Talos health:
```bash
talosctl -n <node-ip> health
```

View ArgoCD applications:
```bash
kubectl get apps -n argocd
```

## Project Information

- **Repository**: [Darkflame72/home-ops](https://github.com/Darkflame72/home-ops)
- **GitOps Tool**: ArgoCD
- **OS**: Talos Linux
- **CNI**: Cilium
- **Gateway**: Envoy Gateway
- **Storage**: Rook-Ceph, OpenEBS
- **Secrets**: Infisical via External Secrets Operator

## Documentation Updates

This documentation was last comprehensively updated: **December 9, 2025**

If you notice any inaccuracies or have suggestions for improvement, please update the relevant documentation files and submit changes through the Git repository.
