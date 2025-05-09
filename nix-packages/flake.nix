{
  description = "flakes for this project";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = inputs @ { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      with nixpkgs.legacyPackages.${system}; rec {
        packages = rec {
          release-please-cli = buildNpmPackage rec {
            pname = "release-please";
            version = "17.0.0";
            src = fetchFromGitHub {
              owner = "googleapis";
              repo = "release-please";
              rev = "v${version}";
              hash = "sha256-/d02gnrKyFJ0rc3Tr6MEOw8hx5ab1xNIfmy0dpiVnIs=";
            };
            npmDepsHash = "sha256-xLG+he/kFJrS24WdPzUiqO3hYynZYy5HGhFpsVopIOA=";
            dontNpmBuild = true;
          };
          default = packages.release-please-cli;
        };
      }
    );
}
