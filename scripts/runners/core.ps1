# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs the step $Step of a backup or a deploy of the Synnax Core on Sandbox-Windows.

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
# Stop-Service and Start-Service warn every 2 s while they wait.
$WarningPreference = 'SilentlyContinue'

# The lock of deploy.ps1, so a step and a deploy never run together.
$Lock = [IO.File]::Open(
    (Join-Path $env:TEMP 'runners-deploy.lock'), 'OpenOrCreate', 'ReadWrite', 'None'
)

$Service = 'SynnaxCore'
$Exe = 'C:\Program Files\Synnax\Core\synnax-server.exe'
$Data = 'C:\ProgramData\Synnax\data'
$Config = 'C:\ProgramData\Synnax\config.yaml'
$Backups = 'C:\Users\Administrator\Desktop\Backups'
$MinFree = 25GB
$Wait = '00:02:00'
$Temp = Join-Path $env:TEMP 'synnax-core'
$New = Join-Path $Temp 'synnax-server.exe'

switch ($Step) {
    'download' {
        Remove-Item $Temp -Recurse -Force -ErrorAction Ignore
        New-Item -ItemType Directory $Temp | Out-Null
        if ($Version) {
            Invoke-WebRequest $Url -OutFile $New -UseBasicParsing
        }
        else {
            # A dev build is the zip of a workflow artifact with one binary in it.
            $artifact = Join-Path $Temp 'artifact'
            Invoke-WebRequest $Url -OutFile "$artifact.zip" -UseBasicParsing
            Expand-Archive "$artifact.zip" $artifact
            Move-Item (Join-Path $artifact '*.exe') $New
        }
        Unblock-File $New
        $reported = "$(& $New version)"
        if ($reported -notmatch "^Synnax $([regex]::Escape($Version))") {
            throw "The download reports '$reported'"
        }
        "Downloaded $reported"
    }
    'stop' {
        Stop-Service $Service
        (Get-Service $Service).WaitForStatus('Stopped', $Wait)
        # The service reports Stopped before its process lets go of its files.
        Get-Process synnax-server -ErrorAction Ignore | Wait-Process -Timeout 120
        "Stopped $Service"
    }
    'backup' {
        # The binary prints one line that starts with "Synnax 0.61.0".
        if ("$(& $Exe version)" -notmatch '^Synnax (\d+\.\d+\.\d+\S*)') {
            throw "Cannot read the version of $Exe"
        }
        $running = $Matches[1]
        $stamp = [DateTime]::UtcNow.ToString('yyyy-MM-dd_HHmm')
        $name = "synnax-data-$stamp-$running"
        $staging = Join-Path $Temp $name
        Remove-Item "$Temp\synnax-data-*" -Recurse -Force -ErrorAction Ignore
        New-Item -ItemType Directory -Force $staging, $Backups | Out-Null
        robocopy $Data "$staging\data" /E /R:0 /W:0 /NFL /NDL /NJH /NJS /NP | Out-Null
        if ($LASTEXITCODE -ge 8) { throw "robocopy failed with code $LASTEXITCODE" }
        Copy-Item $Exe (Join-Path $staging "synnax-server-$running.exe")
        Copy-Item $Config $staging
        # Zipped beside the staging folder, so Backups never holds a partial zip.
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [IO.Compression.ZipFile]::CreateFromDirectory(
            $staging, "$staging.zip", [IO.Compression.CompressionLevel]::Optimal, $true
        )
        Move-Item "$staging.zip" $Backups
        Remove-Item $staging -Recurse -Force
        $zip = Get-Item (Join-Path $Backups "$name.zip")
        "Backup $($zip.Name), $([math]::Round($zip.Length / 1MB)) MB"
        $free = (Get-PSDrive C).Free
        if ($free -lt $MinFree) {
            throw "C: has $([math]::Round($free / 1GB)) GB free, less than 25 GB"
        }
    }
    'install' {
        # Windows lets a binary that just ran be renamed, not removed.
        Move-Item $Exe (Join-Path $Temp 'synnax-server.old.exe')
        Move-Item $New $Exe
        "Installed $(& $Exe version)"
    }
    'start' {
        Start-Service $Service
        (Get-Service $Service).WaitForStatus('Running', $Wait)
        if (Test-Path $Temp) { Remove-Item $Temp -Recurse -Force }
        "Running $(& $Exe version)"
    }
    default { throw "Unknown step '$Step'" }
}
