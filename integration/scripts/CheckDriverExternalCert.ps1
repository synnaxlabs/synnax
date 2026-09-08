# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms a Core serving an externally issued certificate boots with the embedded
# Driver enabled and the Driver connects. The Driver trusts node.crt directly, so the
# cert needs no chain to the Core CA.
#
# The served pair is signed by a CA the Core never sees: generated in a foreign
# directory and copied over without its ca.crt. Same shape as a customer serving an
# ACM or corporate PKI certificate.

$ErrorActionPreference = "Stop"

$binary = "$env:USERPROFILE\synnax-binaries\synnax.exe"
$work = "$env:USERPROFILE\synnax-external-cert-check"
$port = 9098
$deadline = 60

if (Test-Path $work) { Remove-Item -Recurse -Force $work }
New-Item -ItemType Directory -Path "$work\certs" -Force | Out-Null
New-Item -ItemType Directory -Path "$work\foreign" -Force | Out-Null

& $binary cert ca --certs-dir "$work\foreign"
& $binary cert node --certs-dir "$work\foreign" "localhost:$port"
Copy-Item "$work\foreign\node.crt", "$work\foreign\node.key" -Destination "$work\certs"

Write-Host "Starting Synnax with an externally issued certificate on localhost:$port..."
$log = "$work\core.log"
$cmdArgs = "/c `"`"$binary`" start -m -d `"$work\data`" --listen localhost:$port --certs-dir `"$work\certs`" --log-file-path `"$work\logs\synnax.log`" > `"$log`" 2>&1`""
$process = Start-Process -FilePath "cmd.exe" -ArgumentList $cmdArgs -WindowStyle Hidden -PassThru

try {
    for ($i = 0; $i -lt $deadline; $i++) {
        if ((Test-Path $log) -and (Select-String -Path $log -Pattern "started successfully" -Quiet)) {
            Write-Host "Embedded Driver connected with an externally issued certificate"
            exit 0
        }
        if ($process.HasExited) { break }
        Start-Sleep -Seconds 1
    }

    Write-Host "ERROR: the Core never started or the Driver never connected"
    if (Test-Path $log) {
        Write-Host "--- certificate failures ---"
        Select-String -Path $log -Pattern "Core CA|certificate|advertised" |
            Select-Object -First 10 |
            ForEach-Object { $_.Line.Substring(0, [Math]::Min(300, $_.Line.Length)) }
        Write-Host "--- last 40 lines ---"
        Get-Content $log -Tail 40
    }
    exit 1
} finally {
    if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue }
}
