# Talos Patching

This directory contains the machine config patches applied by
[`topf`](https://github.com/postfinance/topf) on top of `talos/topf.yaml`. Patches are plain (or
Go-templated, `.yaml.tpl`) Talos config documents — either legacy `v1alpha1` nested fields, or
explicit `apiVersion: v1alpha1` / `kind: <DocumentKind>` documents for anything Talos 1.14 moved
out of the monolithic `v1alpha1` document (`KubeAPIServerConfig`, `SysctlConfig`, `ResolverConfig`,
etc. — see [the config reference](https://docs.siderolabs.com/talos/v1.14/reference/configuration/document-map)
for the full old-field → new-document mapping).

## Patch Directories

- **`all/`**: applied to every node
- **`control-plane/`**: applied only to control-plane nodes (k8s-1, k8s-3, k8s-5)
- **`node/<host>/`**: applied only to that specific node (currently unused)

Patches are applied in that order (`all/` → role → node), lexicographically within each
directory — hence the `00-`/`01-`/`02-` prefixes on the templated files, which have to run before
anything that might depend on them.

## `all/` patches

| File | Purpose |
|---|---|
| `00-hostname.yaml.tpl` | Sets `HostnameConfig` from `{{ .Node.Host }}`. topf's `host:` field is display-only — without this, a node renames itself to `talos-xxx-xxx` on first apply. |
| `01-network.yaml.tpl` | Per-node `LinkAliasConfig`/`LinkConfig` (MAC-selected interface, static IP) and, for control-plane nodes, `Layer2VIPConfig` — templated off each node's `data.mac`/`data.vip` in `topf.yaml`. |
| `02-install.yaml.tpl` | `UnattendedInstallConfig` (replaces `.machine.install`) — per-node install disk from `data.installDisk`. |
| `03-cluster-identity.yaml` | `machine.certSANs` (legacy field, still valid), `KubeNetworkConfig` (pod/service subnets, dnsDomain — Cilium-sized, not Talos's Flannel-sized defaults), explicit delete of the default `KubeFlannelCNIConfig` (Cilium replaces it; unlike the old `cni.name: none` field, Talos 1.14 doesn't suppress the default Flannel document on its own), and `SecurityProfileConfig` pinned to `workloadIsolation: false` (Talos 1.14's new-cluster default is `true`; this cluster explicitly keeps the pre-1.14, non-sandboxed behavior for now). |
| `machine-files.yaml` | CRI containerd customization (still a valid legacy field; `CRICustomizationConfig` is the eventual replacement). |
| `machine-kubelet.yaml` | `KubeletConfig` (extra kubelet config) + `KubeNodeConfig` (node IP subnet). |
| `machine-network.yaml` | `ResolverConfig` — DNS nameservers, search domain. |
| `machine-sysctls.yaml` | Kernel sysctl tuning (still a valid legacy field). |
| `machine-time.yaml` | NTP servers (still a valid legacy field). |

## `control-plane/` patches

| File | Purpose |
|---|---|
| `cluster.yaml` | `KubeAPIServerConfig`, `KubeControllerManagerConfig`, `KubeSchedulerConfig` (extra args), `KubeAdmissionControlConfig` delete (removes the default `PodSecurity` plugin), `KubeProxyConfig` (`enabled: false` — Cilium replaces kube-proxy), `KubeCoreDNSConfig` (`enabled: false` — CoreDNS is deployed separately via Helm), `KubeNodeConfig` taint delete (allows scheduling on control planes), and legacy `cluster.etcd` (metrics port, advertised subnet — `cluster.etcd.*` has no 1.14 replacement, stays legacy). |
| `machine-upgrade.yaml` | `machine.features.kubernetesTalosAPIAccess` (legacy field, still valid) — grants `tuppr` Talos API access from the `system-upgrade` namespace to drive OS/Kubernetes upgrades. |

## Workflow

```bash
cd talos
topf render                 # write rendered per-node configs to output/ for inspection
topf apply --dry-run         # show the diff topf would apply to the live cluster, no changes made
topf apply                   # apply for real
```

`topf schematic-ids` shows the resolved Image Factory schematic ID (from `schematic.yaml`) without
submitting anything. `output/` is gitignored — it contains fully-rendered, plaintext secrets, never
commit it.

### A note on why some fields are legacy and some are new-document

Talos 1.14 only *requires* the new document form for fields that also have Talos's own
auto-generated default for that document (kube-apiserver/controller-manager/scheduler/proxy,
CoreDNS, kubelet, node IP/taints, DNS resolver, cluster network, install) — setting both the old
and new location for these errors at apply time ("already set in v1alpha1 config"). Fields with no
such auto-generated default (`sysctls`, `time`, `files`, `certSANs`,
`features.kubernetesTalosAPIAccess`, `cluster.etcd`) remain valid as plain legacy fields
indefinitely (Talos 1.14 explicitly keeps backwards compatibility for these), so they're left
alone here rather than converted for the sake of it.
