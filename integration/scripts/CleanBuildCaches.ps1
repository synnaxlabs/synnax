# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Cleans build caches on self-hosted Windows runners to prevent unbounded disk growth.
# Runs only under disk pressure so the Bazel output tree stays warm and builds stay
# incremental:
# - If free space >= MinFreeGB, nothing is cleaned.
# - Otherwise: `bazel clean` first (largest consumer), then oldest Go/binary files
#   until MinFreeGB is available.
#
# Usage: CleanBuildCaches.ps1 [-MinFreeGB 30]

param(
    [int]$MinFreeGB = 30
)

# Best-effort cleanup — must never fail the build
$ErrorActionPreference = "Continue"

$drive = [System.IO.DriveInfo]::new("C")
$minFreeBytes = [int64]$MinFreeGB * 1GB
$totalFreed = 0

function Get-DiskFreeGB {
    return [math]::Round($drive.AvailableFreeSpace / 1GB, 1)
}

function Get-DiskUsedGB {
    return [math]::Round(($drive.TotalSize - $drive.AvailableFreeSpace) / 1GB, 1)
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
    Write-Output ("  Disk total:   {0} GB / Used: {1} GB / Free: {2} GB ({3}%)" -f `
        $total, $used, $free, $pct)
}

$diskBefore = Get-DiskUsedGB
$freeGB = Get-DiskFreeGB
Write-Output "=== Build Cache Cleanup (target: ${MinFreeGB} GB free) ==="
Write-Output "  Current free space: ${freeGB} GB (target: ${MinFreeGB} GB)"
Write-Output ""

# --- Above threshold: Keep Bazel cache warm ---
if (Test-EnoughSpace) {
    Write-Output "Free space ${freeGB} GB >= target ${MinFreeGB} GB - keeping caches warm (no clean)."
    Write-Output ""
    $diskAfter = Get-DiskUsedGB
    $diskFreed = $diskBefore - $diskAfter
    Write-Output "=== Summary ==="
    Write-Output "  Cache freed:  0 GB"
    Write-Output "  Disk before:  ${diskBefore} GB"
    Write-Output "  Disk after:   ${diskAfter} GB"
    Write-Output "  Disk freed:   ${diskFreed} GB"
    Write-DiskSummary
    return
}

# --- Below threshold: Bazel is the largest consumer, so clean it first ---
# The runner bazelrc pins --output_base=C:/_bazel, so `bazel clean` frees it.
Write-Output "Bazel clean:"
$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
if (Test-Path $repoRoot) {
    Push-Location $repoRoot
    $outputBase = (bazel info output_base 2>$null)
    $before = 0
    if ($outputBase -and (Test-Path $outputBase)) {
        $before = [math]::Round(
            ((Get-ChildItem -Recurse -File $outputBase -ErrorAction SilentlyContinue |
                Measure-Object -Property Length -Sum).Sum / 1MB), 0)
    }
    $out = bazel clean 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Output "  clean failed (exit $LASTEXITCODE): $out"
    }
    Pop-Location
    $after = 0
    if ($outputBase -and (Test-Path $outputBase)) {
        $after = [math]::Round(
            ((Get-ChildItem -Recurse -File $outputBase -ErrorAction SilentlyContinue |
                Measure-Object -Property Length -Sum).Sum / 1MB), 0)
    }
    $label = if ($outputBase) { $outputBase } else { "output_base" }
    $script:totalFreed += ($before - $after)
    Write-Output ("  {0,-35} {1,6}MB -> {2,6}MB  (freed {3}MB)" -f `
        $label, $before, $after, ($before - $after))
} else {
    Write-Output ("  {0,-35} skipped (repo not found)" -f "bazel clean")
}
Write-Output ""

# --- C:/tmp: Bazel install/server files that `bazel clean` leaves behind ---
if (Test-Path "C:/tmp") {
    $tmpBefore = [math]::Round(
        ((Get-ChildItem -Recurse -File "C:/tmp" -ErrorAction SilentlyContinue |
            Measure-Object -Property Length -Sum).Sum / 1MB), 0)
    Remove-Item -Recurse -Force "C:/tmp" -ErrorAction SilentlyContinue
    $script:totalFreed += $tmpBefore
    Write-Output ("  {0,-35} freed {1}MB" -f "C:/tmp", $tmpBefore)
}
Write-Output ""

# --- Docker (unbounded image/layer growth, largest consumer on some bots) ---
Write-Output "Docker prune:"
if (Get-Command docker -ErrorAction SilentlyContinue) {
    docker system prune -af 2>&1 | Out-Null
    Write-Output "  done"
} else {
    Write-Output "  skipped (docker not found)"
}
Write-Output ""

# --- Check if we already have enough space after bazel clean ---
if (Test-EnoughSpace) {
    $freeGB = Get-DiskFreeGB
    Write-Output "Free space ${freeGB} GB >= target ${MinFreeGB} GB - skipping cache cleanup."
    Write-Output ""
    $diskAfter = Get-DiskUsedGB
    $diskFreed = $diskBefore - $diskAfter
    Write-Output "=== Summary ==="
    Write-Output ("  Cache freed:  {0} GB" -f [math]::Round($totalFreed / 1024, 1))
    Write-Output "  Disk before:  ${diskBefore} GB"
    Write-Output "  Disk after:   ${diskAfter} GB"
    Write-Output "  Disk freed:   ${diskFreed} GB"
    Write-DiskSummary
    return
}

# --- Collect all cache files sorted oldest-first, delete until target met ---
Write-Output "Deleting oldest cache files until ${MinFreeGB} GB free..."

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
    $freeGB = Get-DiskFreeGB
    Write-Output ("WARNING: Caches exhausted but free space {0} GB < target {1} GB" -f `
        $freeGB, $MinFreeGB)
    Write-Output ""
}

$diskAfter = Get-DiskUsedGB
$diskFreed = $diskBefore - $diskAfter

Write-Output "=== Summary ==="
Write-Output ("  Cache freed:  ~{0} GB" -f [math]::Round($totalFreed / 1024, 1))
Write-Output "  Disk before:  ${diskBefore} GB"
Write-Output "  Disk after:   ${diskAfter} GB"
Write-Output "  Disk freed:   ${diskFreed} GB"
Write-DiskSummary
