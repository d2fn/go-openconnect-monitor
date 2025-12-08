{ config, lib, ... }:
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
}
