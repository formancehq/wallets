{
  description = "Wallets dev env";

  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.2511";
    nixpkgs-unstable.url = "https://flakehub.com/f/NixOS/nixpkgs/0.1";

    nur = {
      url = "github:nix-community/NUR";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, nur }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forEachSupportedSystem = f:
        nixpkgs.lib.genAttrs supportedSystems (system:
          let
            pkgs = import nixpkgs {
              inherit system;
              overlays = [ nur.overlays.default ];
              config.allowUnfreePredicate = pkg: builtins.elem (nixpkgs.lib.getName pkg) [
                "goreleaser-pro"
              ];
            };
            pkgs-unstable = import nixpkgs-unstable {
              inherit system;
            };
          in
          f { pkgs = pkgs; pkgs-unstable = pkgs-unstable; system = system; }
        );

      speakeasyVersion = "1.351.0";
      speakeasyPlatforms = {
        "x86_64-linux"   = "linux_amd64";
        "aarch64-linux"  = "linux_arm64";
        "x86_64-darwin"  = "darwin_amd64";
        "aarch64-darwin" = "darwin_arm64";
      };
      speakeasyHashes = {
        "x86_64-linux"   = "sha256-eeGzoIlsgsW/wnEET+UFgNN1qWgndxRHzulhHDTyHRY=";
        "aarch64-linux"  = "sha256-zOj2QUwLwRz0MyTLdVxLaGU7XEqhgKLCyhsO9S8VCNk=";
        "x86_64-darwin"  = "sha256-vBgEv6WWwJhBW6nMLy4Nj7qjWdGqk/4al5dIUCqrm1I=";
        "aarch64-darwin" = "sha256-N129T0BDRVUXxH6Dl58/hUEApiq1q2B6qTwTuEjLDi4=";
      };

    in
    {
      packages = forEachSupportedSystem ({ pkgs, pkgs-unstable, system }:
        {
          speakeasy = pkgs.stdenv.mkDerivation {
            pname = "speakeasy";
            version = speakeasyVersion;

            src = pkgs.fetchurl {
              url = "https://github.com/speakeasy-api/speakeasy/releases/download/v${speakeasyVersion}/speakeasy_${speakeasyPlatforms.${system}}.zip";
              sha256 = speakeasyHashes.${system};
            };

            nativeBuildInputs = [ pkgs.unzip ];
            dontUnpack = true;

            installPhase = ''
              mkdir -p $out/bin
              unzip $src
              ls -al
              install -m755 speakeasy $out/bin/
            '';

            name = "speakeasy";
          };
        }
      );

      defaultPackage.x86_64-linux   = self.packages.x86_64-linux.speakeasy;
      defaultPackage.aarch64-linux  = self.packages.aarch64-linux.speakeasy;
      defaultPackage.x86_64-darwin  = self.packages.x86_64-darwin.speakeasy;
      defaultPackage.aarch64-darwin = self.packages.aarch64-darwin.speakeasy;

      devShells = forEachSupportedSystem ({ pkgs, pkgs-unstable, system }:
        let
          stablePackages = with pkgs; [
            ginkgo
            go_1_26
            gotools
            just
          ];
          unstablePackages = with pkgs-unstable; [
            golangci-lint
          ];
          otherPackages = [
            pkgs.nur.repos.goreleaser.goreleaser-pro
            self.packages.${system}.speakeasy
          ];
        in
        {
          default = pkgs.mkShell {
            packages = stablePackages ++ unstablePackages ++ otherPackages;
          };
        }
      );
    };
}
