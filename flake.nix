{
	description = "Go app with a locally patched openconnect";

	inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

	outputs = { self, nixpkgs }:
	let
		systems = [ "x86_64-linux" "aarch64-linux" ];
		forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f (import nixpkgs { inherit system; }));
	in
	{
		packages = forAllSystems (pkgs:
			let
				# Patched openconnect
				oc = pkgs.openconnect.overrideAttrs (old: {
					patches = (old.patches or []) ++ [ ./patched/pulse.patch ];
				});

				pname = "go-openconnect-monitor";
				version = "0.1.0";

				goPkg = pkgs.buildGoModule {
					inherit pname version;
					src = ./.;
					vendorHash = null; # fill with the hash Nix prints once you vendor/lock
					ldflags = [ "-s" "-w" ];
					meta.mainProgram = pname;
				};

				# Wrapper that puts patched openconnect on PATH for the runtime
				wrapped = pkgs.stdenv.mkDerivation {
					pname = "${pname}-wrapped";
					version = goPkg.version;
					src = goPkg;	dontUnpack = true;
					nativeBuildInputs = [ pkgs.makeBinaryWrapper ];
					installPhase = ''
						mkdir -p $out/bin
						cp ${goPkg}/bin/${pname} $out/bin/${pname}
						wrapProgram $out/bin/${pname} \
							--prefix PATH : ${pkgs.lib.makeBinPath [ oc ]}
					'';
					meta = goPkg.meta;
				};

				# App launchers (paths), one for each profile
				openconnectApp = pkgs.writeShellApplication {
					name = "openconnect-app";
					text = ''
						# Patched openconnect already on PATH via wrapper closure (see above).
						exec ${wrapped}/bin/${pname} "$@"
					'';
				};

				pollerApp = pkgs.writeShellApplication {
					name = "poller-app";
					text = ''
						exec ${wrapped}/bin/${pname} poll_cookies "$@"
					'';
				};
			in {
				default = wrapped;
				# expose the launchers as buildable packages if you want
				openconnect-app = openconnectApp;
				poller-app = pollerApp;
			});

		apps = forAllSystems (pkgs: {
			default = {
				type = "app";
				program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.openconnect-app}/bin/openconnect-app";
			};
			openconnect = {
				type = "app";
				program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.openconnect-app}/bin/openconnect-app";
			};
			poller = {
				type = "app";
				program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.poller-app}/bin/poller-app";
			};
		});

		devShells = forAllSystems (pkgs:
			let oc = pkgs.openconnect.overrideAttrs (old: {
				patches = (old.patches or []) ++ [ ./patched/pulse.patch ];
			});
			in {
				default = pkgs.mkShell {
					packages = [ pkgs.go_1_25 pkgs.gotools oc pkgs.git ];
					GOTOOLCHAIN = "local";
				};
			});

		formatter = forAllSystems (pkgs: pkgs.alejandra);
	};
}

