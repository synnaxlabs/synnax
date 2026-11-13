# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Installs the Synnax Core binary at $Url on Sandbox-Windows and restarts the Core. Run
# by the Deploy - Runner Core workflow over SSM, which sets $Url and $Version before
# this script. $Version is empty for a dev build.

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
# Stop-Service and Start-Service warn every 2 s while they wait.
$WarningPreference = 'SilentlyContinue'

# The lock of deploy.ps1, so a Core deploy never runs with a backup or a deploy.
$Lock = [IO.File]::Open(
    (Join-Path $env:TEMP 'runners-deploy.lock'), 'OpenOrCreate', 'ReadWrite', 'None'
)

$Service = 'SynnaxCore'
$Exe = 'C:\Program Files\Synnax\Core\synnax-server.exe'
$Wait = '00:02:00'

$staging = Join-Path $env:TEMP 'synnax-core'
Remove-Item $staging -Recurse -Force -ErrorAction Ignore
New-Item -ItemType Directory $staging | Out-Null
$new = Join-Path $staging 'synnax-server.exe'
if ($Version) {
    Invoke-WebRequest $Url -OutFile $new -UseBasicParsing
}
else {
    # A dev build is the zip of a workflow artifact, with the binary as its one file.
    $zip = Join-Path $staging 'artifact.zip'
    Invoke-WebRequest $Url -OutFile $zip -UseBasicParsing
    Expand-Archive $zip (Join-Path $staging 'artifact')
    Move-Item (Join-Path $staging 'artifact\*.exe') $new
}
Unblock-File $new

# The Core keeps running until a binary that runs and reports the right version is on
# disk. A dev build has no version to match.
$reported = "$(& $new version)"
if ($reported -notmatch "^Synnax $([regex]::Escape($Version))") {
    throw "The download reports '$reported'"
}

try {
    Stop-Service $Service
    (Get-Service $Service).WaitForStatus('Stopped', $Wait)
    # The service reports Stopped before its process lets go of the binary.
    Get-Process synnax-server -ErrorAction Ignore | Wait-Process -Timeout 120
    # Move-Item does not replace a file that is there.
    Remove-Item $Exe -Force
    Move-Item $new $Exe
}
finally {
    Start-Service $Service
    (Get-Service $Service).WaitForStatus('Running', $Wait)
}
Remove-Item $staging -Recurse -Force

"Deployed $(& $Exe version)"
