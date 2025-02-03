Push-Location ..\synnax

$version = Get-Content ..\synnax\pkg\version\VERSION -Raw
if (-not $version) {
    Write-Error "Failed to read version file"
    exit 1
}
$version = $version.Trim()
Write-Host "Building version: $version"

Write-Host "Building synnax binary..."
& go build -o synnax-server.exe
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to build synnax binary"
    exit 1
}

# Navigate back to scripts directory
Pop-Location

Write-Host "Moving synnax binary to scripts directory..."
Move-Item -Force ..\synnax\synnax-server.exe .\

Write-Host "Building Windows installer..."
& makensis /DVERSION=$version windows-installer.nsi
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to build Windows installer"
    exit 1
}

Write-Host "Build process completed successfully!"