#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs COMMAND on the preview host INSTANCE through SSM, prints its output, and exits
# non-zero when the command fails or runs longer than ten minutes.

set -euo pipefail

: "${COMMAND:?}" "${INSTANCE:?}"

id=$(aws ssm send-command --instance-ids "$INSTANCE" \
    --document-name AWS-RunShellScript \
    --parameters "$(jq -nc --arg c "$COMMAND" '{commands: [$c]}')" \
    --timeout-seconds 600 --query Command.CommandId --output text)

for _ in $(seq 120); do
    sleep 5
    # SSM reports no invocation for a moment after the send.
    status=$(aws ssm get-command-invocation --command-id "$id" \
        --instance-id "$INSTANCE" --query Status --output text 2> /dev/null \
        || echo Pending)
    case "$status" in
        Pending | InProgress | Delayed) continue ;;
        *) break ;;
    esac
done

aws ssm get-command-invocation --command-id "$id" --instance-id "$INSTANCE" \
    --query '[StandardOutputContent, StandardErrorContent]' --output text
echo "Status: $status"
[[ $status == Success ]]
