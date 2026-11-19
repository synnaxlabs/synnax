# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Downloads the Core and Driver of run REF_RUN_ID into %USERPROFILE%\synnax-binaries.

$ErrorActionPreference = "Stop"

$binaries = "$env:USERPROFILE\synnax-binaries"
$runKey = "$env:REF_RUN_ID-$env:GITHUB_RUN_ATTEMPT"

# Self-hosted runners keep the profile across jobs: skip when this attempt's binaries
# are already in place (a re-run keeps the run id but can rebuild them).
if ($env:REF_RUN_ID -and (Test-Path "$binaries\.run-id") -and
    (Test-Path "$binaries\synnax.exe") -and
    (Test-Path "$binaries\synnax-driver.exe") -and
    ((Get-Content "$binaries\.run-id" -TotalCount 1) -eq $runKey)) {
    Write-Host "Binaries for run $env:REF_RUN_ID already present, skipping download"
    exit 0
}
if (-not $env:REF_RUN_ID) {
    Write-Host "Error: REF_RUN_ID not provided"
    exit 1
}

function Find-Gh {
    if (Get-Command gh -ErrorAction SilentlyContinue) { return "gh" }
    foreach ($path in "$env:ProgramFiles\GitHub CLI\gh.exe",
        "C:\ProgramData\chocolatey\lib\gh\tools\gh.exe") {
        if (Test-Path $path) { return $path }
    }
}

$gh = Find-Gh
if (-not $gh) {
    Write-Host "GitHub CLI not found, installing it with Chocolatey..."
    # Chocolatey can exit non-zero after a successful install, so the search decides.
    choco install gh -y --force
    $env:PATH = [Environment]::GetEnvironmentVariable("PATH", "Machine") + ";" +
        [Environment]::GetEnvironmentVariable("PATH", "User")
    $gh = Find-Gh
    if (-not $gh) {
        Write-Host "Error: GitHub CLI not found after installation"
        exit 1
    }
}
& $gh --version
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: GitHub CLI test failed"
    exit 1
}
& $gh auth status
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: GitHub CLI authentication failed"
    exit 1
}

# download_run_artifacts.sh reads gh_cmd and GH_REPO.
$env:gh_cmd = $gh
$env:GH_REPO = "synnaxlabs/synnax"
$bash = "$env:ProgramFiles\Git\bin\bash.exe"

function Get-Artifact([string]$Pattern, [string]$Directory) {
    Write-Host "Downloading $Pattern artifact from run $env:REF_RUN_ID..."
    & $bash .github/scripts/download_run_artifacts.sh $env:REF_RUN_ID $Pattern `
        $Directory
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error: Failed to download $Pattern"
        exit 1
    }
}

Get-Artifact "synnax-core*windows" binaries
# gh run download creates a subdirectory named after the artifact.
$core = @(Get-Item .\binaries\synnax-core*\synnax-v*.exe -ErrorAction SilentlyContinue)
if ($core.Count -eq 0) {
    Write-Host "Error: No synnax executable found in the binaries directory"
    Get-ChildItem -Recurse .\binaries -ErrorAction SilentlyContinue
    exit 1
}

# A test that restarts its Core runs this Driver apart from the Core.
Get-Artifact "synnax-driver-windows" driver-binaries
$driver = @(Get-Item .\driver-binaries\synnax-driver-windows\synnax-driver-v*.exe `
    -ErrorAction SilentlyContinue)
if ($driver.Count -eq 0) {
    Write-Host "Error: No synnax-driver executable found in driver-binaries directory"
    Get-ChildItem -Recurse .\driver-binaries -ErrorAction SilentlyContinue
    exit 1
}

New-Item -ItemType Directory -Force $binaries | Out-Null
Copy-Item -Force $core[0].FullName "$binaries\synnax.exe"
Copy-Item -Force $driver[0].FullName "$binaries\synnax-driver.exe"
Get-ChildItem "$binaries\synnax*"
Set-Content -Encoding ASCII "$binaries\.run-id" $runKey

Write-Host "Windows artifacts setup completed successfully"
exit 0
