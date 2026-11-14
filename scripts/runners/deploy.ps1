# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs the step $Step of a deploy of the runner scripts on Sandbox-Windows.

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

# Held until this process exits, so two steps never run at the same time.
$Lock = [IO.File]::Open(
    (Join-Path $env:TEMP 'runners-deploy.lock'), 'OpenOrCreate', 'ReadWrite', 'None'
)

$Dir = 'C:\Users\Administrator\Desktop\runner_telemetry'
$Uv = 'C:\Program Files\Python313\Scripts\uv.exe'
$Cert = 'C:\ProgramData\Synnax\certs\ca.crt'
$UvCache = 'C:\Windows\system32\config\systemprofile\AppData\Local\uv'
$Repo = 'https://raw.githubusercontent.com/synnaxlabs/synnax'
# uv keys a script's environment on its path, so each deploy reuses this one path.
$Staging = Join-Path $env:TEMP 'runners-staging'
$Tasks = [ordered]@{
    'Runner controller' = @{
        Args = "controller.py --username controller --secure --ca-cert $Cert --scale enabled"
        Log  = 'runner_controller.out.log'
    }
    'Runner dispatch'   = @{
        Args = "dispatch.py --username controller --secure --ca-cert $Cert"
        Log  = 'runner_commands.out.log'
    }
}

# The scripts that must start before a deploy stops anything.
$Checked = 'controller.py', 'dispatch.py', 'agent_deploy.py', 'agent.py'
$AgentDeploy = Join-Path $Staging 'agent_deploy.py'

function Get-Launcher($Name) {
    Join-Path $Dir ('run_' + ($Name -split ' ')[1])
}

switch ($Step) {
    'download' {
        # Defender scans of the files uv writes made a deploy take minutes of both CPUs.
        Add-MpPreference -ExclusionPath $Dir, $UvCache
        Remove-Item $Staging -Recurse -Force -ErrorAction Ignore
        New-Item -ItemType Directory -Force $Staging | Out-Null
        $names = $Files -split ','
        foreach ($f in $names) {
            $url = "$Repo/$Sha/scripts/runners/$f"
            Invoke-WebRequest $url -OutFile (Join-Path $Staging $f) -UseBasicParsing
        }
        # Resolves each script's dependencies and imports, so a bad commit fails here.
        foreach ($script in $Checked) {
            & $Uv run (Join-Path $Staging $script) --help | Out-Null
            if ($LASTEXITCODE -ne 0) { throw "$script --help failed" }
        }
        "Downloaded $($names.Count) scripts at $Sha"
    }
    'stop-agents' {
        & $Uv run $AgentDeploy stop
        if ($LASTEXITCODE -ne 0) { throw 'The agents did not all stop' }
    }
    'stop' {
        foreach ($name in $Tasks.Keys) {
            if (Get-ScheduledTask -TaskName $name -ErrorAction SilentlyContinue) {
                Stop-ScheduledTask -TaskName $name
            }
        }
        # Covers copies started by hand, which no task owns.
        Get-CimInstance Win32_Process |
            Where-Object {
                $_.CommandLine -match '(controller|dispatch)\.py' -and
                $_.ProcessId -ne $PID
            } |
            ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction Ignore }
        "Stopped $($Tasks.Keys -join ' and ')"
    }
    'start' {
        Copy-Item (Join-Path $Staging '*.py') $Dir -Force
        Remove-Item (Join-Path $Dir 'gh_token.txt') -ErrorAction SilentlyContinue
        foreach ($name in $Tasks.Keys) {
            $task = $Tasks[$name]
            $base = Get-Launcher $name
            Remove-Item "$base.cmd", "$base.launches" -ErrorAction Ignore
            # cmd runs the script once, so stdout and stderr share one log.
            Set-Content -Path "$base.once.cmd" -Encoding ascii -Value @(
                '@echo off'
                "cd /d `"$Dir`""
                "`"$Uv`" run $($task.Args) > $($task.Log) 2>&1"
            )
            # Relaunches a script that exits. The wait doubles from 10 s to 5 min.
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
            $settings = New-ScheduledTaskSettingsSet `
                -ExecutionTimeLimit ([TimeSpan]::Zero) -StartWhenAvailable
            Register-ScheduledTask -TaskName $name -Action $action `
                -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $settings `
                -User 'SYSTEM' -RunLevel Highest -Force | Out-Null
            Start-ScheduledTask -TaskName $name
        }
        # A loaded box can take minutes to launch a task, so the 45 s starts after that.
        $deadline = (Get-Date).AddMinutes(5)
        while ((Get-Date) -lt $deadline) {
            $pending = $Tasks.Keys | Where-Object {
                -not (Test-Path "$(Get-Launcher $_).launches")
            }
            if (-not $pending) { break }
            Start-Sleep -Seconds 5
        }
        Start-Sleep -Seconds 45
        $failed = @()
        foreach ($name in $Tasks.Keys) {
            "=== $name ($((Get-ScheduledTask -TaskName $name).State))"
            Get-Content (Join-Path $Dir $Tasks[$name].Log) -Tail 10 -ErrorAction Ignore
            # A script that crashed shows more than one launch line or has no process.
            $launches = "$(Get-Launcher $name).launches"
            $count = @(Get-Content $launches -ErrorAction Ignore).Count
            $script = [regex]::Escape(($Tasks[$name].Args -split ' ')[0])
            $up = Get-CimInstance Win32_Process | Where-Object {
                $_.Name -eq 'uv.exe' -and $_.CommandLine -match $script
            }
            if ($count -ne 1 -or -not $up) { $failed += $name }
        }
        if ($failed) { throw "Exited after start: $($failed -join ', ')" }
    }
    'deploy-agents' {
        & $Uv run $AgentDeploy install --commit $Sha --ca-cert $Cert
        if ($LASTEXITCODE -ne 0) { throw 'The agent install failed on a runner' }
    }
    default { throw "Unknown step '$Step'" }
}
