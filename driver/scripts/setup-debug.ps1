# Downloads driver debug artifacts and opens Visual Studio.
# Usage: .\setup-debug.ps1 <run-id> [-Args "start -s"] [-OutputDir <path>] [-NoLaunch]
#
# Example: .\setup-debug.ps1 21687324829
# Example: .\setup-debug.ps1 21687324829 -Args "start -s --host localhost"

param(
    [Parameter(Mandatory=$true, Position=0)]
    [string]$RunId,

    [Parameter(Mandatory=$false)]
    [string]$OutputDir = ".\debugdriver",

    [Parameter(Mandatory=$false)]
    [string]$Args = "start -s",

    [Parameter(Mandatory=$false)]
    [switch]$NoLaunch
)

$ErrorActionPreference = "Stop"

Write-Host "Downloading artifacts from run $RunId..."

# Create output directory
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$TempDir = Join-Path $OutputDir "tmp"

# Download artifacts
try {
    gh run download $RunId `
        --repo synnaxlabs/synnax `
        --name synnax-driver-windows `
        --name synnax-driver-debug-symbols-windows `
        --dir $TempDir
} catch {
    gh run download $RunId `
        --repo synnaxlabs/synnax `
        --pattern "*driver*windows*" `
        --dir $TempDir
}

# Find and move exe
$Exe = Get-ChildItem -Path $TempDir -Filter "*.exe" -Recurse | Select-Object -First 1
if (-not $Exe) {
    Write-Error "No exe found in artifacts"
    exit 1
}
Move-Item -Force $Exe.FullName (Join-Path $OutputDir "driver.exe")

# Find and move pdb
$Pdb = Get-ChildItem -Path $TempDir -Filter "*.pdb" -Recurse | Select-Object -First 1
if (-not $Pdb) {
    Write-Error "No pdb found in artifacts"
    exit 1
}
Move-Item -Force $Pdb.FullName (Join-Path $OutputDir "driver.pdb")

# Cleanup
Remove-Item -Recurse -Force $TempDir

$DriverPath = Join-Path (Resolve-Path $OutputDir) "driver.exe"

# Create launch.vs.json for debug arguments
$VsDir = Join-Path $OutputDir ".vs"
New-Item -ItemType Directory -Force -Path $VsDir | Out-Null

# Split args into array for JSON
$ArgsArray = ($Args -split ' ' | ForEach-Object { "`"$_`"" }) -join ", "

$LaunchJson = @"
{
  "version": "0.2.1",
  "configurations": [
    {
      "type": "default",
      "project": "driver.exe",
      "name": "driver.exe",
      "args": [ $ArgsArray ]
    }
  ]
}
"@
$LaunchJson | Out-File -FilePath (Join-Path $VsDir "launch.vs.json") -Encoding utf8
Write-Host ""
Write-Host "Ready to debug:" -ForegroundColor Green
Write-Host "  $DriverPath"
Write-Host "  $(Join-Path (Resolve-Path $OutputDir) 'driver.pdb')"

if (-not $NoLaunch) {
    Write-Host ""
    Write-Host "Opening Visual Studio..."

    # Find Visual Studio
    $VsPaths = @(
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\Community\Common7\IDE\devenv.exe",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\Professional\Common7\IDE\devenv.exe",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\Enterprise\Common7\IDE\devenv.exe",
        "${env:ProgramFiles}\Microsoft Visual Studio\2019\Community\Common7\IDE\devenv.exe",
        "${env:ProgramFiles(x86)}\Microsoft Visual Studio\2022\BuildTools\Common7\IDE\devenv.exe"
    )

    # Also check for VS Preview/Insiders
    $VsPaths += Get-ChildItem "${env:ProgramFiles}\Microsoft Visual Studio" -Directory -ErrorAction SilentlyContinue |
        ForEach-Object { Join-Path $_.FullName "*/Common7/IDE/devenv.exe" } |
        Resolve-Path -ErrorAction SilentlyContinue |
        Select-Object -ExpandProperty Path

    $DevEnv = $VsPaths | Where-Object { Test-Path $_ } | Select-Object -First 1

    if ($DevEnv) {
        # Open folder (not exe) so VS picks up launch.vs.json
        $FolderPath = Resolve-Path $OutputDir
        & $DevEnv $FolderPath
        Write-Host "Select 'driver.exe' from the debug dropdown and press F5"
    } else {
        Write-Warning "Visual Studio not found. Open folder manually: $OutputDir"
        Start-Process explorer.exe -ArgumentList "`"$(Resolve-Path $OutputDir)`""
    }
}
