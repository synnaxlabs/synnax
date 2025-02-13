#!/bin/bash
VERSION="${VERSION:-0.39.0-rc}"
DRIVER_BINARY="synnax-driver-v${VERSION}-nilinuxrt"
curl -LO "https://github.com/synnaxlabs/synnax/releases/download/synnax-v${VERSION}/${DRIVER_BINARY}"
chmod +x $DRIVER_BINARY
./$DRIVER_BINARY stop
sudo ./$DRIVER_BINARY install
./$DRIVER_BINARY start
