#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This script registers an MQTT broker as a device in Synnax.

MQTT tasks run on the driver inside the Core, so the device goes on the rack of that
driver. If the broker is already registered, the script exits.

Before running this example:
1. Start the mock MQTT plant (or use your own broker):
   uv run python examples/mqtt_sim/server.py

2. Login to Synnax (if not already logged in):
   uv run sy login

3. Run this script:
   uv run python examples/mqtt_sim/connect_broker.py

Configuration:
    Modify the constants below to match your broker.
"""

import synnax as sy

DEVICE_NAME = "MQTT Broker"
HOST = "127.0.0.1"
PORT = 1884

client = sy.Synnax()

existing = client.devices.retrieve(name=DEVICE_NAME, ignore_not_found=True)
if existing is not None:
    print(f"Broker already connected: {existing.name} ({existing.location})")
    exit(0)

rack = client.racks.retrieve(integration=sy.mqtt.INTEGRATION)
print(f"Using rack: {rack.name} (key={rack.key})")
device = client.devices.create(
    sy.mqtt.Device(host=HOST, port=PORT, name=DEVICE_NAME, rack=rack.key)
)
print(f"Connected broker {device.name} at {HOST}:{PORT}")
