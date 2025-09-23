{
  description = "Age-based secrets management";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      packages.${system}.default = pkgs.buildGoModule {
        pname = "secrets";
        version = "1.0.0";
        src = ./.;
        vendorHash = null;

        # Rename binary from go-secrets to secrets
        postInstall = ''
          mv $out/bin/go-secrets $out/bin/secrets
        '';
      };
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = [ pkgs.go pkgs.age ];
      };
    };
}
