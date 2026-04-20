# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Cleans build caches on self-hosted Windows runners to prevent unbounded disk growth.
# - Bazel: always runs `bazel clean` (unconditional - remote cache serves next build)
# - Go/binaries: deletes oldest files first until MinFreeGB of disk space is available
#
# Usage: CleanBuildCaches.ps1 [-MinFreeGB 25]

param(
    [int]$MinFreeGB = 25
)

# Best-effort cleanup — must never fail the build
$ErrorActionPreference = "Continue"

$drive = [System.IO.DriveInfo]::new("C")
$minFreeBytes = [int64]$MinFreeGB * 1GB
$totalFreed = 0

function Get-DiskFreeMB {
    return [math]::Round($drive.AvailableFreeSpace / 1MB, 0)
}

function Get-DiskUsedMB {
    return [math]::Round(($drive.TotalSize - $drive.AvailableFreeSpace) / 1MB, 0)
}

function Test-EnoughSpace {
    return $drive.AvailableFreeSpace -ge $minFreeBytes
}

function Write-DiskSummary {
    $total = [math]::Round($drive.TotalSize / 1GB, 1)
    $free = [math]::Round($drive.AvailableFreeSpace / 1GB, 1)
    $used = [math]::Round(($drive.TotalSize - $drive.AvailableFreeSpace) / 1GB, 1)
    $pct = [math]::Round(
        ($drive.TotalSize - $drive.AvailableFreeSpace) / $drive.TotalSize * 100, 1)
    Write-Output ("  Disk total:   {0}GB / Used: {1}GB / Free: {2}GB ({3}%)" -f `
        $total, $used, $free, $pct)
}

$diskBefore = Get-DiskUsedMB
$freeMB = Get-DiskFreeMB
$minFreeMB = $MinFreeGB * 1024
Write-Output "=== Build Cache Cleanup (target: ${MinFreeGB}GB free) ==="
Write-Output "  Current free space: ${freeMB}MB (target: ${minFreeMB}MB)"
Write-Output ""

# --- Bazel clean (unconditional) ---
Write-Output "Bazel clean:"
$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$bazelBase = "C:\_bazel"
if ((Test-Path $bazelBase) -and (Test-Path $repoRoot)) {
    $beforeBazel = [math]::Round(
        ((Get-ChildItem -Recurse -File $bazelBase -ErrorAction SilentlyContinue |
            Measure-Object -Property Length -Sum).Sum / 1MB), 0)
    Push-Location $repoRoot
    $bazelOutput = bazel clean 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Output "  bazel clean failed (exit $LASTEXITCODE): $bazelOutput"
    }
    Pop-Location
    $afterBazel = [math]::Round(
        ((Get-ChildItem -Recurse -File $bazelBase -ErrorAction SilentlyContinue |
            Measure-Object -Property Length -Sum).Sum / 1MB), 0)
    $freedBazel = $beforeBazel - $afterBazel
    $script:totalFreed += $freedBazel
    Write-Output ("  {0,-35} {1,6}MB -> {2,6}MB  (freed {3}MB)" -f `
        "bazel clean", $beforeBazel, $afterBazel, $freedBazel)
} else {
    Write-Output ("  {0,-35} skipped (not found)" -f "bazel clean")
}
Write-Output ""

# --- Check if we already have enough space after bazel clean ---
if (Test-EnoughSpace) {
    $freeMB = Get-DiskFreeMB
    Write-Output "Free space ${freeMB}MB >= target ${minFreeMB}MB - skipping cache cleanup."
    Write-Output ""
    $diskAfter = Get-DiskUsedMB
    $diskFreed = $diskBefore - $diskAfter
    Write-Output "=== Summary ==="
    Write-Output "  Cache freed:  ${totalFreed}MB"
    Write-Output "  Disk before:  ${diskBefore}MB"
    Write-Output "  Disk after:   ${diskAfter}MB"
    Write-Output "  Disk freed:   ${diskFreed}MB"
    Write-DiskSummary
    return
}

# --- Collect all cache files sorted oldest-first, delete until target met ---
Write-Output "Deleting oldest cache files until ${MinFreeGB}GB free..."

$cacheDirs = @(
    "C:\Users\Administrator\AppData\Local\go-build",
    "C:\Users\Administrator\go\pkg\mod\cache",
    "C:\Windows\SystemTemp\go-build"
)
$coreDir = Join-Path $repoRoot "core"

$allFiles = @()
foreach ($dir in $cacheDirs) {
    if (Test-Path $dir) {
        $allFiles += Get-ChildItem -Recurse -File $dir -ErrorAction SilentlyContinue
    }
}
if (Test-Path $coreDir) {
    $allFiles += Get-ChildItem -Path $coreDir -Filter "synnax-v*" `
        -ErrorAction SilentlyContinue |
        Where-Object { -not $_.PSIsContainer }
}

$allFiles = $allFiles | Sort-Object LastWriteTime

$deleted = 0
foreach ($file in $allFiles) {
    $fileSize = $file.Length
    Remove-Item $file.FullName -Force -ErrorAction SilentlyContinue
    $deleted++
    $script:totalFreed += [math]::Round($fileSize / 1MB, 0)
    if (Test-EnoughSpace) { break }
}

Write-Output "  Deleted $deleted files"

foreach ($dir in $cacheDirs) {
    if (Test-Path $dir) {
        Get-ChildItem -Recurse -Directory $dir -ErrorAction SilentlyContinue |
            Where-Object {
                @(Get-ChildItem $_.FullName -Force `
                    -ErrorAction SilentlyContinue).Count -eq 0
            } |
            Remove-Item -Force -ErrorAction SilentlyContinue
    }
}
Write-Output ""

if (-not (Test-EnoughSpace)) {
    $freeMB = Get-DiskFreeMB
    Write-Output ("WARNING: Caches exhausted but free space {0}MB < target {1}MB" -f `
        $freeMB, $minFreeMB)
    Write-Output ""
}

$diskAfter = Get-DiskUsedMB
$diskFreed = $diskBefore - $diskAfter

Write-Output "=== Summary ==="
Write-Output "  Cache freed:  ~${totalFreed}MB"
Write-Output "  Disk before:  ${diskBefore}MB"
Write-Output "  Disk after:   ${diskAfter}MB"
Write-Output "  Disk freed:   ${diskFreed}MB"
Write-DiskSummary
