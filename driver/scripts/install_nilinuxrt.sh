#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

VERSION="${VERSION:-0.39.0-rc}"
DRIVER_BINARY="synnax-driver-v${VERSION}-nilinuxrt"
curl -LO "https://github.com/synnaxlabs/synnax/releases/download/synnax-v${VERSION}/${DRIVER_BINARY}"
chmod +x $DRIVER_BINARY
./$DRIVER_BINARY stop
sudo ./$DRIVER_BINARY install
./$DRIVER_BINARY start
