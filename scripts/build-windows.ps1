# Navigate to the synnax binary directory
Push-Location ..\synnax

# Read version from VERSION file
$version = Get-Content ..\synnax\pkg\version\VERSION -Raw
if (-not $version) {
    Write-Error "Failed to read version file"
    exit 1
}
$version = $version.Trim()
Write-Host "Building version: $version"

# Build the synnax binary
Write-Host "Building synnax binary..."
& go build -o synnax.exe
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to build synnax binary"
    exit 1
}

# Navigate back to scripts directory
Pop-Location

# Move the built binary to scripts directory
Write-Host "Moving synnax binary to scripts directory..."
Move-Item -Force ..\synnax\synnax.exe .\

# Copy python DLL
Write-Host "Copying Python DLL..."
Copy-Item -Force ..\computron\python_install\bin\python311.dll .\

# Build the NSIS installer
Write-Host "Building Windows installer..."
& makensis /DVERSION=$version windows-installer.nsi
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to build Windows installer"
    exit 1
}

Write-Host "Build process completed successfully!"
