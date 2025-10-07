{
  description = "Go app with a locally patched openconnect";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";

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
          vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
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
      });

    apps = forAllSystems (pkgs: {
      default = {
        type = "app";
        program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.default}/bin/my-go-app";
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
            pkgs.go
            oc
            pkgs.git
          ];
        };
      });

    formatter = forAllSystems (pkgs: pkgs.alejandra);
  };
}

