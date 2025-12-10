# Talos Patching

This directory contains Kustomization patches that are applied to the Talos configuration via talhelper. These patches customize the base Talos configuration for this specific cluster's needs.

**Reference**: <https://www.talos.dev/v1.7/talos-guides/configuration/patching/>

## Patch Directories

Under this `patches` directory, there are several sub-directories containing patches applied at different scopes:

- **`global/`**: Patches applied to both control plane and worker nodes
- **`controller/`**: Patches applied only to control plane nodes
- **`worker/`**: Patches applied only to worker nodes (currently unused)

## Global Patches

Applied to all nodes in the cluster (both control plane and workers).

### machine-files.yaml

Configures containerd runtime settings via CRI configuration files.

**Purpose**: Optimize container image handling and security contexts

**Configuration**:

```yaml
[plugins."io.containerd.cri.v1.images"]
  discard_unpacked_layers = false
[plugins."io.containerd.cri.v1.runtime"]
  device_ownership_from_security_context = true
```

**Details**:


- `discard_unpacked_layers = false`: Retains unpacked image layers in containerd's content store, improving performance for repeated image operations
- `device_ownership_from_security_context = true`: Allows containers to own devices based on their security context (required for certain workloads)

### machine-kubelet.yaml

Configures kubelet settings for optimal performance and networking.

**Purpose**: Optimize image pulls and restrict node IP selection

**Configuration**:

- **Parallel Image Pulls**: Disables serialized image pulling, allowing multiple images to download simultaneously
- **Node IP Subnet**: Restricts node IPs to the cluster network (`10.0.60.0/24`)

**Benefits**:


- Faster pod startup times with parallel image downloads
- Prevents Talos from selecting incorrect network interfaces as the node IP

### machine-network.yaml

Configures system-level networking settings.

**Purpose**: Set reliable DNS and disable search domains

**Configuration**:

- **Nameservers**: `1.1.1.1`, `1.0.0.1` (Cloudflare DNS)
- **Search Domain**: Disabled for performance

**Benefits**:


- Consistent, fast DNS resolution
- Avoids DNS lookup delays from search domain expansion

### machine-sysctls.yaml

Tunes kernel parameters via sysctl for various workload requirements.

**Purpose**: System tuning for file watching, networking, and user namespaces

**Configuration**:

#### File System Tuning

- `fs.inotify.max_user_watches: "1048576"`: Increases maximum inotify watches (required for file-watching tools)
- `fs.inotify.max_user_instances: "8192"`: Increases inotify instances per user

**Use Case**: Essential for development tools, log watchers, and applications that monitor file changes

#### Network Tuning

- `net.core.rmem_max: "7500000"`: Increases max receive buffer size
- `net.core.wmem_max: "7500000"`: Increases max send buffer size

**Use Case**: Optimizes QUIC protocol performance for Cloudflare Tunnel

#### ARP Cache Management

- `net.ipv4.neigh.default.gc_thresh1: "4096"`: ARP cache garbage collection threshold 1
- `net.ipv4.neigh.default.gc_thresh2: "8192"`: ARP cache garbage collection threshold 2
- `net.ipv4.neigh.default.gc_thresh3: "16384"`: ARP cache garbage collection threshold 3

**Use Case**: Prevents ARP cache overflows in environments with many neighbors (important for Kubernetes clusters)

#### TCP Optimization

- `net.ipv4.tcp_slow_start_after_idle: "0"`: Disables TCP slow start after idle

**Use Case**: Preserves congestion window after idle periods, improving performance for bursty traffic

#### User Namespaces

- `user.max_user_namespaces: "11255"`: Increases maximum user namespaces

**Use Case**: Required for rootless containers and certain security features

### machine-time.yaml

Configures time synchronization (NTP).

**Purpose**: Accurate time synchronization across the cluster

**Configuration**:

- **NTP Servers**:
  - `162.159.200.1` (Cloudflare)
  - `162.159.200.123` (Cloudflare)

**Benefits**:

- Accurate cluster time (critical for certificate validation, log correlation, distributed systems)
- Uses Cloudflare's time service for consistency with other Cloudflare services

## Control Plane Patches

Applied only to control plane nodes (k8s-1, k8s-3, k8s-5).

### cluster.yaml

Configures Kubernetes control plane components.

**Purpose**: Customize API server, controller manager, scheduler, and etcd

**Configuration**:

#### Scheduling on Control Planes

```yaml
allowSchedulingOnControlPlanes: true
```

Enables workload scheduling on control plane nodes to maximize resource utilization.

#### API Server

```yaml
admissionControl:
  $$patch: delete
extraArgs:
  enable-aggregator-routing: true
    feature-gates: ImageVolume=true,MutatingAdmissionPolicy=true
    runtime-config: admissionregistration.k8s.io/v1beta1=true
```

- Removes default admission control configuration (rely on defaults or other configs)
- Enables API aggregation layer for extension API servers
- Enables feature gates for ImageVolume and MutatingAdmissionPolicy

#### Controller Manager

```yaml
extraArgs:
  bind-address: 0.0.0.0
```

Binds metrics endpoint to all interfaces (enables monitoring from other nodes).

#### CoreDNS

```yaml
coreDNS:
  disabled: true
```

Disables built-in CoreDNS as it's deployed separately via Helm.

#### Etcd

```yaml
extraArgs:
  listen-metrics-urls: http://0.0.0.0:2381
advertisedSubnets:
  - 10.0.60.0/24
```

- Exposes etcd metrics on all interfaces for monitoring
- Restricts etcd advertisements to cluster network

#### Kube-Proxy

```yaml
proxy:
  disabled: true
```

Disables kube-proxy as Cilium provides this functionality.

#### Scheduler

```yaml
extraArgs:
  bind-address: 0.0.0.0
```

Binds metrics endpoint to all interfaces.

## Worker Patches

Currently no worker-specific patches are defined. Worker nodes receive only the global patches.

## Patch Application Order

Patches are applied in the following order by talhelper:

1. Global patches (applied to all nodes)
2. Role-specific patches (controller or worker)
3. Node-specific patches (if defined)

## Modifying Patches

To modify patches:

1. Edit the patch YAML file in the appropriate directory
2. Regenerate Talos configurations:

   ```bash
   talhelper genconfig
   ```

3. Review generated configurations in `talos/clusterconfig/`
4. Apply updated configuration to nodes:

   ```bash
   talhelper gencommand upgrade | bash
   ```
