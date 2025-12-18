{
  description = "Go app with a locally patched openconnect";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs {
        inherit system;
      };
    in
    {
      homeManagerModules.vpnManager = ./modules/hm-vpn-manager.nix;
      nixosModules.vpnManager = ./modules/nixos-vpn-manager.nix;
      # Package
      packages.${system} = {
        vpnManager = pkgs.callPackage ./pkgs/vpn-manager.nix { };
        default = self.packages.${system}.vpnManager;
      };
    };
}
