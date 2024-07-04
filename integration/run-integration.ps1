param (
    [string]$testConfigName
)

if (-not $testConfigName) {
    Write-Host "Usage: .\script.ps1 <test config name>"
    exit 1
}

Set-Location -Path "..\synnax"
Write-Host "--Compiling with PGO"
go build -o "..\integration\bin\synnax" -pgo="auto"
Set-Location -Path "..\integration"

# Run the Go program with the provided arguments
& go run . $testConfigName
$exit_code = $LASTEXITCODE

# Check if the exit code is not 0
if ($exit_code -ne 0) {
    Write-Host "Test failed, see stdout for more info"
    exit $exit_code
}

Write-Host "`n"

# Find the last occurrence of "Test Started" and print everything after that
$last_occurrence = Select-String -Path "./timing.log" -Pattern "Test Started" | Select-Object -Last 1

# Check if the last occurrence is found
if (-not $last_occurrence) {
    Write-Host "No occurrences of 'Test Started' found in timing.log"
    exit 1
}

# We want everything after the last occurrence
$result = Get-Content "./timing.log" | Select-Object -Skip ($last_occurrence.LineNumber - 1)

$result | ForEach-Object { Write-Host $_ }

# Check if any assertions failed
if ($result -match "FAIL!!") {
    Write-Host "Integration test failed: at least one assertion failed"
    exit 1
}
