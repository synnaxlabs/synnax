# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Installs agent_boot.py at $Commit on a Windows runner and restarts the agent. Also
# installs start-runner.ps1 and the GitHubActionsRunner task for the runner name prefix
# $Prefix and the $Labels of the runner's pool, in the first of the $Runners folders
# that holds a GitHub runner, registered or not.

param($Commit, $CoreHost, $Ca, $Prefix, $Labels, [string[]]$Runners)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Dir = 'C:\runner-telemetry'
$Task = 'RunnerAgent'
$Uv = 'https://github.com/astral-sh/uv/releases/download/0.12.5'
$Source = "https://raw.githubusercontent.com/synnaxlabs/synnax/$Commit/scripts/runners"

$Runner = $Runners | Where-Object { Test-Path (Join-Path $_ 'config.cmd') } |
    Select-Object -First 1
if (-not $Runner) { throw "No GitHub runner in $($Runners -join ', ')" }
if (-not [Environment]::GetEnvironmentVariable('GITHUB_PAT', 'Machine')) {
    throw 'GITHUB_PAT is not set'
}
New-Item -ItemType Directory -Force -Path $Dir | Out-Null
if (-not (Test-Path (Join-Path $Dir 'uv.exe'))) {
    $zip = Join-Path $Dir 'uv.zip'
    Invoke-WebRequest "$Uv/uv-x86_64-pc-windows-msvc.zip" -OutFile $zip -UseBasicParsing
    Expand-Archive -Path $zip -DestinationPath $Dir -Force
    Remove-Item $zip
}
# Every download ends before the first write, so a failure keeps the old install. Not
# in the temp folder, which the CI cache cleanup empties.
$staging = Join-Path $Dir 'install-staging'
Remove-Item $staging -Recurse -Force -ErrorAction Ignore
New-Item -ItemType Directory -Force -Path $staging | Out-Null
foreach ($file in 'agent_boot.py', 'pyproject.toml', 'uv.lock', 'start-runner.ps1') {
    $out = Join-Path $staging $file
    Invoke-WebRequest "$Source/$file" -OutFile $out -UseBasicParsing
}
# A running runner already read its script, so the copy does not disturb it.
Move-Item (Join-Path $staging 'start-runner.ps1') $Runner -Force
# The agent starts the GitHub runner, so the task has no trigger.
$start = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument (
    "-ExecutionPolicy Bypass -NoProfile -File `"$Runner\start-runner.ps1`" " +
    "-Prefix $Prefix -Labels $Labels")
$runnerPrincipal = New-ScheduledTaskPrincipal -UserId 'SYSTEM' `
    -LogonType ServiceAccount -RunLevel Highest
$runnerSettings = New-ScheduledTaskSettingsSet -MultipleInstances IgnoreNew `
    -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable `
    -ExecutionTimeLimit (New-TimeSpan -Hours 72) -RestartCount 3 `
    -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName 'GitHubActionsRunner' -Action $start `
    -Principal $runnerPrincipal -Settings $runnerSettings -Force | Out-Null
[IO.File]::WriteAllBytes((Join-Path $Dir 'ca.crt'), [Convert]::FromBase64String($Ca))
$log = Join-Path $Dir 'agent.out.log'
if ((Test-Path $log) -and ((Get-Item $log).Length -gt 5MB)) { Clear-Content $log }

Stop-ScheduledTask -TaskName $Task -ErrorAction Ignore
# Does not match a CI job that runs a file such as test_agent.py.
$agent = 'runner-telemetry\\agent\.py|[ "]agent(_boot)?\.py'
Get-CimInstance Win32_Process |
    Where-Object { $_.Name -match 'python|uv|cmd' -and $_.CommandLine -match $agent } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction Ignore }
Move-Item (Join-Path $staging '*') $Dir -Force
Remove-Item $staging -Recurse -Force
# agent_boot.py then downloads every agent file at the deployed commit.
Remove-Item (Join-Path $Dir 'agent_commit.txt') -ErrorAction Ignore

$argument = "/c set UV_CACHE_DIR=$Dir\uv-cache&& `"$Dir\uv.exe`" run --project . " +
    "--frozen --no-dev agent_boot.py --host $CoreHost --secure >> agent.out.log 2>&1"
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
