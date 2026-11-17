# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs the GitHub runner of this Windows instance, and registers it again first when
# GitHub does not list it. Started by the GitHubActionsRunner task, which
# agent_install.ps1 registers with $Prefix and $Labels. GITHUB_PAT is a machine
# environment variable. A runner GitHub lists keeps its registration, so it can take a
# job GitHub bound to it. One whose registration the scaler deleted registers again.

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
$name = "$Prefix-$($instance.Substring(0, 6))"
$api = "https://api.github.com/repos/$repo/actions/runners"
$headers = @{ Authorization = "token $pat"; Accept = 'application/vnd.github+json' }
$listing = Invoke-RestMethod -Headers $headers -Uri "$api`?name=$name"
$listed = $listing.total_count
if ($null -eq $listed) { throw 'No runner count in the GitHub response' }

Set-Location $PSScriptRoot
if ((Test-Path '.runner') -and $listed -gt 0) {
    # Sets the pool's labels in place, which a new registration would do but would
    # strand a job GitHub bound to this one. GitHub sets the default labels itself.
    $custom = @($Labels -split ',' |
            Where-Object { $_ -notmatch '^(self-hosted|linux|windows|x64)$' })
    $body = @{ labels = $custom } | ConvertTo-Json -Compress
    Invoke-RestMethod -Method Put -Headers $headers -Body $body `
        -ContentType 'application/json' `
        -Uri "$api/$($listing.runners[0].id)/labels" | Out-Null
    & .\run.cmd
    exit $LASTEXITCODE
}
$token = (Invoke-RestMethod -Method Post -Headers $headers `
        -Uri "$api/registration-token").token
# Without a registration on GitHub, remove skips the server and deletes the local
# config. When remove fails, the local config goes so that --replace can register.
if (Test-Path '.runner') {
    $removed = $false
    try {
        & .\config.cmd remove --token $token 2>$null
        $removed = $LASTEXITCODE -eq 0
    } catch { Write-Output $_ }
    if (-not $removed) {
        Remove-Item '.runner', '.credentials', '.credentials_rsaparams' `
            -Force -ErrorAction Ignore
    }
}
& .\config.cmd --url "https://github.com/$repo" --token $token `
    --name $name --labels $Labels --unattended --replace
if ($LASTEXITCODE -ne 0) { throw "config exit $LASTEXITCODE" }
& .\run.cmd
exit $LASTEXITCODE
