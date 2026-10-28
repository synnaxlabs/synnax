#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Sets up a seat instance that Terraform created. Run it with AWS access, the Session
# Manager plugin, and a GitHub CLI login that administers the organization's runners.
#
# Usage:
#   seat.sh register <seat>  Registers the seat's runners in the agent-runners group.
#   seat.sh login <seat>     Signs the seat's agent CLI in, interactively.

set -euo pipefail

action=$1
seat=$2
export AWS_REGION=${AWS_REGION:-us-east-1}

read -r id agent <<< "$(aws ec2 describe-instances --output text \
    --filters Name=tag:project,Values=synnax-agents Name=tag:seat,Values="$seat" \
    Name=instance-state-name,Values=pending,running,stopping,stopped \
    --query 'Reservations[].Instances[].[InstanceId,Tags[?Key==`agent`]|[0].Value]')"
if [ -z "$id" ]; then
    echo "No instance for seat $seat in $AWS_REGION" >&2
    exit 1
fi
aws ec2 start-instances --instance-ids "$id" > /dev/null
aws ec2 wait instance-running --instance-ids "$id"
until [ "$(aws ssm describe-instance-information --output text \
    --filters "Key=InstanceIds,Values=$id" \
    --query 'InstanceInformationList[0].PingStatus')" = Online ]; do
    sleep 10
done

# Runs a shell script on the instance once its bootstrap finishes, which can take half
# an hour on a new instance. cloud-init exits 2 on errors it recovered from.
run() {
    local script="cloud-init status --wait > /dev/null || [ \$? = 2 ]
$1"
    local command_id status
    command_id=$(aws ssm send-command --instance-ids "$id" --output text \
        --document-name AWS-RunShellScript --query Command.CommandId \
        --parameters "$(jq -n --arg c "$script" \
            '{commands: [$c], executionTimeout: ["3600"]}')")
    while :; do
        status=$(aws ssm get-command-invocation --command-id "$command_id" \
            --instance-id "$id" --output text --query Status 2> /dev/null || true)
        case $status in
            "" | Pending | InProgress | Delayed) sleep 15 ;;
            *) break ;;
        esac
    done
    if [ "$status" != Success ]; then
        aws ssm get-command-invocation --command-id "$command_id" \
            --instance-id "$id" --output text --query StandardErrorContent >&2
        exit 1
    fi
}

case $action in
    register)
        # The token registers runners for an hour, then expires.
        token=$(gh api -X POST orgs/synnaxlabs/actions/runners/registration-token \
            --jq .token)
        run "register-agent-runners $token"
        ;;
    login)
        case $agent in
            claude) login="claude auth login --claudeai && claude auth status" ;;
            codex) login="codex login --device-auth && codex login status" ;;
        esac
        run true
        aws ssm start-session --target "$id" \
            --document-name AWS-StartInteractiveCommand \
            --parameters "{\"command\":[\"sudo -iu runner bash -c '$login'\"]}"
        ;;
    *)
        echo "Unknown action $action: use register or login" >&2
        exit 1
        ;;
esac
