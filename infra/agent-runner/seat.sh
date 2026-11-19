#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Sets up a seat instance that Terraform created, or stores a Claude login. Run it with
# AWS access, the Session Manager plugin, and a GitHub CLI login that administers the
# organization's runners.
#
# Usage:
#   seat.sh register <seat>  Registers the seat's runners in the agent-runners group,
#                            as many as its `runners` tag, and starts each as a service.
#                            It adds runners, never removes them, and is safe to run
#                            again.
#   seat.sh login <seat>     Signs the seat's agent CLI in, interactively.
#   seat.sh token <login>    Stores a token from `claude setup-token`, read from stdin,
#                            as the login. It clears the login's usage limit and broken
#                            mark, and records when the token expires.

set -euo pipefail

action=$1
seat=$2
export AWS_REGION=${AWS_REGION:-us-east-1}

# Each Claude subscription is one login: a secret that holds its one-year token. The
# fleet controller reads the tags of the secret, never its value.
if [ "$action" = token ]; then
    if [[ ! $seat =~ ^[a-z0-9-]+$ ]]; then
        echo "Login names use a-z, 0-9, and dashes" >&2
        exit 1
    fi
    name=synnax-agents/logins/$seat
    # The fleet controller and the jobs read the logins in this region.
    export AWS_REGION=us-east-1
    if [ -t 0 ]; then echo "Paste the token from claude setup-token:" >&2; fi
    # A pasted or piped token can lack the final newline, which makes read fail.
    IFS= read -rs token || true
    if [[ ! $token =~ ^sk-ant-oat01-[A-Za-z0-9_-]+$ ]]; then
        echo "That is not a token from claude setup-token" >&2
        exit 1
    fi
    # A file keeps the token out of the process list.
    file=$(mktemp)
    trap 'rm -f "$file"' EXIT
    printf '%s' "$token" > "$file"
    unset token
    # The fleet controller reads the store time as a year before expires.
    expires=$(($(date +%s) + 365 * 24 * 3600))
    if deleted=$(aws secretsmanager describe-secret --secret-id "$name" \
        --query DeletedDate --output text 2> /dev/null); then
        # A deleted secret takes no new value until it is restored.
        if [ "$deleted" != None ]; then
            aws secretsmanager restore-secret --secret-id "$name" > /dev/null
        fi
        aws secretsmanager put-secret-value --secret-id "$name" \
            --secret-string "file://$file" > /dev/null
        aws secretsmanager untag-resource --secret-id "$name" \
            --tag-keys limited-until broken
        aws secretsmanager tag-resource --secret-id "$name" \
            --tags "Key=expires,Value=$expires"
    else
        aws secretsmanager create-secret --name "$name" \
            --secret-string "file://$file" --tags Key=project,Value=synnax-agents \
            Key=agent,Value=claude "Key=expires,Value=$expires" > /dev/null
    fi
    # GNU date reads -d, BSD date reads -r.
    day=$(date -u -d "@$expires" +%F 2> /dev/null || date -u -r "$expires" +%F)
    echo "Stored login $seat. Its token expires $day."
    exit 0
fi

tag() { echo "Tags[?Key==\`$1\`]|[0].Value"; }
fields="InstanceId,$(tag agent),$(tag pool),$(tag runners)"
read -r id agent pool runners <<< "$(aws ec2 describe-instances --output text \
    --filters Name=tag:project,Values=synnax-agents Name=tag:seat,Values="$seat" \
    Name=instance-state-name,Values=pending,running,stopping,stopped \
    --query "Reservations[].Instances[].[$fields]")"
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
        if [[ ! $runners =~ ^[1-9][0-9]*$ ]]; then
            echo "Seat $seat has no runners tag: run terraform apply" >&2
            exit 1
        fi
        if [ "$pool" = review ]; then
            labels=review,seat-$seat
        else
            labels=agent,agent-$agent,seat-$seat
        fi
        # All runners run as the runner user, so they share its agent login. SSM runs
        # the script with sh, which has no pipefail.
        run "set -eu
for i in \$(seq $runners); do
    dir=/home/runner/runners/\$i
    if [ ! -e \$dir/.runner ]; then
        mkdir -p \$dir
        cp -R /opt/actions-runner/. \$dir
        chown -R runner:runner \$dir
        cd \$dir
        sudo -u runner ./config.sh --unattended --replace \\
            --url https://github.com/synnaxlabs --runnergroup agent-runners \\
            --token $token --name agent-$seat-\$i --labels $labels
    fi
    cd \$dir
    if [ ! -e .service ]; then ./svc.sh install runner; fi
    ./svc.sh start
done"
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
        # Check - Runners records the new state when it next finds the seat running.
        aws ec2 delete-tags --resources "$id" --tags Key=auth Key=auth-checked
        ;;
    *)
        echo "Unknown action $action: use register, login, or token" >&2
        exit 1
        ;;
esac
