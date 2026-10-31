#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Installs this checkout's preview script and the applied settings on the running
# preview host through SSM, then smoke-tests the script there. Run it after terraform
# apply. The previews keep running, and the host keeps its address.

set -euo pipefail

dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
instance=$(terraform -chdir="$dir" output -raw instance_id)
env=$(terraform -chdir="$dir" output -raw host_env | base64 | tr -d '\n')
script=$(base64 < "$dir/preview.sh" | tr -d '\n')

# The new script replaces the old one only after it parses. upgrade brings Caddy and the
# running previews to the new settings. The smoke test marks and removes pull request
# 1, which never has a preview.
COMMAND="set -e
echo $env | base64 -d > /etc/synnax-preview.env
echo $script | base64 -d > /usr/local/bin/preview.new
bash -n /usr/local/bin/preview.new
chmod +x /usr/local/bin/preview.new
mv /usr/local/bin/preview.new /usr/local/bin/preview
/usr/local/bin/preview upgrade
/usr/local/bin/preview mark 1 none 0
/usr/local/bin/preview down 1
sha256sum /usr/local/bin/preview" \
    INSTANCE=$instance "$dir/../../.github/scripts/run_preview_command.sh"
