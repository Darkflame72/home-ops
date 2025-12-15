{
}:
let
  pkgs = import (fetchTarball "https://nixos.org/channels/nixpkgs-unstable/nixexprs.tar.xz") { };

in
pkgs.mkShell {
  packages = [
    pkgs.cilium-cli
    pkgs.cloudflared
    pkgs.age
    pkgs.argocd
    pkgs.sops
    pkgs.helmfile
    pkgs.kustomize
    pkgs.kubectl
    pkgs.talosctl
    pkgs.kubernetes-helm
    pkgs.talhelper
    pkgs.cmctl
    pkgs.kubectl-cnpg
    pkgs.kubectl-rook-ceph
    pkgs.kubectl-view-secret
    pkgs.infisical
    pkgs.yq-go
    pkgs.pre-commit
    pkgs.yamllint
    pkgs.kopia
    pkgs.kubectl-rook-ceph
  ];

  KUBECONFIG = "${toString ./.}/kubeconfig";
  SOPS_AGE_KEY_FILE = "${toString ./.}/age.key";
  TALOSCONFIG = "${toString ./.}/talos/clusterconfig/talosconfig";

}
