# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# install-poetry-deps-windows.ps1
# Installs Poetry and Python dependencies on Windows
# Used by GitHub Actions workflow: test.integration.yaml

$ErrorActionPreference = "Stop"

Write-Host "📦 Installing Poetry and dependencies on Windows..."

# Change to the integration test directory
Set-Location "integration/test/py"

# Install Poetry on Windows
(Invoke-WebRequest -Uri https://install.python-poetry.org -UseBasicParsing).Content | python -

# Add Poetry to PATH
$env:PATH = "$env:APPDATA\Python\Scripts;$env:USERPROFILE\.local\bin;$env:PATH"

# Remove existing lock file and recreate it fresh
if (Test-Path "poetry.lock") { 
    Remove-Item "poetry.lock" 
}

poetry install

Write-Host "✅ Poetry and dependencies installed successfully"