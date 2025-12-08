# hm-vpn-manager.nix
{ config, lib, pkgs, vpnManager, ... }:

let
  cfg = config.vpnManager;
  system = pkgs.stdenv.hostPlatform.system;
  pkg = vpnManager.packages.${system}.vpnManager;
	tomlFormat = pkgs.formats.toml { };
	vpnConfigToml = tomlFormat.generate "vpn-manager-config.toml" cfg;
in
{

  imports = [
    ./options.nix
  ];

  config = lib.mkIf cfg.enable {

		# symlink from xdg to the generated toml for the generation
		xdg.configFile."vpn-manager/config.toml".source = vpnConfigToml;

		# add btop config for vpn-btop below
		xdg.configFile."vpn-manager/btop.config".source = ../config/btop.config;

    systemd.user.services.vpn-dsid-poller = {
      Unit = {
        Description = "VPN DSID cookie poller";
        After = [ "network-online.target" ];
      };

      Service = {
        Type = "simple";

        # %h = home dir, so this hits $HOME/.config/vpn-manager/...
        ExecStart = ''
          ${pkg}/bin/go-openconnect-monitor \
            --mode=poll_cookies \
            --dsid_path=%h/.config/vpn-manager/.dsid \
            --config_path=%h/.config/vpn-manager/config.toml
        '';

        Restart = "on-failure";
        RestartSec = 2;
      };

      Install.WantedBy = [ "default.target" ];
    };

    home.packages =
      [
        pkg

        # vpn-manager
				# runs under sudo to manage the network interface
        (pkgs.writeShellScriptBin "vpn-manager" ''
          exec sudo "${pkg}/bin/go-openconnect-monitor" \
						-mode=manage_openconnect \
						-dsid_path=$XDG_CONFIG_HOME/vpn-manager/.dsid \
						-config_path=$XDG_CONFIG_HOME/vpn-manager/config.toml \
						"$@"
        '')

				# vpn-dsid-poller
				# runs under user account so that cookies can be decrypted using AES keys
				(pkgs.writeShellScriptBin "vpn-dsid-poller" ''
          exec "${pkg}/bin/go-openconnect-monitor" \
						--mode=poll_cookies \
						--dsid_path="$XDG_CONFIG_HOME/vpn-manager/.dsid" \
						--config_path="$XDG_CONFIG_HOME/vpn-manager/config.toml" \
						"$@"
        '')

				# vpn-btop
				# runs under user account so that cookies can be decrypted using AES keys
				(pkgs.writeShellScriptBin "vpn-btop" ''
					#!/usr/bin/env bash
					echo "Waiting for VPN tunnel tun0 to come up..."
					while ! ip route show | grep tun0; do
						sleep 1
					done
					btop -c $XDG_CONFIG_HOME/vpn-manager/btop.config
        '')

				# vpn-reset-dns
				# hard reset any lingering dns config left by the vpn
				(pkgs.writeShellScriptBin "vpn-reset-dns" ''
					#!/usr/bin/env bash
					sudo tee /etc/resolv.conf >/dev/null <<'EOF'
					nameserver 1.1.1.1
					nameserver 8.8.8.8
					EOF
        '')

				# vpn-manager-tmux
				# runs under user account so that cookies can be decrypted using AES keys
				(pkgs.writeShellScriptBin "vpn-manager-tmux" ''
					#!/usr/bin/env bash
					session_name="vpn"
					tmux new-session  -d -s $session_name -n btop 'vpn-btop'
					tmux split-window -v -t $session_name:1 'vpn-dsid-poller'
					tmux split-window -v -t $session_name:1 'vpn-manager'
					tmux attach -t $session_name
        '')
      ];
  };
}
