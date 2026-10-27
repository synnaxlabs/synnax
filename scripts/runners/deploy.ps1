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
$UvCache = 'C:\Windows\system32\config\systemprofile\AppData\Local\uv'
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

# Defender scanning every file uv writes made a deploy take both CPUs for minutes.
Add-MpPreference -ExclusionPath $Dir, $UvCache

# Downloads everything before touching the running scripts. uv keys a script's
# environment on its path, so one staging path lets each deploy reuse the last one's.
$Staging = Join-Path $env:TEMP 'runners-staging'
Remove-Item $Staging -Recurse -Force -ErrorAction Ignore
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

foreach ($name in $Tasks.Keys) {
    $task = $Tasks[$name]
    $base = Join-Path $Dir ('run_' + ($name -split ' ')[1])
    Remove-Item "$base.cmd", "$base.launches" -ErrorAction Ignore
    # cmd runs the script once, so stdout and stderr share one log.
    Set-Content -Path "$base.once.cmd" -Encoding ascii -Value @(
        '@echo off'
        "cd /d `"$Dir`""
        "`"$Uv`" run $($task.Args) > $($task.Log) 2>&1"
    )
    # Relaunches a script that exits, waiting 10 s and doubling to 5 min while it keeps
    # crashing. A run that outlasts the longest wait resets the backoff.
    Set-Content -Path "$base.ps1" -Encoding ascii -Value @(
        '$delay = 10'
        'while ($true) {'
        "    Add-Content '$base.launches' 'launch'"
        '    $start = Get-Date'
        "    cmd.exe /c '$base.once.cmd'"
        '    if (((Get-Date) - $start).TotalSeconds -gt 300) { $delay = 10 }'
        '    Start-Sleep -Seconds $delay'
        '    $delay = [Math]::Min($delay * 2, 300)'
        '}'
    )
    $action = New-ScheduledTaskAction -Execute 'powershell.exe' `
        -Argument "-NoProfile -ExecutionPolicy Bypass -File `"$base.ps1`"" `
        -WorkingDirectory $Dir
    $settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) `
        -StartWhenAvailable
    Register-ScheduledTask -TaskName $name -Action $action `
        -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $settings `
        -User 'SYSTEM' -RunLevel Highest -Force | Out-Null
    Start-ScheduledTask -TaskName $name
}

# A loaded box can take minutes to run a task's first launch, so the 45 s that
# shows a crash loop counts from the last first launch.
$deadline = (Get-Date).AddMinutes(5)
while ((Get-Date) -lt $deadline) {
    $pending = $Tasks.Keys | Where-Object {
        -not (Test-Path (Join-Path $Dir ('run_' + ($_ -split ' ')[1] + '.launches')))
    }
    if (-not $pending) { break }
    Start-Sleep -Seconds 5
}
Start-Sleep -Seconds 45
$failed = @()
foreach ($name in $Tasks.Keys) {
    "=== $name ($((Get-ScheduledTask -TaskName $name).State))"
    Get-Content (Join-Path $Dir $Tasks[$name].Log) -Tail 10 -ErrorAction Ignore
    # The launcher logs a line per launch, so a script that exited and relaunched
    # shows more than one, and one waiting to relaunch has no process.
    $launches = Join-Path $Dir ('run_' + ($name -split ' ')[1] + '.launches')
    $count = @(Get-Content $launches -ErrorAction Ignore).Count
    $script = [regex]::Escape(($Tasks[$name].Args -split ' ')[0])
    $up = Get-CimInstance Win32_Process | Where-Object {
        $_.Name -eq 'uv.exe' -and $_.CommandLine -match $script
    }
    if ($count -ne 1 -or -not $up) { $failed += $name }
}
if ($failed) { throw "Exited after start: $($failed -join ', ')" }
