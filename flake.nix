{
  description = "Go app with a locally patched openconnect";

	inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs, ... }:
  let
    system = "x86_64-linux";
    pkgs = import nixpkgs {
			inherit system;
		};
  in {
    # HM module
    homeManagerModules.vpnManager = ./hm-vpn-manager.nix;
    # Package
    packages.${system} = {
			vpnManager = pkgs.callPackage ./default.nix { };
			default = self.packages.${system}.vpnManager;
		};
  };
}

