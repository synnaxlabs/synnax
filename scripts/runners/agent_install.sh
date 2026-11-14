#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Installs agent_boot.py at a commit on an Ubuntu runner and restarts the agent.

set -euo pipefail

commit=$1
host=$2
ca=$3

dir=/opt/runner-telemetry
uv=https://github.com/astral-sh/uv/releases/download/0.12.5
source=https://raw.githubusercontent.com/synnaxlabs/synnax/$commit/scripts/runners

mkdir -p "$dir"
if [[ ! -x $dir/uv ]]; then
    curl -fsSL -o /tmp/uv.tgz "$uv/uv-x86_64-unknown-linux-gnu.tar.gz"
    tar -xzf /tmp/uv.tgz -C "$dir" --strip-components=1
    rm -f /tmp/uv.tgz
fi
# The agent starts the GitHub runner, so the runner must not start at boot.
if systemctl is-enabled --quiet github-runner.service 2> /dev/null; then
    systemctl disable github-runner.service
fi
curl -fsSL -o "$dir/agent_boot.py" "$source/agent_boot.py"
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
ExecStart=$dir/uv run $dir/agent_boot.py --host $host --secure
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
