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
  options.vpnManager = {
    enable = lib.mkEnableOption "VPN manager Go app";
		vpn = {
			url = lib.mkOption {
				type = lib.types.str;
				description = "Pulse VPN URL";
			};
		};
		openconnect = {
			verbose = lib.mkOption {
				type = lib.types.bool;
				default = true;
				description = "Enable verbose output from openconnect";
			};
			extraArgs = lib.mkOption {
				type = lib.types.str;
				default = "";
				description = "Add extra args to openconnect";
			};
			shutdownGracePeriodSeconds = lib.mkOption {
				type = lib.types.int;
				default = 10;
				description = "When manually shutting down openconnect, the number of seconds we wait to force kill";
			};
			dryRun = lib.mkOption {
				type = lib.types.bool;
				default = false;
				description = "Don't attempt to connect, just show commands that would be run";
			};
		};
		controller = {
			intervalSeconds = lib.mkOption {
				type = lib.types.int;
				default = 1;
				description = "Event loop timeout";
			};
			healthCheckGracePeriodSeconds = lib.mkOption {
				type = lib.types.int;
				default = 1;
				description = "Number of seconds that health checks must fail before killing openconnect";
			};
		};
		healthCheck = {
			host = lib.mkOption {
				type = lib.types.str;
				default = "192.173.91.18";
				description = "Host to dial for health check";
			};
			port = lib.mkOption {
				type = lib.types.str;
				default = "443";
				description = "Port to dial on health check host";
			};
			timeoutSeconds = lib.mkOption {
				type = lib.types.int;
				default = 2;
				description = "Dial timeout for health check";
			};
		};
		dsidCookiePoller = {
			cookieName = lib.mkOption {
				type = lib.types.str;
				default = "DSID";
				description = "Cookie name for DSID";
			};
			cookiePath = lib.mkOption {
				type = lib.types.str;
				description = "Path to Chrome cookies";
			};
			cookieHost = lib.mkOption {
				type = lib.types.str;
				description = "Domain under which cookie is stored";
			};
		};
  };

  config = lib.mkIf cfg.enable {

		# this is a *path* to a generated TOML file in the store
		xdg.configFile."vpn-manager/config.toml".source = vpnConfigToml;

		# add btop config for vpn-btop below
		xdg.configFile."vpn-manager/btop.config".source = ./config/btop.config;

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
