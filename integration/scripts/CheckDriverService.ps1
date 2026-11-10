# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms the documented Windows install flow: the Driver installs itself as a service,
# logs in, starts, reaches a Core, stops, and uninstalls. Needs an administrator shell.

$ErrorActionPreference = "Stop"

$core = "$env:USERPROFILE\synnax-binaries\synnax.exe"
$driver = "$env:USERPROFILE\synnax-binaries\synnax-driver.exe"
$installed = "$env:ProgramFiles\Synnax Driver\synnax-driver.exe"
$state = "C:\ProgramData\synnax-driver"
$log = "$state\logs\driver.log"
$service = "SynnaxDriver"
$work = "$env:USERPROFILE\synnax-driver-service-check"
$port = 9094
$deadline = 60

if (Test-Path $work) { Remove-Item -Recurse -Force $work }
New-Item -ItemType Directory -Path $work -Force | Out-Null

function Invoke-Driver([string]$exe, [string]$arguments) {
    $out = (cmd /c "`"$exe`" $arguments 2>&1" | Out-String)
    return @{ Code = $LASTEXITCODE; Out = $out }
}

function Fail([string]$message) {
    Write-Host "ERROR: $message"
    Write-Host "--- Driver log (last 40 lines) ---"
    if (Test-Path $log) { Get-Content $log -Tail 40 }
    Write-Host "--- Core log (last 40 lines) ---"
    if (Test-Path "$work\core.log") { Get-Content "$work\core.log" -Tail 40 }
    exit 1
}

function Get-DriverService {
    return Get-Service -Name $service -ErrorAction SilentlyContinue
}

Write-Host "Starting an insecure Core on localhost:$port..."
$coreLog = "$work\core.log"
$coreArgs = "/c `"`"$core`" start -mi --listen localhost:$port -d `"$work\data`" > `"$coreLog`" 2>&1`""
$coreProc = Start-Process -FilePath "cmd.exe" -ArgumentList $coreArgs -WindowStyle Hidden -PassThru

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
    if (-not $started) { Fail "the Core never started" }

    Write-Host "Installing over a state folder that the Users group owns..."
    New-Item -ItemType Directory -Path $state -Force | Out-Null
    cmd /c "icacls `"$state`" /setowner *S-1-5-32-545 >nul"
    if ($LASTEXITCODE -ne 0) { Fail "could not give $state to the Users group" }
    $r = Invoke-Driver $driver "install"
    if ($r.Code -eq 0) { Fail "install accepted a state folder that the Users group owns" }
    if ($r.Out -notmatch "another user owns") { Write-Host $r.Out; Fail "install failed without the owner error" }
    if ($null -ne (Get-DriverService)) { Fail "install created the service before it checked the state folder" }
    Remove-Item -Recurse -Force $state

    Write-Host "Installing the service..."
    $r = Invoke-Driver $driver "install"
    if ($r.Code -ne 0) { Write-Host $r.Out; Fail "install failed" }
    if ($null -eq (Get-DriverService)) { Fail "install did not create the $service service" }
    if (-not (Test-Path $installed)) { Fail "install did not copy the binary to $installed" }

    Write-Host "Logging in..."
    $answers = "localhost`r`n$port`r`nn`r`nsynnax`r`nseldon`r`n"
    Set-Content -Path "$work\login-answers.txt" -Value $answers -Encoding ASCII -NoNewline
    $r = Invoke-Driver $installed "login < `"$work\login-answers.txt`""
    if ($r.Code -ne 0) { Write-Host $r.Out; Fail "login failed" }

    Write-Host "Starting the service..."
    $r = Invoke-Driver $installed "start"
    if ($r.Code -ne 0) { Write-Host $r.Out; Fail "start failed" }
    $running = $false
    for ($i = 0; $i -lt $deadline; $i++) {
        if ((Get-DriverService).Status -eq "Running") {
            $running = $true
            break
        }
        Start-Sleep -Seconds 1
    }
    if (-not $running) { Fail "the service never started running" }

    Write-Host "Waiting for the Driver to reach the Core..."
    $reached = $false
    for ($i = 0; $i -lt $deadline; $i++) {
        if ((Test-Path $log) -and (Select-String -Path $log -Pattern "successfully reached cluster" -Quiet)) {
            $reached = $true
            break
        }
        Start-Sleep -Seconds 1
    }
    if (-not $reached) { Fail "the Driver never reached the Core" }

    $r = Invoke-Driver $installed "status"
    if ($r.Code -ne 0) { Write-Host $r.Out; Fail "status failed while the service was running" }

    Write-Host "Stopping the service..."
    $r = Invoke-Driver $installed "stop"
    if ($r.Code -ne 0) { Write-Host $r.Out; Fail "stop failed" }
    if ((Get-DriverService).Status -ne "Stopped") { Fail "the service is still running after stop" }

    Write-Host "Uninstalling the service..."
    $r = Invoke-Driver $installed "uninstall"
    if ($r.Code -ne 0) { Write-Host $r.Out; Fail "uninstall failed" }
    if ($null -ne (Get-DriverService)) { Fail "uninstall left the $service service behind" }

    Write-Host "Driver service lifecycle passed"
    exit 0
} finally {
    if ($null -ne (Get-DriverService)) {
        cmd /c "sc.exe stop $service >nul 2>&1" | Out-Null
        cmd /c "sc.exe delete $service >nul 2>&1" | Out-Null
    }
    # The Core is a child of a cmd.exe wrapper, so kill it by image name.
    cmd /c "taskkill /F /IM synnax-driver.exe 2>nul" | Out-Null
    cmd /c "taskkill /F /IM synnax.exe 2>nul" | Out-Null
    if (-not $coreProc.HasExited) { Stop-Process -Id $coreProc.Id -Force -ErrorAction SilentlyContinue }
    Remove-Item -Recurse -Force (Split-Path $installed) -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force $state -ErrorAction SilentlyContinue
}
