# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

param (
    [string]$FilePath
)

$extension = [System.IO.Path]::GetExtension($FilePath).ToLower()

if ($extension -eq ".exe" -or $extension -eq ".msi") {
    trusted-signing-cli -e https://wcus.codesigning.azure.net -a CodeSigningAccountName -c CodeSigningCertificateProfile -d "Synnax Console" $FilePath
}
