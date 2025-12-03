#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This script deletes a LabJack device from Synnax.

If the device is not found, the script exits with an appropriate message.
Otherwise, it prompts for confirmation before deleting.

Before running this example:
1. Start the Synnax Driver (if not already running).

2. Login to Synnax (if not already logged in):
   poetry run sy login

3. Run this script:
   poetry run python examples/labjack/delete_device.py

Note: You do NOT need the LabJack device connected to delete the device registration.
      This script only removes the device registration from Synnax.

Configuration:
    Modify the constants below to match your LabJack device name.
"""

import synnax as sy

# Configuration
DEVICE_NAME = "My LabJack T7"

# Connect to Synnax
client = sy.Synnax()

print("=" * 70)
print("LabJack Device Deletion Script")
print("=" * 70)
print(f"Target Device Name: {DEVICE_NAME}")
print()

# Check if device exists
existing_device = client.hardware.devices.retrieve(
    name=DEVICE_NAME, ignore_not_found=True
)

if existing_device is None:
    print("✓ Device not found - nothing to delete.")
    print()
    print("The device is already disconnected from Synnax.")
    print("=" * 70)
    exit(0)

# Device found - show info and prompt for confirmation
print("Device found:")
print(f"- Name: {existing_device.name}")
print(f"- Model: {existing_device.model}")
print(f"- Key: {existing_device.key}")
print(f"- Location: {existing_device.location}")
print()
print("⚠ Warning: This will also delete all tasks associated with this device!")
print()
print("Are you sure you want to delete this device? [Y/N]: ", end="", flush=True)

try:
    response = input().strip().lower()
except (KeyboardInterrupt, EOFError):
    print("\n")
    response = "n"

if response in ("y", "yes"):
    # Delete the device
    print()
    print("Deleting device...")

    try:
        client.hardware.devices.delete([existing_device.key])

        print("✓ Device deleted successfully!")
        print()
        print("The LabJack device has been disconnected from Synnax.")
        print("All associated tasks have been removed.")
        print()
        print("To reconnect, run: poetry run python examples/labjack/connect_device.py")
        print("=" * 70)
        exit(0)

    except Exception as e:
        print(f"✗ Failed to delete device: {e}")
        print()
        print("Please try deleting manually in the Synnax Console:")
        print("1. Open Synnax Console")
        print("2. Go to Resources → Devices")
        print(f"3. Find '{DEVICE_NAME}' and delete it")
        print("=" * 70)
        exit(1)

else:
    # User cancelled
    print()
    print("Deletion cancelled - device preserved.")
    print("=" * 70)
    exit(0)
