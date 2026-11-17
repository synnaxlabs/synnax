#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs the GitHub runner of this Ubuntu instance, and registers it again first when
# GitHub does not list it. Started by github-runner.service, which agent_install.sh
# writes with RUNNER_PREFIX and RUNNER_LABELS. GITHUB_PAT comes from
# /etc/github-runner.env. A runner GitHub lists keeps its registration, so it can take a
# job GitHub bound to it. One whose registration the scaler deleted registers again.

set -euo pipefail

: "${RUNNER_PREFIX:?}" "${RUNNER_LABELS:?}" "${GITHUB_PAT:?}"

repo=synnaxlabs/synnax
api=https://api.github.com/repos/$repo/actions/runners
imds=http://169.254.169.254/latest
imds_token=$(curl -fsS -X PUT "$imds/api/token" \
    -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
instance=$(curl -fsS -H "X-aws-ec2-metadata-token: $imds_token" \
    "$imds/meta-data/instance-id")
name=$RUNNER_PREFIX-${instance:0:6}
github=(-H "Authorization: token $GITHUB_PAT" -H "Accept: application/vnd.github+json")
listing=$(curl -fsS "${github[@]}" "$api?name=$name")
listed=$(sed -n 's/.*"total_count": *\([0-9]*\).*/\1/p' <<< "$listing")
: "${listed:?no runner count in the GitHub response}"

# A job the runner shutdown cut short never ran its Stop containers step, so its Core
# service container still holds host port 9090. Without Docker, no container is left.
if command -v docker > /dev/null; then
    docker ps -aq | xargs -r docker rm -f
    docker network prune -f
fi

cd /home/ubuntu/actions-runner
if [[ -f .runner ]] && ((listed > 0)); then
    # Sets the pool's labels in place, which a new registration would do but would
    # strand a job GitHub bound to this one. GitHub sets the default labels itself.
    id=$(grep -o '"id": *[0-9]*' <<< "$listing" | sed -n '1s/[^0-9]//gp')
    custom=$(tr ',' '\n' <<< "$RUNNER_LABELS" \
        | grep -viE '^(self-hosted|linux|windows|x64)$' \
        | sed 's/.*/"&"/' | paste -sd, -)
    curl -fsS -X PUT "${github[@]}" "$api/$id/labels" \
        -d "{\"labels\":[$custom]}" > /dev/null
    exec ./run.sh
fi
token=$(curl -fsS -X POST "${github[@]}" "$api/registration-token" \
    | sed -n 's/.*"token": *"\([^"]*\)".*/\1/p')
: "${token:?no registration token in the GitHub response}"
# Without a registration on GitHub, remove skips the server and deletes the local
# config. When remove fails, the local config goes so that --replace can register.
if [[ -f .runner ]]; then
    ./config.sh remove --token "$token" \
        || rm -f .runner .credentials .credentials_rsaparams
fi
./config.sh --url "https://github.com/$repo" \
    --token "$token" \
    --name "$name" \
    --labels "$RUNNER_LABELS" \
    --unattended \
    --replace
exec ./run.sh
