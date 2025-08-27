# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# run-test-conductor-windows.ps1
# Runs the integration test conductor on Windows
# Used by GitHub Actions workflow: test.integration.yaml

$ErrorActionPreference = "Stop"

Write-Host "🧪 Running integration test conductor on Windows..."

# Change to the integration test directory
Set-Location "integration/test/py"

# Verify Synnax is still running
$synnaxPid = Get-Content "$env:USERPROFILE\synnax-pid.txt" -ErrorAction SilentlyContinue
if ($synnaxPid -and (Get-Process -Id $synnaxPid -ErrorAction SilentlyContinue)) {
    Write-Host "Synnax is running with PID: $synnaxPid"
} else {
    Write-Host "❌ ERROR: Synnax process not running"
    exit 1
}

# Run tests with UTF-8 encoding
$env:PYTHONIOENCODING = "utf-8"
$env:PYTHONUTF8 = "1"
$env:PATH = "$env:APPDATA\Python\Scripts;$env:USERPROFILE\.local\bin;$env:PATH"

poetry run test-conductor --name test-conductor-windows --sequence testcases/basic_tests.json

Write-Host "✅ Integration test conductor completed successfully"