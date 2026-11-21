# Home Ops

This site documents my homelab: a Talos Linux Kubernetes cluster managed entirely through Git with
ArgoCD. Everything in the cluster is described in this repository, so the repository is also the
best source of truth for how things are wired together.

## Where to start

- **[Setup Guide](setup.md)** walks through bootstrapping the cluster from bare Talos nodes.
- **[Architecture](architecture.md)** explains how GitOps, networking, secrets, storage, backups and
  observability fit together.
- **[Cluster Overview](cluster.md)** lists versions, control plane, upgrades and add-ons.
- **[Hardware](hardware.md)**, **[Networking](networking.md)** and **[Storage](storage.md)** cover the
  physical and network layers.
- **[Databases](databases.md)**, **[Backups](backups.md)**, **[Secrets Management](secrets.md)**,
  **[Security](security.md)** and **[Observability](observability.md)** cover the services that run on
  top.
- **[CI Checks](ci.md)** lists what runs on pull requests and pushes, and what each check enforces.

## Repository layout

| Directory   | Purpose                                                                 |
|-------------|-------------------------------------------------------------------------|
| `apps/`     | Helm values and supporting manifests, grouped by Kubernetes namespace   |
| `argocd/`   | ArgoCD `Application` manifests (the `apps` root application reads these) |
| `bootstrap/`| Helmfile and CRDs needed before ArgoCD can take over                    |
| `talos/`    | Talos machine configuration, patches and `topf` definition              |
| `docs/`     | This documentation site, built with [Zensical](https://zensical.org)    |

## Building these docs locally

```bash
pip install -r requirements-docs.txt
zensical serve     # live preview at http://localhost:8000
zensical build --strict
```

The generated site is written to `site/`, which is git-ignored.
