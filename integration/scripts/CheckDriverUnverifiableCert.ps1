# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms a Core refuses to boot when the advertised listener serves a certificate
# the embedded Driver cannot verify: not the node certificate and not signed by the
# Core CA. Guards the startup check itself; the positive checks alone cannot tell a
# relaxed check from a deleted one.

$ErrorActionPreference = "Stop"

$binary = "$env:USERPROFILE\synnax-binaries\synnax.exe"
$work = "$env:USERPROFILE\synnax-unverifiable-cert-check"
$port = 9097
$deadline = 60

if (Test-Path $work) { Remove-Item -Recurse -Force $work }
New-Item -ItemType Directory -Path "$work\certs" -Force | Out-Null
New-Item -ItemType Directory -Path "$work\foreign" -Force | Out-Null

& $binary cert ca --certs-dir "$work\certs"
& $binary cert node --certs-dir "$work\certs" "localhost:$port"
& $binary cert ca --certs-dir "$work\foreign"
& $binary cert node --certs-dir "$work\foreign" "localhost:$port"

@"
listen:
  - address: localhost:$port
    advertise: true
    cert:
      source: file
      cert: $work\foreign\node.crt
      key: $work\foreign\node.key
"@ | Set-Content -Path "$work\config.yaml" -Encoding ASCII

Write-Host "Starting Synnax with a certificate outside the trust anchors..."
$log = "$work\core.log"
$cmdArgs = "/c `"`"$binary`" start -m -d `"$work\data`" -c `"$work\config.yaml`" --certs-dir `"$work\certs`" --log-file-path `"$work\logs\synnax.log`" > `"$log`" 2>&1`""
$process = Start-Process -FilePath "cmd.exe" -ArgumentList $cmdArgs -WindowStyle Hidden -PassThru

try {
    for ($i = 0; $i -lt $deadline; $i++) {
        if ((Test-Path $log) -and (Select-String -Path $log -Pattern "the embedded Driver cannot verify it" -Quiet)) {
            Write-Host "Core refused the unverifiable advertised certificate"
            exit 0
        }
        if ((Test-Path $log) -and (Select-String -Path $log -Pattern "started successfully" -Quiet)) {
            Write-Host "ERROR: the Core booted with a certificate the Driver cannot verify"
            Get-Content $log -Tail 40
            exit 1
        }
        if ($process.HasExited) { break }
        Start-Sleep -Seconds 1
    }

    if ((Test-Path $log) -and (Select-String -Path $log -Pattern "the embedded Driver cannot verify it" -Quiet)) {
        Write-Host "Core refused the unverifiable advertised certificate"
        exit 0
    }
    Write-Host "ERROR: the Core never rejected the certificate"
    if (Test-Path $log) { Get-Content $log -Tail 40 }
    exit 1
} finally {
    if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue }
}
