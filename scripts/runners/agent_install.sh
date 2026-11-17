#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Installs agent_boot.py at a commit on an Ubuntu runner and restarts the agent. Also
# installs start-runner.sh and github-runner.service for the runner name prefix and
# labels of the runner's pool. A running GitHub runner keeps running.

set -euo pipefail

commit=$1
host=$2
ca=$3
prefix=$4
labels=$5

dir=/opt/runner-telemetry
runner=/home/ubuntu/actions-runner
uv=https://github.com/astral-sh/uv/releases/download/0.12.5
source=https://raw.githubusercontent.com/synnaxlabs/synnax/$commit/scripts/runners

[[ -x $runner/config.sh ]] || {
    echo "no GitHub runner in $runner" >&2
    exit 1
}
grep -q '^GITHUB_PAT=.' /etc/github-runner.env || {
    echo "/etc/github-runner.env has no GITHUB_PAT" >&2
    exit 1
}
mkdir -p "$dir"
if [[ ! -x $dir/uv ]]; then
    curl -fsSL -o /tmp/uv.tgz "$uv/uv-x86_64-unknown-linux-gnu.tar.gz"
    tar -xzf /tmp/uv.tgz -C "$dir" --strip-components=1
    rm -f /tmp/uv.tgz
fi
# Every download ends before the first write, so a failure keeps the old install.
staging=$(mktemp -d)
trap 'rm -rf "$staging"' EXIT
for file in agent_boot.py pyproject.toml uv.lock start-runner.sh; do
    curl -fsSL -o "$staging/$file" "$source/$file"
done
# A rename gives the script a new inode, so a running runner keeps reading the old one.
install -m 755 "$staging/start-runner.sh" "$runner/start-runner.sh.new"
mv "$runner/start-runner.sh.new" "$runner/start-runner.sh"
rm "$staging/start-runner.sh"
cat > /etc/systemd/system/github-runner.service << EOF
[Unit]
Description=GitHub Actions Runner
After=network-online.target ec2-reset.service docker.service
Wants=network-online.target
Wants=ec2-reset.service
Wants=docker.service

[Service]
# One OOM-killed child must not take the Worker down with it: systemd would then
# tear down the whole cgroup and the job would skip its Stop containers step.
OOMPolicy=continue
Type=simple
User=ubuntu
WorkingDirectory=$runner
EnvironmentFile=/etc/github-runner.env
Environment="RUNNER_PREFIX=$prefix" "RUNNER_LABELS=$labels"
ExecStart=$runner/start-runner.sh
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
# The agent starts the GitHub runner, so the runner must not start at boot.
if systemctl is-enabled --quiet github-runner.service 2> /dev/null; then
    systemctl disable github-runner.service
fi
mv "$staging"/* "$dir/"
# agent_boot.py then downloads every agent file at the deployed commit.
rm -f "$dir/agent_commit.txt"
base64 -d <<< "$ca" > "$dir/ca.crt"
log=$dir/agent.out.log
if [[ -f $log ]] && (($(stat -c%s "$log") > 5242880)); then
    : > "$log"
fi
cat > /etc/systemd/system/runner-agent.service << EOF
[Unit]
Description=Synnax runner telemetry
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$dir
Environment=UV_CACHE_DIR=$dir/uv-cache
ExecStart=$dir/uv run --project $dir --frozen --no-dev $dir/agent_boot.py \\
    --host $host --secure
Restart=always
RestartSec=15
Nice=19
IOSchedulingClass=idle
StandardOutput=append:$log
StandardError=append:$log

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable runner-agent.service > /dev/null 2>&1
systemctl restart runner-agent.service
sleep 5
if ! systemctl is-active --quiet runner-agent.service; then
    echo "runner-agent.service did not stay up" >&2
    exit 1
fi
echo "Installed agent_boot.py at $commit"
