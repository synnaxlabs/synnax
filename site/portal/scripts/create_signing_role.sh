#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Creates the AWS role the portal signs license keys with. Only production deployments
# of the Vercel project can assume it, through Vercel's OIDC tokens. Safe to rerun: it
# updates the trust and signing policies in place. Needs an AWS CLI identity that can
# manage IAM.
#
#   VERCEL_TEAM=synnaxlabs scripts/create_signing_role.sh

set -euo pipefail

: "${VERCEL_TEAM:?set VERCEL_TEAM to the Vercel team slug}"
PROJECT="${VERCEL_PROJECT:-portal}"
KEY_ALIAS="${KEY_ALIAS:-alias/synnax-license-signing}"
ROLE=portal-signer

ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
ISSUER="oidc.vercel.com/$VERCEL_TEAM"
AUDIENCE="https://vercel.com/$VERCEL_TEAM"
PROVIDER_ARN="arn:aws:iam::$ACCOUNT:oidc-provider/$ISSUER"
KEY_ARN=$(
  aws kms describe-key --key-id "$KEY_ALIAS" --query KeyMetadata.Arn --output text
)

if ! aws iam get-open-id-connect-provider \
  --open-id-connect-provider-arn "$PROVIDER_ARN" >/dev/null 2>&1; then
  aws iam create-open-id-connect-provider \
    --url "https://$ISSUER" --client-id-list "$AUDIENCE" >/dev/null
fi

TRUST=$(
  cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "Federated": "$PROVIDER_ARN" },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {
        "$ISSUER:aud": "$AUDIENCE",
        "$ISSUER:sub": "owner:$VERCEL_TEAM:project:$PROJECT:environment:production"
      }
    }
  }]
}
EOF
)

SIGN=$(
  cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["kms:Sign", "kms:GetPublicKey"],
    "Resource": "$KEY_ARN"
  }]
}
EOF
)

if aws iam get-role --role-name "$ROLE" >/dev/null 2>&1; then
  aws iam update-assume-role-policy --role-name "$ROLE" --policy-document "$TRUST"
else
  aws iam create-role --role-name "$ROLE" --assume-role-policy-document "$TRUST" \
    >/dev/null
fi
aws iam put-role-policy --role-name "$ROLE" --policy-name sign-licenses \
  --policy-document "$SIGN"

ROLE_ARN=$(aws iam get-role --role-name "$ROLE" --query Role.Arn --output text)
echo "Set these on every environment of the $PROJECT Vercel project:"
echo "  AWS_ROLE_ARN=$ROLE_ARN"
echo "  AWS_REGION=$(echo "$KEY_ARN" | cut -d: -f4)"
echo "  LICENSE_KMS_KEY_ARN=$KEY_ARN"
