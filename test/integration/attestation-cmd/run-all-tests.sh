#!/usr/bin/env bash
set -euo pipefail

# Get the root directory of the repository
rootDir="$(git rev-parse --show-toplevel)"

verify_test_dir="$rootDir/test/integration/attestation-cmd/verify"
for script in "$verify_test_dir"/*.sh; do
  if [ -f "$script" ]; then
    echo "Running $script..."
    $script
  fi
done
