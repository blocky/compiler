{
  description = "bky-c";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-24.11";
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

        # read go version from go.mod
        goModVersion = builtins.match ".*go ([0-9]+\\.[0-9]+).*"
          (builtins.readFile ./go.mod);

        goVersion = if goModVersion != null && goModVersion != [] then
          builtins.head goModVersion
        else
          "1.23";

        goPkgVersion = "go_${builtins.replaceStrings ["."] ["_"] goVersion}";
        go = pkgs.${goPkgVersion};

        docker = pkgs.docker_27;
      in
      {
        devShells.default = pkgs.mkShell {
          name = "bky-c-dev-shell";
          packages = [
            go
            pkgs.gotools
            pkgs.golangci-lint
            pkgs.go-licenses
            pkgs.go-mockery
            pkgs.goreleaser
            docker
          ];
        };
      }
    );
}
