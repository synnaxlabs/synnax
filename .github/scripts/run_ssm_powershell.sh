#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs the PowerShell script on stdin on INSTANCE through SSM and prints its output.

set -euo pipefail

: "${INSTANCE:?}" "${COMMENT:?}" "${TIMEOUT:?}"

encoded=$(iconv -t UTF-16LE | base64 | tr -d '\n')
command="powershell -NoProfile -EncodedCommand $encoded"
# SSM reports the exit code of its own script, so that script exits with ours.
id=$(aws ssm send-command --instance-ids "$INSTANCE" \
    --document-name AWS-RunPowerShellScript --comment "$COMMENT" \
    --parameters "{\"commands\":[\"$command\",\"exit \$LASTEXITCODE\"]}" \
    --query Command.CommandId --output text)

waited=0
while :; do
    if out=$(aws ssm get-command-invocation --command-id "$id" \
        --instance-id "$INSTANCE" --query Status --output text 2>&1); then
        status=$out
    # SSM answers this until the invocation reaches the instance.
    elif [[ $out == *InvocationDoesNotExist* ]]; then
        status=Pending
    else
        echo "$out" >&2
        exit 1
    fi
    case $status in
        Pending | InProgress | Delayed) ;;
        *) break ;;
    esac
    if ((waited >= TIMEOUT)); then
        echo "Command $id is still $status after $TIMEOUT seconds" >&2
        exit 1
    fi
    sleep 10
    waited=$((waited + 10))
done

aws ssm get-command-invocation --command-id "$id" --instance-id "$INSTANCE" \
    --query '[StandardOutputContent,StandardErrorContent]' --output text
[[ $status == Success ]]
