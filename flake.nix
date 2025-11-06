{
  description = "bky-c";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";

    # this tool allows us use nix-shell and nix shell
    # and is used for our shell.nix
    flake-compat.url = "https://flakehub.com/f/edolstra/flake-compat/1.tar.gz";

  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

      in
      {
        packages.bky-c = pkgs.buildGoModule {
          pname = "bky-c";
          version = "0.1.0-prerelease";

          src = ./.;

          vendorHash = "sha256-yIs80Le6rbksqZyNF1PA7DPkKokz+QYexH1UwkItncU=";
          doCheck = false;
        };

        devShells.default = pkgs.mkShell {
          name = "bky-c-dev-shell";
          packages = [
            pkgs.go
            pkgs.gotools
            pkgs.golangci-lint
            pkgs.go-licenses
            pkgs.go-mockery
            pkgs.goreleaser
            pkgs.docker_27
          ];
        };
      }
    );
}
