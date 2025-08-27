# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# debug-test-environment-windows.ps1
# Provides debugging information when tests fail on Windows
# Used by GitHub Actions workflow: test.integration.yaml

Write-Host "🔍 Test failed - debugging info:"
Write-Host "Matrix OS: windows"

Write-Host "=== Python/Poetry environment ==="
python --version
$poetryPath1 = "$env:APPDATA\Python\Scripts"
$poetryPath2 = "$env:USERPROFILE\.local\bin" 
$env:PATH = "$poetryPath1;$poetryPath2;$env:PATH"
poetry --version

Write-Host "=== Synnax connectivity ==="
try {
    $connection = Test-NetConnection -ComputerName localhost -Port 9090 -WarningAction SilentlyContinue
    if ($connection.TcpTestSucceeded) {
        Write-Host "✅ Port 9090 reachable"
    } else {
        Write-Host "❌ Port 9090 unreachable"
    }
} catch {
    Write-Host "Cannot test port 9090"
}

Write-Host "=== Service status ==="
Get-Process | Where-Object {$_.ProcessName -like "*synnax*"} | Format-Table -Property ProcessName, Id
if (-not (Get-Process | Where-Object {$_.ProcessName -like "*synnax*"})) {
    Write-Host "No synnax processes found"
}

Write-Host "=== Test artifacts ==="
if (Test-Path "integration/test/py") {
    Get-ChildItem "integration/test/py" -File | Select-Object Name, Length, LastWriteTime | Format-Table
} else {
    Write-Host "No test directory found"
}

Write-Host "🔍 Debug information collection completed"