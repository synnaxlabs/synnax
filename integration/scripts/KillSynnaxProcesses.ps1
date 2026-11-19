# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

$ErrorActionPreference = "Stop"

Write-Host "Checking for existing synnax processes on Windows..."

# The runner's user cannot end other users' processes, so a failed stop is not an error.
foreach ($name in "synnax", "synnax-driver") {
    $procs = @(Get-Process -Name $name -ErrorAction SilentlyContinue)
    if ($procs.Count -eq 0) {
        Write-Host "No $name.exe processes found"
        continue
    }
    $procs | Stop-Process -Force -ErrorAction SilentlyContinue
    Write-Host "Stopped $name.exe processes"
}

# A force-killed Core never removes its cache directory (UserCacheDir\synnax), so each
# run would leak one.
$dirs = @(
    "$env:USERPROFILE\synnax-binaries",
    "$env:USERPROFILE\synnax-data",
    "$env:LOCALAPPDATA\synnax",
    ".\binaries"
)
foreach ($dir in $dirs) {
    if (Test-Path $dir) {
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $dir
        Write-Host "Removed $dir"
    } else {
        Write-Host "No $dir directory found"
    }
}

$desktop = "$env:USERPROFILE\Desktop\synnax*.exe"
Remove-Item -Force -ErrorAction SilentlyContinue $desktop

Write-Host "Verifying cleanup..."
$failed = $false
foreach ($path in ".\binaries", "$env:USERPROFILE\synnax-binaries",
    "$env:USERPROFILE\synnax-data", $desktop) {
    if (Test-Path $path) {
        Write-Host "ERROR: $path still exists after cleanup"
        $failed = $true
    }
}
if ($failed) {
    Write-Host "Cleanup verification failed"
    exit 1
}

Write-Host "Synnax process cleanup completed"
exit 0
