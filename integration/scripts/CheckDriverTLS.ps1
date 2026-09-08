# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Confirms the embedded Driver connects to a Core serving TLS. The rest of the
# integration suite runs insecure, so this is the only coverage of that path.
#
# The listen list with an auto advertised listener is load-bearing. A single
# file-source listener serves node.crt, which is what the Driver used to pin, so it
# passes even when nothing else can verify the Core.

$ErrorActionPreference = "Stop"

$binary = "$env:USERPROFILE\synnax-binaries\synnax.exe"
$work = "$env:USERPROFILE\synnax-tls-check"
$port = 9099
$deadline = 60

if (Test-Path $work) { Remove-Item -Recurse -Force $work }
New-Item -ItemType Directory -Path "$work\certs" -Force | Out-Null

@"
listen:
  - address: localhost:$port
    advertise: true
    cert:
      source: auto
"@ | Set-Content -Path "$work\config.yaml" -Encoding ASCII

# A listen list rejects --auto-cert, and the security provider loads the node pair
# whatever source the listeners use, so generate both the way an operator would.
& $binary cert ca --certs-dir "$work\certs"
& $binary cert node --certs-dir "$work\certs" "localhost:$port"

Write-Host "Starting Synnax with TLS on localhost:$port..."
$log = "$work\core.log"
$cmdArgs = "/c `"`"$binary`" start -m -d `"$work\data`" -c `"$work\config.yaml`" --certs-dir `"$work\certs`" --log-file-path `"$work\logs\synnax.log`" > `"$log`" 2>&1`""
$process = Start-Process -FilePath "cmd.exe" -ArgumentList $cmdArgs -WindowStyle Hidden -PassThru

try {
    for ($i = 0; $i -lt $deadline; $i++) {
        if ((Test-Path $log) -and (Select-String -Path $log -Pattern "started successfully" -Quiet)) {
            Write-Host "Embedded Driver connected over TLS"
            exit 0
        }
        if ($process.HasExited) { break }
        Start-Sleep -Seconds 1
    }

    Write-Host "ERROR: the embedded Driver never connected to a Core serving TLS"
    if (Test-Path $log) {
        Write-Host "--- handshake failures ---"
        Select-String -Path $log -Pattern "handshake|certificate|verify" |
            Select-Object -First 10 |
            ForEach-Object { $_.Line.Substring(0, [Math]::Min(300, $_.Line.Length)) }
        Write-Host "--- last 40 lines ---"
        Get-Content $log -Tail 40
    }
    exit 1
} finally {
    if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue }
}
