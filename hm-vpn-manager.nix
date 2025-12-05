# hm-vpn-manager.nix
{ config, lib, pkgs, vpnManager, ... }:

let
  cfg = config.vpnManager;
  system = pkgs.stdenv.hostPlatform.system;
  pkg = vpnManager.packages.${system}.vpnManager;
in
{
  options.vpnManager = {
    enable = lib.mkEnableOption "VPN manager Go app";
    enablePoller = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Expose vpn-dsid-poller command.";
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages =
      [
        pkg

        # vpn-manager wrapper script (no extra args)
        (pkgs.writeShellScriptBin "vpn-manager" ''
          exec "${pkg}/bin/go-openconnect-monitor" "$@"
        '')

				(pkgs.writeShellScriptBin "vpn-dsid-poller" ''
          exec "${pkg}/bin/go-openconnect-monitor" poller "$@"
        '')
      ];
  };
}
