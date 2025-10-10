{
  description = "Go app with a locally patched openconnect";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
  let
    systems = [ "x86_64-linux" "aarch64-linux" ];
    forAllSystems = f: nixpkgs.lib.genAttrs systems
      (system: f (import nixpkgs { inherit system; }));
  in
  {
    packages = forAllSystems (pkgs:
      let
        # ----- Patch openconnect -----
        # Option A: local patch file
        ocPatchedLocal = pkgs.openconnect.overrideAttrs (old: {
          patches = (old.patches or []) ++ [ ./patched/pulse.patch ];
        });

        # Choose which one you want to use:
        oc = ocPatchedLocal;  # or ocPatchedFetched

        pname = "go-openconnect-monitor";
        version = "0.1.0";

        goPkg = pkgs.buildGoModule {
          inherit pname version;
          src = ./.;
          vendorHash = "sha256-xuqWgUQWYQs5Y8NLcb9VjuwM4CYdSJ5Z2iAW/oIA77U=";
          ldflags = [ "-s" "-w" ];
        };
      in {
        default = pkgs.stdenv.mkDerivation {
          pname = "${pname}-wrapped";
          version = goPkg.version;
          src = goPkg;
          dontUnpack = true;
          nativeBuildInputs = [ pkgs.makeBinaryWrapper ];
          installPhase = ''
            mkdir -p $out/bin
            cp ${goPkg}/bin/${pname} $out/bin/${pname}
            wrapProgram $out/bin/${pname} \
              --prefix PATH : ${pkgs.lib.makeBinPath [ oc ]}
          '';
          meta = with pkgs.lib; {
            description = "Wrapped ${pname} with patched openconnect on PATH";
            license = licenses.mit;
            platforms = platforms.linux;
          };
        };
        packages.x86_64-linux.default = pkgs.buildGoModule {
          pname = "go-openconnect-monitor";
          version = "1.0.0";
          src = ./.;
          vendorSha256 = null;
        };
        apps.x86_64-linux.default = {
          type = "app";
          program = "${self.packages.x86_64-linux.default}/bin/go-openconnect-monitor";
        };
      });

    apps = forAllSystems (pkgs: {
      default = {
        type = "app";
        program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.default}/bin/go-openconnect-monitor";
      };
    });

    devShells = forAllSystems (pkgs:
      let
        oc = pkgs.openconnect.overrideAttrs (old: {
          patches = (old.patches or []) ++ [ ./patched/pulse.patch ];
        });
      in {
        default = pkgs.mkShell {
          packages = [
            pkgs.go_1_25
						pkgs.gotools
            oc
            pkgs.git
          ];
					GOTOOLCHAIN = "local";
        };
      });

    formatter = forAllSystems (pkgs: pkgs.alejandra);
  };
}

