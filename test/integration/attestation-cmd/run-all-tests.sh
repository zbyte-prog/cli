#!/usr/bin/env bash
set -euo pipefail

# Get the root directory of the repository
rootDir="$(git rev-parse --show-toplevel)"

attestation_cmd_test_dir="$rootDir/test/integration/attestation-cmd"
for script in "$attestation_cmd_test_dir"/*.sh; do
  if [ -f "$script" ]; then
    echo "Running $script..."
    bash "$script"
  fi
done
