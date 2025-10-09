# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

$ErrorActionPreference = "Stop"

Write-Host "Starting Synnax server on Windows..."

# Create data directory
if (-not (Test-Path "$env:USERPROFILE\synnax-data")) {
    New-Item -ItemType Directory -Path "$env:USERPROFILE\synnax-data"
}

# Start Synnax using PowerShell Start-Process which creates a truly detached process
Set-Location "$env:USERPROFILE\synnax-data"
$synnaxPath = "$env:USERPROFILE\Desktop\synnax.exe"

Write-Host "Starting Synnax server..."

# Verify the Synnax binary exists
if (-not (Test-Path $synnaxPath)) {
    Write-Host "ERROR: Synnax binary not found at: $synnaxPath"
    Write-Host "Checking for available binaries..."
    
    # List desktop contents
    if (Test-Path "$env:USERPROFILE\Desktop") {
        Write-Host "Desktop contents:"
        Get-ChildItem "$env:USERPROFILE\Desktop" | Format-Table Name, Length, LastWriteTime
    }
    
    # List current directory contents  
    Write-Host "Current directory contents:"
    Get-ChildItem . | Format-Table Name, Length, LastWriteTime
    
    exit 1
}

Write-Host "Found Synnax binary at: $synnaxPath"
$process = Start-Process -FilePath $synnaxPath -ArgumentList "start", "-mi" -WindowStyle Hidden -PassThru -WorkingDirectory "$env:USERPROFILE\synnax-data"

# Store the process ID for tracking
$process.Id | Out-File -FilePath "$env:USERPROFILE\synnax-pid.txt" -Encoding ASCII
Write-Host "Started Synnax with PID: $($process.Id)"

# Wait for startup and verify it's still running
Write-Host "Waiting for server startup..."
Start-Sleep -Seconds 5

$synnaxProcess = Get-Process -Id $process.Id -ErrorAction SilentlyContinue
if ($synnaxProcess) {
    Write-Host "Synnax is running with PID: $($synnaxProcess.Id)"
    
    # Verify port 9090 is listening
    $portReady = $false
    for ($i = 1; $i -le 5; $i++) {
        $connection = Test-NetConnection -ComputerName localhost -Port 9090 -WarningAction SilentlyContinue
        if ($connection.TcpTestSucceeded) {
            Write-Host "Port 9090 is ready"
            $portReady = $true
            break
        }
        Write-Host "Waiting for port 9090... (attempt $i/5)"
        Start-Sleep -Seconds 3
    }
    
    if (-not $portReady) {
        Write-Host "ERROR: Port 9090 never became available"
        exit 1
    }
} else {
    Write-Host "ERROR: Synnax process died during startup"
    exit 1
}

Write-Host "Synnax server started successfully and is ready!"

# Output Synnax version
Write-Host "Synnax version:"
& "$env:USERPROFILE\Desktop\synnax.exe" version