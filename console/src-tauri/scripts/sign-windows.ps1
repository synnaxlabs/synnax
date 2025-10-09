param (
    [string]$FilePath
)

$extension = [System.IO.Path]::GetExtension($FilePath).ToLower()

if ($extension -eq ".exe" -or $extension -eq ".msi") {
    relic sign --file $FilePath --key azure --config ..\..\relic.conf
}