{ lib, makeWrapper, pkgs, ... }:

let
  openconnect-patched = pkgs.openconnect.overrideAttrs (old: {
    patches = (old.patches or [ ]) ++ [ ./patched/pulse.patch ];
  });

  runtimePaths = [ openconnect-patched ];
  runtimePathString = lib.makeSearchPath "bin" runtimePaths;

  goPkg = pkgs.buildGoModule {
    pname = "go-openconnect-monitor";
    version = "0.1.0";
    src = ../.;
    vendorHash = null;
    subPackages = [ "." ];
  };
in
pkgs.stdenv.mkDerivation {
  pname = "${goPkg.pname}-wrapped";
  version = goPkg.version;
  src = goPkg;
  dontUnpack = true;
  nativeBuildInputs = [ pkgs.makeBinaryWrapper ];

  installPhase = ''
    mkdir -p $out/bin
    cp ${goPkg}/bin/${goPkg.pname} $out/bin/${goPkg.pname}
    wrapProgram $out/bin/${goPkg.pname} \
      --prefix PATH : ${runtimePathString}
  '';

}

