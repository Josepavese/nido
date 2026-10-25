{
  description = "Nido - local VM manager for AI agents";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f {
        inherit system;
        pkgs = nixpkgs.legacyPackages.${system};
      });

      # The release version lives in CHANGELOG.md; extract it from the first
      # `## [x.y.z]` heading so the flake never needs a manual version bump.
      version = let
        afterHeading = let
          chunks = nixpkgs.lib.splitString "## [" (builtins.readFile ./CHANGELOG.md);
        in
        if builtins.length chunks < 2 then "" else builtins.elemAt chunks 1;
        closing = nixpkgs.lib.splitString "]" afterHeading;
        candidate = if builtins.length closing < 2 then "" else builtins.elemAt closing 0;
      in
      if builtins.match "[0-9]+[.][0-9]+[.][0-9]+" candidate == null then "0.0.0" else candidate;

      mkNido = pkgs: pkgs.buildGoModule {
        pname = "nido";
        inherit version;
        src = nixpkgs.lib.cleanSource ./.;
        subPackages = [ "cmd/nido" "cmd/nido-validator" ];
        vendorHash = "sha256-1jaFQXX2ej4dnc3e4+TnEcj308x0G+8q6rbo/CYbKKI=";
        ldflags = [
          "-s"
          "-w"
          "-X github.com/Josepavese/nido/v5/internal/build.Version=v${version}"
        ];
        nativeBuildInputs = [ pkgs.makeWrapper ];
        # QEMU is a runtime dependency, not a build one; wrapping keeps
        # `nix run github:Josepavese/nido` usable without a shell setup.
        postInstall = ''
          for binary in nido nido-validator; do
            wrapProgram "$out/bin/$binary" \
              --prefix PATH : ${nixpkgs.lib.makeBinPath [ pkgs.qemu ]}
          done
        '';
        meta = with nixpkgs.lib; {
          description = "Local VM manager for AI agents: QEMU/KVM virtual machines through MCP, CLI or TUI";
          homepage = "https://github.com/Josepavese/nido";
          license = licenses.mit;
          mainProgram = "nido";
          platforms = platforms.unix;
        };
      };
    in
    {
      packages = forAllSystems ({ pkgs, ... }: {
        nido = mkNido pkgs;
        default = mkNido pkgs;
      });

      apps = forAllSystems ({ pkgs, ... }: {
        nido = {
          type = "app";
          program = "${(mkNido pkgs)}/bin/nido";
        };
        default = {
          type = "app";
          program = "${(mkNido pkgs)}/bin/nido";
        };
      });

      devShells = forAllSystems ({ pkgs, ... }: {
        default = pkgs.mkShell {
          packages = [ pkgs.go pkgs.gopls pkgs.qemu ];
        };
      });
    };
}
