# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms the Windows service story: every daemon command exits 1 with the
# not-supported error, and the documented alternative — login plus start
# --standalone — reaches a Core.

$ErrorActionPreference = "Stop"

$core = "$env:USERPROFILE\synnax-binaries\synnax.exe"
$driver = "$env:USERPROFILE\synnax-binaries\synnax-driver.exe"
$work = "$env:USERPROFILE\synnax-driver-service-check"
$port = 9094
$deadline = 60

if (Test-Path $work) { Remove-Item -Recurse -Force $work }
New-Item -ItemType Directory -Path $work -Force | Out-Null

foreach ($cmd in @("install", "uninstall", "stop", "restart", "status", "logs", "start")) {
    $out = (cmd /c "`"$driver`" $cmd 2>&1" | Out-String)
    if ($LASTEXITCODE -eq 0) {
        Write-Host "ERROR: 'synnax-driver $cmd' succeeded but must be rejected on Windows"
        exit 1
    }
    if ($out -notmatch "not supported") {
        Write-Host "ERROR: 'synnax-driver $cmd' failed without the not-supported message"
        Write-Host $out
        exit 1
    }
}
Write-Host "Service commands are rejected with the not-supported error"

$out = (cmd /c "`"$driver`" version 2>&1" | Out-String)
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: 'synnax-driver version' failed"
    Write-Host $out
    exit 1
}

Write-Host "Starting an insecure Core on localhost:$port..."
$coreLog = "$work\core.log"
$coreArgs = "/c `"`"$core`" start -mi --listen localhost:$port -d `"$work\data`" > `"$coreLog`" 2>&1`""
$coreProc = Start-Process -FilePath "cmd.exe" -ArgumentList $coreArgs -WindowStyle Hidden -PassThru

$driverProc = $null
try {
    $started = $false
    for ($i = 0; $i -lt $deadline; $i++) {
        if ((Test-Path $coreLog) -and (Select-String -Path $coreLog -Pattern "started successfully" -Quiet)) {
            $started = $true
            break
        }
        if ($coreProc.HasExited) { break }
        Start-Sleep -Seconds 1
    }
    if (-not $started) {
        Write-Host "ERROR: the Core never started"
        if (Test-Path $coreLog) { Get-Content $coreLog -Tail 40 }
        exit 1
    }

    Write-Host "Logging in..."
    $answers = "localhost`r`n$port`r`nn`r`nsynnax`r`nseldon`r`n"
    Set-Content -Path "$work\login-answers.txt" -Value $answers -Encoding ASCII -NoNewline
    $out = (cmd /c "`"$driver`" login < `"$work\login-answers.txt`" 2>&1" | Out-String)
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: login failed"
        Write-Host $out
        exit 1
    }

    Write-Host "Starting the Driver in standalone mode..."
    $driverLog = "$work\driver.log"
    $driverArgs = "/c `"`"$driver`" start --standalone > `"$driverLog`" 2>&1`""
    $driverProc = Start-Process -FilePath "cmd.exe" -ArgumentList $driverArgs -WindowStyle Hidden -PassThru

    $reached = $false
    for ($i = 0; $i -lt $deadline; $i++) {
        if ((Test-Path $driverLog) -and (Select-String -Path $driverLog -Pattern "successfully reached cluster" -Quiet)) {
            $reached = $true
            break
        }
        if ($driverProc.HasExited) { break }
        Start-Sleep -Seconds 1
    }
    if (-not $reached) {
        Write-Host "ERROR: the standalone Driver never reached the Core"
        if (Test-Path $driverLog) { Get-Content $driverLog -Tail 40 }
        exit 1
    }

    Write-Host "Driver service checks passed"
    exit 0
} finally {
    # The processes are children of cmd.exe wrappers, so kill them by image name.
    cmd /c "taskkill /F /IM synnax-driver.exe 2>nul" | Out-Null
    cmd /c "taskkill /F /IM synnax.exe 2>nul" | Out-Null
    if (-not $coreProc.HasExited) { Stop-Process -Id $coreProc.Id -Force -ErrorAction SilentlyContinue }
    if ($driverProc -and -not $driverProc.HasExited) { Stop-Process -Id $driverProc.Id -Force -ErrorAction SilentlyContinue }
    Remove-Item -Recurse -Force "$env:LOCALAPPDATA\synnax-driver" -ErrorAction SilentlyContinue
}
