# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Cleans stale build caches on self-hosted Windows runners to prevent unbounded disk
# growth. Deletes files older than MaxAgeHours (default 2) from Bazel directories,
# Go build/module caches, and old core binaries. Safe to run - Bazel rebuilds from
# S3 remote cache on miss, Go rebuilds on cache miss.

param(
    [int]$MaxAgeHours = 2
)

# Best-effort cleanup — must never fail the build
$ErrorActionPreference = "Continue"

$cutoff = (Get-Date).AddHours(-$MaxAgeHours)
$totalFreed = 0

function Get-DiskUsedMB {
    $d = Get-WmiObject Win32_LogicalDisk -Filter "DriveType=3 AND DeviceID='C:'"
    return [math]::Round(($d.Size - $d.FreeSpace) / 1MB, 0)
}

function Clean-StaleFiles {
    param(
        [string]$Path,
        [string]$Label
    )
    if (Test-Path $Path) {
        $beforeSize = [math]::Round(
            ((Get-ChildItem -Recurse -File $Path -ErrorAction SilentlyContinue |
                Measure-Object -Property Length -Sum).Sum / 1MB), 0)
        Get-ChildItem -Recurse -File $Path -ErrorAction SilentlyContinue |
            Where-Object { $_.LastWriteTime -lt $cutoff } |
            Remove-Item -Force -ErrorAction SilentlyContinue
        Get-ChildItem -Recurse -Directory $Path -ErrorAction SilentlyContinue |
            Where-Object { @(Get-ChildItem $_.FullName -Force -ErrorAction SilentlyContinue).Count -eq 0 } |
            Remove-Item -Force -ErrorAction SilentlyContinue
        $afterSize = [math]::Round(
            ((Get-ChildItem -Recurse -File $Path -ErrorAction SilentlyContinue |
                Measure-Object -Property Length -Sum).Sum / 1MB), 0)
        $freed = $beforeSize - $afterSize
        $script:totalFreed += $freed
        Write-Output ("  {0,-30} {1,6}MB -> {2,6}MB  (freed {3}MB)" -f $Label, $beforeSize, $afterSize, $freed)
    } else {
        Write-Output ("  {0,-30} skipped (not found)" -f $Label)
    }
}

$diskBefore = Get-DiskUsedMB
Write-Output "=== Build Cache Cleanup (max age: ${MaxAgeHours}h) ==="
Write-Output ""

Write-Output "Bazel clean:"
$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$bazelBase = "C:\_bazel"
if ((Test-Path $bazelBase) -and (Test-Path $repoRoot)) {
    $beforeBazel = [math]::Round(
        ((Get-ChildItem -Recurse -File $bazelBase -ErrorAction SilentlyContinue |
            Measure-Object -Property Length -Sum).Sum / 1MB), 0)
    Push-Location $repoRoot
    bazel clean 2>$null
    Pop-Location
    $afterBazel = [math]::Round(
        ((Get-ChildItem -Recurse -File $bazelBase -ErrorAction SilentlyContinue |
            Measure-Object -Property Length -Sum).Sum / 1MB), 0)
    $freedBazel = $beforeBazel - $afterBazel
    $script:totalFreed += $freedBazel
    Write-Output ("  {0,-35} {1,6}MB -> {2,6}MB  (freed {3}MB)" -f "bazel clean", $beforeBazel, $afterBazel, $freedBazel)
} else {
    Write-Output ("  {0,-35} skipped (not found)" -f "bazel clean")
}
Write-Output ""

Write-Output "Go build cache:"
Clean-StaleFiles "C:\Users\Administrator\AppData\Local\go-build" "go-build"
Clean-StaleFiles "C:\Windows\SystemTemp\go-build*" "SystemTemp\go-build"
Write-Output ""

Write-Output "Go module cache:"
Clean-StaleFiles "C:\Users\Administrator\go\pkg\mod\cache" "go\pkg\mod\cache"
Write-Output ""

Write-Output "Old core binaries:"
$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$coreDir = Join-Path $repoRoot "core"
if (Test-Path $coreDir) {
    $oldBinaries = Get-ChildItem -Path $coreDir -Filter "synnax-v*" -ErrorAction SilentlyContinue |
        Where-Object { $_.LastWriteTime -lt $cutoff }
    $count = ($oldBinaries | Measure-Object).Count
    $oldBinaries | Remove-Item -Force -ErrorAction SilentlyContinue
    Write-Output ("  {0,-30} deleted {1} old binaries" -f "core\synnax-v*", $count)
} else {
    Write-Output ("  {0,-30} skipped (not found)" -f "core\")
}
Write-Output ""

$diskAfter = Get-DiskUsedMB
$diskFreed = $diskBefore - $diskAfter
$disk = Get-WmiObject Win32_LogicalDisk -Filter "DriveType=3 AND DeviceID='C:'"

Write-Output "=== Summary ==="
Write-Output "  Cache freed:  ${totalFreed}MB"
Write-Output "  Disk before:  ${diskBefore}MB"
Write-Output "  Disk after:   ${diskAfter}MB"
Write-Output "  Disk freed:   ${diskFreed}MB"
Write-Output ("  Disk total:   {0}GB / Used: {1}GB / Free: {2}GB ({3}%)" -f `
    [math]::Round($disk.Size/1GB, 1),
    [math]::Round(($disk.Size - $disk.FreeSpace)/1GB, 1),
    [math]::Round($disk.FreeSpace/1GB, 1),
    [math]::Round(($disk.Size - $disk.FreeSpace)/$disk.Size * 100, 1))
