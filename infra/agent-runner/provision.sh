#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Bootstraps a seat instance on Ubuntu 24.04. cloud-init runs it once, as root. It is a
# Terraform template: write $VAR, never the brace form, which Terraform interpolates.

set -euxo pipefail

GO_VERSION=1.27.2
NODE_MAJOR=24
RUNNER_VERSION=2.338.2
BAZELISK_VERSION=1.30.0
UV_VERSION=0.13.0
PNPM_VERSION=12.14.0
GOLANGCI_LINT_VERSION=2.15.0
ORG=synnaxlabs
REPO=$ORG/synnax
HOME_DIR=/home/runner
IDLE_MINUTES=${idle_minutes}
SEAT=${seat}
AGENT=${agent}
RUNNERS=${runners}
export DEBIAN_FRONTEND=noninteractive

# Stops the instance once no job has run for the idle period. It goes first, so a
# failed bootstrap still stops. The first hour of bootstrap counts as busy.
cat > /usr/local/bin/agent-idle-stop << EOF
#!/bin/bash
stamp=/run/agent-last-busy
booting() {
    cloud-init status | grep -q running && [ "\$(cut -d. -f1 /proc/uptime)" -lt 3600 ]
}
if pgrep -f Runner.Worker > /dev/null || booting || [ ! -e \$stamp ]; then
    touch \$stamp
elif [ \$((\$(date +%s) - \$(stat -c %Y \$stamp))) -ge $((IDLE_MINUTES * 60)) ]; then
    shutdown -h now
fi
EOF
chmod +x /usr/local/bin/agent-idle-stop
cat > /etc/systemd/system/agent-idle-stop.service << EOF
[Service]
Type=oneshot
ExecStart=/usr/local/bin/agent-idle-stop
EOF
cat > /etc/systemd/system/agent-idle-stop.timer << EOF
[Timer]
OnBootSec=1min
OnUnitActiveSec=1min
[Install]
WantedBy=timers.target
EOF
systemctl daemon-reload
systemctl enable --now agent-idle-stop.timer

apt-get update
apt-get install -y build-essential git curl unzip jq docker.io python3 \
    libsystemd-dev autoconf automake libtool
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    -o /usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg]" \
    "https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/gh.list
curl -fsSL "https://deb.nodesource.com/setup_$NODE_MAJOR.x" | bash -
apt-get install -y gh nodejs

curl -fsSL "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz" | tar -C /usr/local -xz
BAZELISK_URL=https://github.com/bazelbuild/bazelisk/releases/download
curl -fsSL -o /usr/local/bin/bazel \
    "$BAZELISK_URL/v$BAZELISK_VERSION/bazelisk-linux-amd64"
chmod +x /usr/local/bin/bazel
curl -LsSf "https://astral.sh/uv/$UV_VERSION/install.sh" \
    | env UV_INSTALL_DIR=/usr/local/bin sh

# The pre-commit hook formats Go and C++, so the formatters CI pins are installed.
curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
    | sh -s -- -b /usr/local/bin "v$GOLANGCI_LINT_VERSION"
RAW_URL=https://raw.githubusercontent.com/$REPO/main
curl -fsSL "$RAW_URL/scripts/install_clang_format.sh" \
    | env UV_TOOL_DIR=/opt/uv-tools UV_TOOL_BIN_DIR=/usr/local/bin bash
# Each agent job updates the agent CLIs to the latest release before it runs.
npm install -g "pnpm@$PNPM_VERSION" @anthropic-ai/claude-code@latest \
    @openai/codex@latest

# The agent builds, installs packages, and runs Docker, so the user is an admin of
# this single-purpose machine.
useradd --create-home --shell /bin/bash --groups docker runner
echo "runner ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/runner
cat > "$HOME_DIR/.bazelrc" << EOF
common --disk_cache=$HOME_DIR/.cache/bazel-disk
common --repository_cache=$HOME_DIR/.cache/bazel-repo
common --experimental_disk_cache_gc_max_size=100G
EOF

# Each runner copies this install, then updates itself.
RUNNER_DIR=/opt/actions-runner
mkdir -p "$RUNNER_DIR"
RUNNER_URL=https://github.com/actions/runner/releases/download/v$RUNNER_VERSION
curl -fsSL "$RUNNER_URL/actions-runner-linux-x64-$RUNNER_VERSION.tar.gz" \
    | tar -C "$RUNNER_DIR" -xz
"$RUNNER_DIR/bin/installdependencies.sh"
echo "PATH=/usr/local/go/bin:$HOME_DIR/go/bin:/usr/local/bin:/usr/bin:/bin" \
    > "$RUNNER_DIR/.env"
chown -R runner:runner "$HOME_DIR"

# Registers the seat's runners in the agent-runners group and starts each as a service.
# All run as the runner user, so they share its agent login. A Claude seat adds a runner
# that only Label - Classify uses, so a label never waits behind a long agent job. It
# is safe to run again.
# Usage: register-agent-runners <organization registration token>
cat > /usr/local/bin/register-agent-runners << EOF
#!/bin/bash
set -euo pipefail
token=\$1
register() {
    dir=$HOME_DIR/runners/\$1
    if [ ! -e "\$dir/.runner" ]; then
        mkdir -p "\$dir"
        cp -R $RUNNER_DIR/. "\$dir"
        chown -R runner:runner "\$dir"
        cd "\$dir"
        sudo -u runner ./config.sh --unattended --replace \
            --url https://github.com/$ORG --runnergroup agent-runners \
            --token "\$token" --name "agent-$SEAT-\$1" --labels "\$2"
    fi
    cd "\$dir"
    if [ ! -e .service ]; then ./svc.sh install runner; fi
    ./svc.sh start
}
for i in \$(seq $RUNNERS); do register "\$i" "agent,agent-$AGENT,seat-$SEAT"; done
if [ $AGENT = claude ]; then register classify "agent-classify,seat-$SEAT"; fi
EOF
chmod +x /usr/local/bin/register-agent-runners

# Warms the Go build cache and the pnpm store before the first job. A failed warm only
# makes the first job slower, so it must not stop the idle timer from being installed.
if ! timeout 30m sudo -H -u runner bash -euxc "
    cd $HOME_DIR
    git clone --depth 1 https://github.com/$REPO warm
    cd warm/core && /usr/local/go/bin/go build ./...
    cd .. && pnpm install --frozen-lockfile
"; then
    echo "Warming the caches failed" >&2
fi
rm -rf "$HOME_DIR/warm"
