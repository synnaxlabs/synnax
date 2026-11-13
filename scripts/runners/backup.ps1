# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Backs up the Synnax Core on Sandbox-Windows to one zip in the Backups folder: the data
# directory, the binary that wrote it, and the config. Run by the Backup - Runner Core
# workflow over SSM. The Core is stopped only while its files are copied.

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
# Stop-Service and Start-Service warn every 2 s while they wait.
$WarningPreference = 'SilentlyContinue'

# The lock of deploy.ps1, so a backup and a deploy never run together.
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

# The binary prints one line, such as "Synnax 0.61.0 (commit: 1a2b3c4, built: ...)".
if ("$(& $Exe version)" -notmatch '^Synnax (\d+\.\d+\.\d+\S*)') {
    throw "Cannot read the version of $Exe"
}
$version = $Matches[1]
$name = "synnax-data-$([DateTime]::UtcNow.ToString('yyyy-MM-dd_HHmm'))-$version"

$staging = Join-Path $env:TEMP $name
Remove-Item (Join-Path $env:TEMP 'synnax-data-*') -Recurse -Force -ErrorAction Ignore
New-Item -ItemType Directory -Force $staging, $Backups | Out-Null

try {
    Stop-Service $Service
    (Get-Service $Service).WaitForStatus('Stopped', $Wait)
    # The service reports Stopped before its process lets go of the data files.
    Get-Process synnax-server -ErrorAction Ignore | Wait-Process -Timeout 120
    robocopy $Data (Join-Path $staging 'data') /E /R:0 /W:0 /NFL /NDL /NJH /NJS /NP |
        Out-Null
    if ($LASTEXITCODE -ge 8) { throw "robocopy failed with exit code $LASTEXITCODE" }
    Copy-Item $Exe (Join-Path $staging "synnax-server-$version.exe")
    Copy-Item $Config $staging
}
finally {
    Start-Service $Service
    (Get-Service $Service).WaitForStatus('Running', $Wait)
}

# Zipped beside the staging folder and then moved, so Backups never holds a partial zip.
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
