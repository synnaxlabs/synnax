#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs a PowerShell script on INSTANCE through SSM, with the given Name=value variables.

set -euo pipefail

: "${INSTANCE:?}" "${COMMENT:?}" "${TIMEOUT:?}"

script=$1
shift
quote="'"
encoded=$({
    for variable in "$@"; do
        value=${variable#*=}
        printf "\$%s = '%s'\n" "${variable%%=*}" "${value//$quote/$quote$quote}"
    done
    cat "$script"
} | iconv -t UTF-16LE | base64 | tr -d '\n')
command="powershell -NoProfile -EncodedCommand $encoded"
# SSM reports the exit code of its own script, so that script exits with ours.
parameters="{\"commands\":[\"$command\",\"exit \$LASTEXITCODE\"]"
# SSM ends the command at TIMEOUT, so it does not run on after this script gives up.
parameters+=",\"executionTimeout\":[\"$TIMEOUT\"]}"
id=$(aws ssm send-command --instance-ids "$INSTANCE" \
    --document-name AWS-RunPowerShellScript --comment "$COMMENT" \
    --parameters "$parameters" --query Command.CommandId --output text)

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
