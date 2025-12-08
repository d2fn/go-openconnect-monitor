# nixos-vpn-manager.nix in your vpnManager repo
{ config, lib, pkgs, ... }:

let
  system = pkgs.stdenv.hostPlatform.system;

  # this is the package from your flake
  pkg = config.vpnManager.package or null;
in
{
  options.vpnManager.package = lib.mkOption {
    type = lib.types.package;
    description = "The vpn-manager package (go-openconnect-monitor).";
  };

  config = lib.mkIf (pkg != null) {
    # systemd system service (root)
    systemd.services.vpn-manager = {
      description = "VPN Manager (root openconnect controller)";
      after = [ "network-online.target" ];
      wants = [ "network-online.target" ];

      serviceConfig = {
        Type = "simple";
        User = "root";

        ExecStart = ''
          ${pkg}/bin/go-openconnect-monitor \
            -mode=manage_openconnect \
            -dsid_path=/home/d/.config/vpn-manager/.dsid \
            -config_path=/home/d/.config/vpn-manager/config.toml
        '';

        Restart = "on-failure";
        RestartSec = 2;
      };

      wantedBy = [ "multi-user.target" ];
    };
  };
}
