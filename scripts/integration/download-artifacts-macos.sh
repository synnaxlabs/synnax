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

# Download artifacts from run
download_artifacts() {
    local run_id=$1
    echo "Downloading artifacts from run: $run_id"
    
    # Verify the run exists
    echo "Verifying run $run_id exists..."
    gh run view $run_id
    
    # Create binaries directory
    mkdir -p ./binaries
    
    # Download artifacts using GitHub CLI
    echo "Downloading synnax-core-macos artifact..."
    gh run download $run_id --name synnax-core-macos --dir ./binaries
    
    # Verify artifacts were downloaded
    if [ ! -f "./binaries/synnax-"*"-macos" ]; then
        echo "❌ Error: No synnax executable found in binaries directory"
        echo "Available files in binaries directory:"
        ls -la ./binaries/
        exit 1
    fi
    
    echo "✅ Artifacts downloaded successfully"
}

# Setup binaries in home directory
setup_binaries() {
    echo "Setting up binaries..."
    
    # Create a binaries directory in a reliable location
    mkdir -p $HOME/synnax-binaries
    cp ./binaries/synnax-*-macos $HOME/synnax-binaries/synnax
    chmod +x $HOME/synnax-binaries/synnax*
    
    # Verify setup
    if [ ! -f "$HOME/synnax-binaries/synnax" ]; then
        echo "❌ Error: synnax binary not found in $HOME/synnax-binaries after setup"
        exit 1
    fi
    
    echo "✅ Binaries prepared in $HOME/synnax-binaries:"
    ls -la $HOME/synnax-binaries/synnax*
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
    echo "DEBUG: REF_RUN_ID='${REF_RUN_ID:-}'"
    
    # Download artifacts from the specified run
    if [ -n "${REF_RUN_ID:-}" ]; then
        download_artifacts "$REF_RUN_ID"
    else
        echo "❌ Error: REF_RUN_ID not provided"
        exit 1
    fi
    
    setup_binaries
    
    echo "✅ macOS artifacts setup completed successfully"
}

# Run main function
main "$@"