#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# download-artifacts-linux.sh
# Downloads build artifacts for Linux platform and sets up binaries
# Used by GitHub Actions workflow: test.integration.yaml

set -euo pipefail

echo "🔧 Setting up Linux artifacts download..."

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

# Setup binaries in home directory
setup_binaries() {
    echo "Setting up binaries..."
    
    # Create a binaries directory in a reliable location
    mkdir -p $HOME/synnax-binaries
    cp ./binaries/driver $HOME/synnax-binaries/synnax-driver
    cp ./binaries/synnax-*-linux $HOME/synnax-binaries/synnax
    chmod +x $HOME/synnax-binaries/synnax*
    
    echo "Binaries prepared in $HOME/synnax-binaries:"
    ls -la $HOME/synnax-binaries/synnax*
}

# Main execution
main() {
    echo "Starting Linux artifacts download and setup..."
    
    install_github_cli
    setup_binaries
    
    echo "✅ Linux artifacts setup completed successfully"
}

# Run main function
main "$@"