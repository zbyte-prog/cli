#!/usr/bin/env bash
set -euo pipefail

# Get the root directory of the repository
rootDir="$(git rev-parse --show-toplevel)"

ghBuildPath="$rootDir/bin/gh"

# Verify an OCI artifact with bundles stored on the GHCR OCI registry
echo "Testing with package $sigstore02PackageFile and attestation $sigstore02AttestationFile"
if ! $ghBuildPath attestation verify oci://ghcr.io/malancas/attest-demo:latest --owner=malancas --bundle-from-oci; then
    echo "Failed to verify oci://ghcr.io/malancas/attest-demo:latest with bundles from the GHCR OCI registry"
    exit 1
fi
