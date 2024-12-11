#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <matrix-os>"
  exit 1
fi

os=$1

# Get the root directory of the repository
rootDir="$(git rev-parse --show-toplevel)"

ghBuildPath="$rootDir/bin/gh"

artifactPath="$rootDir/pkg/cmd/attestation/test/data/sigstore-js-2.1.0.tgz"

# Download attestations for the package
if ! $ghBuildPath attestation download "$artifactPath" --owner=sigstore --digest-alg=sha512; then
    # cleanup test data
    echo "Failed to download attestations"
    exit 1
fi

digest=$(shasum -a 512 sigstore-js-2.1.0.tgz.sha512 | awk '{print ""$1""}')

attestation_filename="sha512:$digest.jsonl"
if [ "$os" == "windows-latest" ]; then
  echo "Running the test on Windows."
  echo "Build the expected filename accordingly"
  attestation_filename="sha512-$digest.jsonl"
fi

if [ ! -f "$attestation_filename" ]; then
  echo "Expected attestation file $attestation_filename not found"
  exit 1
fi

if [ ! -s "$attestation_filename" ]; then
  echo "Attestation file $attestation_filename is empty"
  rm "$attestation_filename"
  exit 1
fi

cat "$attestation_filename"

# Clean up the downloaded attestation file
rm "$attestation_filename"
