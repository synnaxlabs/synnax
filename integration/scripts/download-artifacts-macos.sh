#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# download-artifacts-macos.sh
# Downloads build artifacts for macOS platform and sets up binaries
# Supports both current-run artifacts and reference-run artifacts
# Used by GitHub Actions workflow: test.integration.yaml

set -euo pipefail

echo "Setting up macOS artifacts download..."

# Install GitHub CLI if not present
install_github_cli() {
    if ! command -v gh &> /dev/null; then
        echo "Installing GitHub CLI via Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)" || echo "Homebrew already installed"
        brew install gh
    else
        echo "GitHub CLI already installed"
    fi
}

# Download artifacts from reference run
download_reference_artifacts() {
    local run_id=$1
    echo "Downloading artifacts from reference run: $run_id"
    
    # Create binaries directory
    mkdir -p ./binaries
    
    # Download artifacts using GitHub CLI
    echo "Downloading driver-macos artifact..."
    gh run download $run_id --name driver-macos --dir ./binaries
    
    echo "Downloading synnax-server-macos artifact..."
    gh run download $run_id --name synnax-server-macos --dir ./binaries
    
    echo "Reference artifacts downloaded successfully"
}

# Setup binaries in home directory
setup_binaries() {
    echo "Setting up binaries..."
    
    # Create a binaries directory in a reliable location
    mkdir -p $HOME/synnax-binaries
    cp ./binaries/driver $HOME/synnax-binaries/synnax-driver
    cp ./binaries/synnax-*-macos $HOME/synnax-binaries/synnax
    chmod +x $HOME/synnax-binaries/synnax*
    
    echo "Binaries prepared in $HOME/synnax-binaries:"
    ls -la $HOME/synnax-binaries/synnax*
}

# Download current run artifacts
download_current_artifacts() {
    echo "Downloading current run artifacts..."
    mkdir -p ./binaries
    
    # Use GitHub CLI to download from current run
    echo "Downloading driver-macos artifact..."
    gh run download --name driver-macos --dir ./binaries
    
    echo "Downloading synnax-server-macos artifact..."
    gh run download --name synnax-server-macos --dir ./binaries
    
    echo "Current run artifacts downloaded successfully"
}

# Main execution
main() {
    # Clean up any existing binaries
    if [ -d "./binaries" ]; then
        echo "Cleaning existing binaries directory..."
        rm -rf "./binaries"
    fi
    
    echo "Starting macOS artifacts download and setup..."
    
    install_github_cli
    
    # Debug: Print environment variables
    echo "DEBUG: SKIP_BUILD='${SKIP_BUILD:-}'"
    echo "DEBUG: REF_RUN_ID='${REF_RUN_ID:-}'"
    
    # Check if we should skip build and use reference artifacts
    if [ "${SKIP_BUILD:-false}" = "true" ] && [ -n "${REF_RUN_ID:-}" ]; then
        echo "SKIP build mode: using reference run $REF_RUN_ID"
        download_reference_artifacts "$REF_RUN_ID"
    elif [ "${SKIP_BUILD:-false}" != "true" ]; then
        echo "Build mode: using current run artifacts"
        download_current_artifacts
    else
        echo "ERROR: SKIP_BUILD is true but no REF_RUN_ID provided"
        exit 1
    fi
    
    setup_binaries
    
    echo "macOS artifacts setup completed successfully"
}

# Run main function
main "$@"