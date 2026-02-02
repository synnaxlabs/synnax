param (
    [string]$FilePath
)

$extension = [System.IO.Path]::GetExtension($FilePath).ToLower()

if ($extension -eq ".exe" -or $extension -eq ".msi") {
    trusted-signing-cli -e https://wcus.codesigning.azure.net -a CodeSigningAccountName -c CodeSigningCertificateProfile -d "Synnax Console" $FilePath
}
