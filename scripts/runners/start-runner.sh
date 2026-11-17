#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Registers the GitHub runner of this Ubuntu instance again, then runs it. Started by
# github-runner.service, which agent_install.sh writes with RUNNER_PREFIX and
# RUNNER_LABELS. GITHUB_PAT comes from /etc/github-runner.env. A fresh registration on
# every start brings back a runner whose registration the scaler deleted.

set -euo pipefail

: "${RUNNER_PREFIX:?}" "${RUNNER_LABELS:?}" "${GITHUB_PAT:?}"

repo=synnaxlabs/synnax
imds=http://169.254.169.254/latest
imds_token=$(curl -fsS -X PUT "$imds/api/token" \
    -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
instance=$(curl -fsS -H "X-aws-ec2-metadata-token: $imds_token" \
    "$imds/meta-data/instance-id")
token=$(curl -fsS -X POST \
    -H "Authorization: token $GITHUB_PAT" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/$repo/actions/runners/registration-token" |
    jq -r .token)

# A job the runner shutdown cut short never ran its Stop containers step, so its Core
# service container still holds host port 9090. Nothing else runs in Docker here.
docker ps -aq | xargs -r docker rm -f
docker network prune -f

cd /home/ubuntu/actions-runner
# Without a registration on GitHub, remove skips the server and deletes the local
# config. A stale registration that is running a job fails it, and --replace below
# registers the runner anyway.
if [[ -f .runner ]]; then
    ./config.sh remove --token "$token" || true
fi
./config.sh --url "https://github.com/$repo" \
    --token "$token" \
    --name "$RUNNER_PREFIX-${instance:0:6}" \
    --labels "$RUNNER_LABELS" \
    --unattended \
    --replace
exec ./run.sh
