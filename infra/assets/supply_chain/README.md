# Supply Chain Security Demo Target

A deliberately vulnerable Go program used as the scanning target for the supply chain tooling: `govulncheck`, Trivy, Syft/Grype, and Cosign. Part of the [Supply Chain Security Workshop](../../supply_chain_security_en.md).

> **⚠️ Intentionally outdated dependencies**: the `go.mod` pins a known-vulnerable `golang.org/x/crypto v0.13.0` and Go 1.21 on purpose, so that vulnerability scanners have something real to find. This is a demo target only — do not reuse its dependency choices in any real project.

## What this is

`main.go` configures an SSH client with an insecure host-key callback and prints a message pointing at `govulncheck`. The program itself does nothing harmful; its value is as an object for the security toolchain in the Makefile: source scanning, image scanning, SBOM generation, and signing/verification.

## Requirements

- Go 1.21 (`go.mod`, intentionally old)
- Podman (image build) and the tools invoked by the Makefile: `govulncheck`, `trivy`, `syft`, `grype`, and `cosign` (sign/verify additionally need a key pair)
- Makefile targets: `vuln`, `build`, `scan`, `sbom`, `sign`, `verify`, `all`, `clean`

## How to run

```bash
go run main.go
# Output: the SSH config dump and:
# "Run 'govulncheck ./...' to detect vulnerabilities in this code."

make vuln    # govulncheck: reports the known x/crypto findings
make build   # podman build -t supply-chain-demo:latest
make scan    # trivy image scan (HIGH,CRITICAL)
make sbom    # syft SBOM + grype scan against it
make sign    # cosign sign (requires cosign.key)
make verify  # cosign verify (requires cosign.pub)
```

The [workshop](../../supply_chain_security_en.md) explains each tool and how to read its output.

## Layout

```text
.
├── main.go       # Vulnerable demo program (insecure SSH config)
├── Dockerfile    # Container image for image-scanning exercises
└── Makefile      # vuln / build / scan / sbom / sign / verify
```
