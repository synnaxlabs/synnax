# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Registers the GitHub runner of this Windows instance again, then runs it. Started by
# the GitHubActionsRunner task, which agent_install.ps1 registers with $Prefix and
# $Labels. GITHUB_PAT is a machine environment variable. A fresh registration on every
# start brings back a runner whose registration the scaler deleted.

param($Prefix, $Labels)

$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

if (-not $Prefix -or -not $Labels) { throw 'Prefix and Labels are required' }
$pat = [Environment]::GetEnvironmentVariable('GITHUB_PAT', 'Machine')
if (-not $pat) { throw 'GITHUB_PAT is not set' }

$repo = 'synnaxlabs/synnax'
$imds = 'http://169.254.169.254/latest'
$imdsToken = Invoke-RestMethod -Uri "$imds/api/token" -Method Put `
    -Headers @{ 'X-aws-ec2-metadata-token-ttl-seconds' = '21600' }
$instance = Invoke-RestMethod -Uri "$imds/meta-data/instance-id" `
    -Headers @{ 'X-aws-ec2-metadata-token' = $imdsToken }
$token = (Invoke-RestMethod -Method Post `
        -Uri "https://api.github.com/repos/$repo/actions/runners/registration-token" `
        -Headers @{ Authorization = "token $pat"; Accept = 'application/vnd.github+json' }
).token

Set-Location 'C:\actions-runner'
# Without a registration on GitHub, remove skips the server and deletes the local
# config. A stale registration that is running a job fails it, and --replace below
# registers the runner anyway.
if (Test-Path '.runner') {
    try { & .\config.cmd remove --token $token 2>$null } catch { Write-Output $_ }
    if ($LASTEXITCODE -ne 0) { Write-Output "config remove exit $LASTEXITCODE" }
}
& .\config.cmd --url "https://github.com/$repo" --token $token `
    --name "$Prefix-$($instance.Substring(0, 6))" --labels $Labels --unattended --replace
if ($LASTEXITCODE -ne 0) { throw "config exit $LASTEXITCODE" }
& .\run.cmd
exit $LASTEXITCODE
