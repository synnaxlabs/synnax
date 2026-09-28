# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

output "license_kms_key_arn" {
  value = aws_kms_key.license_signing.arn
}

# The Ed25519 public key as SPKI DER. The last 32 bytes, base64 encoded, are the anchor
# the Core embeds for this kid.
data "aws_kms_public_key" "license_signing" {
  key_id = aws_kms_key.license_signing.key_id
}

output "license_public_key" {
  value = data.aws_kms_public_key.license_signing.public_key
}
