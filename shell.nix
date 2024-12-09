{ pkgs ? import <nixpkgs> {}, lib ? pkgs.lib }:
pkgs.mkShell {
  packages = [
    pkgs.direnv
    pkgs.age
    pkgs.flux
    pkgs.helmfile
    pkgs.go-task
    pkgs.sops
    pkgs.jq
    pkgs.fluxcd
    pkgs.python3
    pkgs.kubectl
    pkgs.kubernetes-helm
    pkgs.kubeconform
    pkgs.kustomize
    pkgs.talosctl
    pkgs.cloudflared
    pkgs.envsubst
    pkgs.kubectl-cnpg
    pkgs.yq
    pkgs.minijinja
    (builtins.getFlake "github:budimanjojo/talhelper").packages.x86_64-linux.default
  ];
}
