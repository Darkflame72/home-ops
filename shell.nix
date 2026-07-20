{
}:
let
  pkgs = import (fetchTarball "https://nixos.org/channels/nixpkgs-unstable/nixexprs.tar.xz") { };

  # kopiur is not packaged in nixpkgs; fetch the prebuilt release binary.
  # https://github.com/home-operations/kopiur
  kopiurVersion = "0.7.5";
  kopiurPlatforms = {
    x86_64-linux = {
      target = "linux_amd64";
      sha256 = "758b79899a6ab359b46db9131410c6be7eb16e00c2e02975fb3e840e1aa8f6e6";
    };
    aarch64-linux = {
      target = "linux_arm64";
      sha256 = "fae3dfc02be88b0895f8efc5ff6325e6495c64bb4585fe384d317e8b1875f094";
    };
    x86_64-darwin = {
      target = "darwin_amd64";
      sha256 = "b263e2680b2852eb11e9af311e907d4b18a96f7f398bef48f327e5fdaa1d83a3";
    };
    aarch64-darwin = {
      target = "darwin_arm64";
      sha256 = "ba03d17db4fa10e5a67ce98fbb787b6beee8d97ade2ada0ef9a542fa1734bad5";
    };
  };
  kopiurPlatform = kopiurPlatforms.${pkgs.stdenv.hostPlatform.system};

  kopiur = pkgs.stdenv.mkDerivation {
    pname = "kopiur";
    version = kopiurVersion;

    src = pkgs.fetchurl {
      url = "https://github.com/home-operations/kopiur/releases/download/${kopiurVersion}/kubectl-kopiur_${kopiurVersion}_${kopiurPlatform.target}.tar.gz";
      sha256 = kopiurPlatform.sha256;
    };

    sourceRoot = ".";
    dontBuild = true;

    installPhase = ''
      install -Dm755 kopiur $out/bin/kopiur
    '';

    meta = {
      description = "A Kopia-native Kubernetes backup operator, written in Rust";
      homepage = "https://kopiur.home-operations.com/";
    };
  };

in
pkgs.mkShell {
  packages = [
    kopiur
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
    pkgs.cosign
    pkgs.kopia
  ];

  KUBECONFIG = "${toString ./.}/kubeconfig";
  SOPS_AGE_KEY_FILE = "${toString ./.}/age.key";
  TALOSCONFIG = "${toString ./.}/talos/clusterconfig/talosconfig";

}
