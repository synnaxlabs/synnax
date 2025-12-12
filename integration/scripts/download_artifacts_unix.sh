#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

set -euo pipefail

echo "Setting up artifacts download..."

# Install GitHub CLI if not present
install_github_cli() {
    if ! command -v gh &> /dev/null; then
        echo "Installing GitHub CLI..."
        curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
        sudo apt update
        sudo apt install gh -y
    else
        echo "GitHub CLI already installed"
    fi
}

# Detect OS and set artifact name
get_artifact_name() {
    if [[ "$(uname)" == "Darwin" ]]; then
        echo "macos-core"
    else
        echo "linux-core"
    fi
}

# Download artifacts from run
download_artifacts() {
    local run_id=$1
    local artifact_name=$(get_artifact_name)
    echo "Downloading artifacts from run: $run_id"

    # Verify the run exists
    echo "Verifying run $run_id exists..."
    gh run view $run_id

    # Create binaries directory
    mkdir -p ./binaries

    # Download artifacts using GitHub CLI
    echo "Downloading $artifact_name artifact..."
    gh run download $run_id --name $artifact_name --dir ./binaries

    # Verify artifacts were downloaded (binary is named synnax-v{VERSION})
    if ! ls ./binaries/synnax-v* 1> /dev/null 2>&1; then
        echo "❌ Error: No synnax executable found in binaries directory"
        echo "Available files in binaries directory:"
        ls -la ./binaries/ || echo "No ./binaries directory found"
        exit 1
    fi

    echo "✅ Artifacts downloaded successfully"
}

# Setup binaries in home directory
setup_binaries() {
    echo "Setting up binaries..."

    # Create a binaries directory in a reliable location
    mkdir -p $HOME/synnax-binaries

    # Debug: Check what's in binaries directory
    echo "Contents of ./binaries directory:"
    ls -la ./binaries/ || echo "No ./binaries directory found"

    # Copy the synnax binary (binary is named synnax-v{VERSION})
    if ls ./binaries/synnax-v* 1> /dev/null 2>&1; then
        cp ./binaries/synnax-v* $HOME/synnax-binaries/synnax
        echo "✅ Server binary copied successfully"
    else
        echo "❌ Error: Server binary (synnax-v*) not found in ./binaries/"
        exit 1
    fi

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
    else
        echo "No existing binaries directory to clean"
    fi

    local os_name="Linux"
    if [[ "$(uname)" == "Darwin" ]]; then
        os_name="macOS"
    fi
    echo "Starting $os_name artifacts download and setup..."

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

    echo "✅ $os_name artifacts setup completed successfully"
}

# Run main function
main "$@"
