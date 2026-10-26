# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Installs the runner scripts at commit $Sha on Sandbox-Windows and restarts
# controller.py and dispatch.py as startup tasks that relaunch them when they exit. Run
# by the Deploy - Runners workflow over SSM, which sets $Sha and $Files before this
# script. Secrets come from Parameter Store, so none appear on a command line.

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$Dir = 'C:\Users\Administrator\Desktop\runner_telemetry'
$Uv = 'C:\Program Files\Python313\Scripts\uv.exe'
$Base = "https://raw.githubusercontent.com/synnaxlabs/synnax/$Sha/scripts/runners"
$Tasks = [ordered]@{
    'Runner controller' = @{
        Args = 'controller.py --username controller --secure --scale enabled'
        Log  = 'runner_controller.out.log'
    }
    'Runner dispatch'   = @{
        Args = 'dispatch.py --username controller --secure'
        Log  = 'runner_commands.out.log'
    }
}

# Downloads everything before touching the running scripts.
$Staging = Join-Path $env:TEMP "runners-$Sha"
New-Item -ItemType Directory -Force $Staging | Out-Null
foreach ($f in $Files) {
    Invoke-WebRequest "$Base/$f" -OutFile (Join-Path $Staging $f) -UseBasicParsing
}
# Resolves each script's PEP 723 dependencies and imports, so a broken commit fails
# here while the old scripts still run.
foreach ($name in $Tasks.Keys) {
    $script = ($Tasks[$name].Args -split ' ')[0]
    & $Uv run (Join-Path $Staging $script) --help | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "$script --help failed" }
}

foreach ($name in $Tasks.Keys) {
    if (Get-ScheduledTask -TaskName $name -ErrorAction SilentlyContinue) {
        Stop-ScheduledTask -TaskName $name
    }
}
# Covers copies started by hand, which no task owns.
Get-CimInstance Win32_Process |
    Where-Object {
        $_.CommandLine -match '(controller|dispatch)\.py' -and $_.ProcessId -ne $PID
    } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction Ignore }

Copy-Item (Join-Path $Staging '*.py') $Dir -Force
Remove-Item (Join-Path $Dir 'gh_token.txt') -ErrorAction SilentlyContinue
Remove-Item $Staging -Recurse -Force

$Started = Get-Date

foreach ($name in $Tasks.Keys) {
    $task = $Tasks[$name]
    $launcher = Join-Path $Dir ('run_' + ($name -split ' ')[1] + '.cmd')
    # ping is the sleep: timeout.exe fails without a console, as under SYSTEM.
    Set-Content -Path $launcher -Encoding ascii -Value @(
        '@echo off'
        'cd /d "%~dp0"'
        ':loop'
        "`"$Uv`" run $($task.Args) > $($task.Log) 2>&1"
        'ping -n 11 127.0.0.1 > nul'
        'goto loop'
    )
    $action = New-ScheduledTaskAction -Execute 'cmd.exe' -Argument "/c `"$launcher`"" `
        -WorkingDirectory $Dir
    $settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) `
        -StartWhenAvailable
    Register-ScheduledTask -TaskName $name -Action $action `
        -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $settings `
        -User 'SYSTEM' -RunLevel Highest -Force | Out-Null
    Start-ScheduledTask -TaskName $name
}

Start-Sleep -Seconds 45
$failed = @()
foreach ($name in $Tasks.Keys) {
    "=== $name ($((Get-ScheduledTask -TaskName $name).State))"
    Get-Content (Join-Path $Dir $Tasks[$name].Log) -Tail 10 -ErrorAction Ignore
    # The launcher waits 10 s before it relaunches a script that exits, so only the
    # first launch starts within 5 s of $Started.
    $script = [regex]::Escape(($Tasks[$name].Args -split ' ')[0])
    $up = Get-CimInstance Win32_Process | Where-Object {
        $_.Name -eq 'uv.exe' -and $_.CommandLine -match $script -and
        $_.CreationDate -lt $Started.AddSeconds(5)
    }
    if (-not $up) { $failed += $name }
}
if ($failed) { throw "Exited after start: $($failed -join ', ')" }
