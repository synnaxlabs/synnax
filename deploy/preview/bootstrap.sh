#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# EC2 user data for the preview host, rendered by Terraform's templatefile.

set -euo pipefail

dnf install -y docker amazon-ecr-credential-helper
systemctl enable --now docker

mkdir -p /root/.docker /etc/caddy/sites
echo '{"credsStore": "ecr-login"}' > /root/.docker/config.json

cat > /etc/synnax-preview.env << 'EOF'
${env}
EOF

cat > /etc/caddy/Caddyfile << 'EOF'
${caddyfile}
EOF

cat > /usr/local/bin/preview << 'EOF'
${preview}
EOF
chmod +x /usr/local/bin/preview

docker network create preview
/usr/local/bin/preview upgrade

cat > /etc/systemd/system/preview-sweep.service << 'EOF'
[Unit]
Description=Remove expired preview Cores
[Service]
Type=oneshot
ExecStart=/usr/local/bin/preview sweep
EOF

cat > /etc/systemd/system/preview-sweep.timer << 'EOF'
[Unit]
Description=Remove expired preview Cores every hour
[Timer]
OnCalendar=hourly
[Install]
WantedBy=timers.target
EOF

systemctl daemon-reload
systemctl enable --now preview-sweep.timer
