# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Installs agent_boot.py at $Commit on a Windows runner and restarts the agent.

param($Commit, $CoreHost, $Ca)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Dir = 'C:\runner-telemetry'
$Task = 'RunnerAgent'
$Uv = 'https://github.com/astral-sh/uv/releases/download/0.12.5'
$Source = "https://raw.githubusercontent.com/synnaxlabs/synnax/$Commit/scripts/runners"

New-Item -ItemType Directory -Force -Path $Dir | Out-Null
if (-not (Test-Path (Join-Path $Dir 'uv.exe'))) {
    $zip = Join-Path $Dir 'uv.zip'
    Invoke-WebRequest "$Uv/uv-x86_64-pc-windows-msvc.zip" -OutFile $zip -UseBasicParsing
    Expand-Archive -Path $zip -DestinationPath $Dir -Force
    Remove-Item $zip
}
# The agent starts the GitHub runner, so the runner must not start at boot.
$runner = Get-ScheduledTask -TaskName 'GitHubActionsRunner' -ErrorAction Ignore
if ($runner -and ($runner.Triggers | Where-Object Enabled)) {
    $runner.Triggers | ForEach-Object { $_.Enabled = $false }
    Set-ScheduledTask -InputObject $runner | Out-Null
}
$boot = Join-Path $Dir 'agent_boot.py'
Invoke-WebRequest "$Source/agent_boot.py" -OutFile $boot -UseBasicParsing
[IO.File]::WriteAllBytes((Join-Path $Dir 'ca.crt'), [Convert]::FromBase64String($Ca))
$log = Join-Path $Dir 'agent.out.log'
if ((Test-Path $log) -and ((Get-Item $log).Length -gt 5MB)) { Clear-Content $log }

Stop-ScheduledTask -TaskName $Task -ErrorAction Ignore
# Does not match a CI job that runs a file such as test_agent.py.
$agent = 'runner-telemetry\\agent\.py|[ "]agent(_boot)?\.py'
Get-CimInstance Win32_Process |
    Where-Object { $_.Name -match 'python|uv|cmd' -and $_.CommandLine -match $agent } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction Ignore }

$argument = "/c set UV_CACHE_DIR=$Dir\uv-cache&& `"$Dir\uv.exe`" run agent_boot.py " +
    "--host $CoreHost --secure >> agent.out.log 2>&1"
$action = New-ScheduledTaskAction -Execute 'cmd.exe' -WorkingDirectory $Dir `
    -Argument $argument
# A task does not restart a program that exits, so a second trigger repeats the start.
$repeat = New-ScheduledTaskTrigger -Once -At (Get-Date) `
    -RepetitionInterval (New-TimeSpan -Minutes 5)
$principal = New-ScheduledTaskPrincipal -UserId 'SYSTEM' -LogonType ServiceAccount `
    -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -MultipleInstances IgnoreNew `
    -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable `
    -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 `
    -RestartInterval (New-TimeSpan -Minutes 1) -Priority 10
Register-ScheduledTask -TaskName $Task -Action $action `
    -Trigger @((New-ScheduledTaskTrigger -AtStartup), $repeat) `
    -Principal $principal -Settings $settings -Force | Out-Null
Start-ScheduledTask -TaskName $Task
Start-Sleep -Seconds 5
if ((Get-ScheduledTask -TaskName $Task).State -ne 'Running') {
    throw "$Task did not stay up"
}
"Installed agent_boot.py at $Commit"
