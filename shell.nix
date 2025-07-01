{
}:
let
  pkgs = import (fetchTarball "https://nixos.org/channels/nixpkgs-unstable/nixexprs.tar.xz") { };

in
pkgs.mkShell {
  packages = [
    pkgs.python3
    pkgs.makejinja
    pkgs.cilium-cli
    pkgs.cloudflared
    pkgs.cue
    pkgs.age
    pkgs.fluxcd
    pkgs.sops
    pkgs.go-task
    pkgs.helmfile
    pkgs.jq
    pkgs.kustomize
    pkgs.kubectl
    pkgs.yq
    pkgs.talosctl
    pkgs.kubeconform
    pkgs.kubernetes-helm
    pkgs.talhelper
    pkgs.nmap
    pkgs.cmctl
    pkgs.kubectl-cnpg
    pkgs.kubectl-rook-ceph
    pkgs.kubectl-view-secret
    pkgs.kubescape
  ];

  KUBECONFIG = "${toString ./.}/kubeconfig";
  SOPS_AGE_KEY_FILE = "${toString ./.}/age.key";
  TALOSCONFIG = "${toString ./.}/talos/clusterconfig/talosconfig";

}
