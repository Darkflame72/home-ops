{
  description = "home-ops";

  inputs = {
    nixpkgs = { url = "github:NixOS/nixpkgs/nixos-unstable"; };
    talhelper = { url = "github:budimanjojo/talhelper"; inputs.nixpkgs.follows = "nixpkgs"; };
  };

  outputs = { self, nixpkgs, ... }@inputs:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      talhelper = inputs.talhelper.packages.${system}.default;
    in
    rec {
      # Accessible via 'nix develop' or 'nix shell'
      packages.${system} = {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
    direnv
    age
    flux
    helmfile
    go-task
    sops
    jq
    fluxcd
    python3
    kubectl
    kubernetes-helm
    kubeconform
    kustomize
    talosctl
    talhelper
    cloudflared
    envsubst
          ];
        };
      };
    };
}
