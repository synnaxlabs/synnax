# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Cleans stale build caches on self-hosted Windows runners to prevent unbounded disk
# growth. Deletes files older than MaxAgeHours (default 6) from Bazel directories,
# Go build/module caches, and old core binaries. Safe to run - Bazel rebuilds from
# S3 remote cache on miss, Go rebuilds on cache miss.

param(
    [int]$MaxAgeHours = 2
)

$cutoff = (Get-Date).AddHours(-$MaxAgeHours)
Write-Output "Cleaning build caches older than ${MaxAgeHours}h..."

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
        # Remove empty directories
        Get-ChildItem -Recurse -Directory $Path -ErrorAction SilentlyContinue |
            Where-Object { @(Get-ChildItem $_.FullName -Force).Count -eq 0 } |
            Remove-Item -Force -ErrorAction SilentlyContinue
        $afterSize = [math]::Round(
            ((Get-ChildItem -Recurse -File $Path -ErrorAction SilentlyContinue |
                Measure-Object -Property Length -Sum).Sum / 1MB), 0)
        $freed = $beforeSize - $afterSize
        Write-Output "  ${Label}: ${beforeSize}MB -> ${afterSize}MB (freed ${freed}MB)"
    } else {
        Write-Output "  ${Label}: not found, skipping"
    }
}

Write-Output "--- Bazel caches ---"
Clean-StaleFiles "C:\_bazel" "C:\_bazel"
Clean-StaleFiles "C:\_bazel-disk" "C:\_bazel-disk"
Clean-StaleFiles "C:\_bazel-repo" "C:\_bazel-repo"
Clean-StaleFiles "C:\Users\Administrator\_bazel_Administrator" "_bazel_Administrator"

Write-Output "--- Go build cache ---"
Clean-StaleFiles "C:\Users\Administrator\AppData\Local\go-build" "go-build"
Clean-StaleFiles "C:\Windows\SystemTemp\go-build*" "SystemTemp\go-build"

Write-Output "--- Go module cache ---"
Clean-StaleFiles "C:\Users\Administrator\go\pkg\mod\cache" "go\pkg\mod\cache"

Write-Output "--- Old core binaries ---"
$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$coreDir = Join-Path $repoRoot "core"
if (Test-Path $coreDir) {
    $oldBinaries = Get-ChildItem -Path $coreDir -Filter "synnax-v*" -ErrorAction SilentlyContinue |
        Where-Object { $_.LastWriteTime -lt $cutoff }
    $count = ($oldBinaries | Measure-Object).Count
    $oldBinaries | Remove-Item -Force -ErrorAction SilentlyContinue
    Write-Output "  core\synnax-v*: deleted $count old binaries"
} else {
    Write-Output "  core\: not found, skipping"
}

Write-Output "--- Disk usage ---"
$disk = Get-WmiObject Win32_LogicalDisk -Filter "DriveType=3"
$usage = "SizeGB={0} UsedGB={1} FreeGB={2} Used={3}%" -f `
    [math]::Round($disk.Size/1GB, 1),
    [math]::Round(($disk.Size - $disk.FreeSpace)/1GB, 1),
    [math]::Round($disk.FreeSpace/1GB, 1),
    [math]::Round(($disk.Size - $disk.FreeSpace)/$disk.Size * 100, 1)
Write-Output "  $usage"

Write-Output "Done: Build cache cleanup complete"
