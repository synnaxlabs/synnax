#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Takes the seat out of the pool when its login cannot serve jobs. It tags the instance
# for the fleet controller, then drains the seat outside the job: each runner stops once
# its job ends, then the seat powers off. A drain that already runs is left alone.
#
# Usage: leave.sh limited-until <reset time> | leave.sh login-broken <start time>

set -euo pipefail

key=$1
value=$2
imds=http://169.254.169.254/latest
token=$(curl -fsS -X PUT "$imds/api/token" \
    -H "X-aws-ec2-metadata-token-ttl-seconds: 60")
meta() {
    curl -fsS -H "X-aws-ec2-metadata-token: $token" "$imds/meta-data/$1"
}
uvx --from awscli==1.46.2 aws ec2 create-tags \
    --region "$(meta placement/region)" --resources "$(meta instance-id)" \
    --tags "Key=$key,Value=$value"
sudo install -m 644 "$(dirname "$0")/drain.py" /run/agent-drain.py
sudo systemd-run --unit agent-drain python3 /run/agent-drain.py \
    || systemctl is-active --quiet agent-drain
