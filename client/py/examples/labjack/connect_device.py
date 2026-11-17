#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This script names the LabJack device that the Synnax Driver found, so that the other
examples can retrieve it by name.

The Driver scans USB and the network for LabJacks every few seconds. It creates a
Synnax device for each one it finds, keyed by the device's serial number.

Before running this example:
1. Start the Synnax Driver (if not already running).

2. Connect your LabJack device via USB, Ethernet, or WiFi. If the Driver does not find
   a network device, add the device's IP address to LJM's ljm_specific_ips.config file.

3. Login to Synnax (if not already logged in):
   uv run sy login

4. Run this script:
   uv run python examples/labjack/connect_device.py

Configuration:
    Modify the constants below to match your LabJack device.
"""

import time

import synnax as sy

DEVICE_NAME = "My LabJack T7"
MODEL: sy.labjack.SUPPORTED_MODELS = sy.labjack.T7  # Options: T4, T7, T8
# The serial number on the device label, or None for the first device of MODEL.
SERIAL_NUMBER: int | None = None
TIMEOUT = 30 * sy.TimeSpan.SECOND

client = sy.Synnax()

existing_device = client.devices.retrieve(name=DEVICE_NAME, ignore_not_found=True)
if existing_device is not None:
    print(f"✓ {DEVICE_NAME} is already connected (serial {existing_device.key}).")
    exit(0)


def find() -> sy.Device | None:
    if SERIAL_NUMBER is not None:
        return client.devices.retrieve(key=str(SERIAL_NUMBER), ignore_not_found=True)
    return client.devices.retrieve(model=MODEL, one=True, ignore_not_found=True)


print(f"Waiting up to {TIMEOUT} for the Driver to find the device...")
deadline = sy.TimeStamp.now() + TIMEOUT
device = find()
while device is None and sy.TimeStamp.now() < deadline:
    time.sleep(1)
    device = find()

if device is None:
    print("✗ The Driver did not find the device. Check that:")
    print("  - The Synnax Driver is running on this computer.")
    print("  - The LabJack LJM library is installed.")
    print("  - No other software (such as Kipling) has the device open.")
    exit(1)

client.devices.dispatch(
    device.key,
    [
        sy.device.set(sy.device.SetPayload(path=["name"], value=DEVICE_NAME)),
        sy.device.set(sy.device.SetPayload(path=["configured"], value=True)),
    ],
)
print(f"✓ Connected {DEVICE_NAME} (model {device.model}, serial {device.key}).")
