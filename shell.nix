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
    pkgs.flux
    pkgs.sops
    pkgs.go-task
    pkgs.helm
    pkgs.helmfile
    pkgs.jq
    pkgs.kustomize
    pkgs.kubectl
    pkgs.yq
    pkgs.talosctl
    pkgs.kubeconform
    pkgs.talhelper
  ];

  # KUBECONFIG = "nvim";
  # SOPS_AGE_KEY_FILE = "nvim";
  # TALOSCONFIG = "";

}
