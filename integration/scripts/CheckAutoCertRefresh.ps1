# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms --auto-cert reissues a node certificate that no longer covers the listen
# address, and the embedded Driver connects after the replacement. The first boot
# doubles as the plain --auto-cert happy path over TLS.

$ErrorActionPreference = "Stop"

$binary = "$env:USERPROFILE\synnax-binaries\synnax.exe"
$work = "$env:USERPROFILE\synnax-auto-cert-refresh-check"
$port = 9096
$deadline = 60

if (Test-Path $work) { Remove-Item -Recurse -Force $work }
New-Item -ItemType Directory -Path $work -Force | Out-Null

function Start-Core($listen, $log) {
    $cmdArgs = "/c `"`"$binary`" start -m -d `"$work\data`" --auto-cert --listen $listen --certs-dir `"$work\certs`" --log-file-path `"$work\logs\synnax.log`" > `"$log`" 2>&1`""
    Start-Process -FilePath "cmd.exe" -ArgumentList $cmdArgs -WindowStyle Hidden -PassThru
}

# Stop-Process alone kills only the cmd.exe wrapper; the Core child keeps the port.
function Stop-Core($process) {
    if (-not $process.HasExited) {
        & taskkill /PID $process.Id /T /F 2>$null | Out-Null
    }
    $process.WaitForExit()
}

Write-Host "Starting Synnax with --auto-cert on localhost:$port..."
$log1 = "$work\core1.log"
$process = Start-Core "localhost:$port" $log1

try {
    $started = $false
    for ($i = 0; $i -lt $deadline; $i++) {
        if ((Test-Path $log1) -and (Select-String -Path $log1 -Pattern "started successfully" -Quiet)) {
            $started = $true
            break
        }
        if ($process.HasExited) { break }
        Start-Sleep -Seconds 1
    }
    if (-not $started) {
        Write-Host "ERROR: the first boot never started"
        if (Test-Path $log1) { Get-Content $log1 -Tail 40 }
        exit 1
    }
    Stop-Core $process

    Write-Host "Restarting on 127.0.0.1:$port to force a certificate refresh..."
    $log2 = "$work\core2.log"
    $process = Start-Core "127.0.0.1:$port" $log2

    for ($i = 0; $i -lt $deadline; $i++) {
        if ((Test-Path $log2) -and (Select-String -Path $log2 -Pattern "started successfully" -Quiet)) {
            if (Select-String -Path $log2 -Pattern "replacing node certificate" -Quiet) {
                Write-Host "Node certificate refreshed and the embedded Driver reconnected"
                exit 0
            }
            Write-Host "ERROR: the Core restarted without reissuing the node certificate"
            Get-Content $log2 -Tail 40
            exit 1
        }
        if ($process.HasExited) { break }
        Start-Sleep -Seconds 1
    }

    Write-Host "ERROR: the second boot never started"
    if (Test-Path $log2) {
        Write-Host "--- certificate failures ---"
        Select-String -Path $log2 -Pattern "certificate|advertised|cover" |
            Select-Object -First 10 |
            ForEach-Object { $_.Line.Substring(0, [Math]::Min(300, $_.Line.Length)) }
        Write-Host "--- last 40 lines ---"
        Get-Content $log2 -Tail 40
    }
    exit 1
} finally {
    Stop-Core $process
}
