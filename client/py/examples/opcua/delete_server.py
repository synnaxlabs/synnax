#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This script deletes the OPC UA test server device from Synnax.

If the device is not found, the script exits with an appropriate message.
Otherwise, it prompts for confirmation before deleting.

Before running this example:
1. Login to Synnax (if not already logged in):
   uv run sy login

2. Run this script:
   uv run python examples/opcua/delete_server.py

   For the TLS-encrypted server:
   uv run python examples/opcua/delete_server.py --tls

Note: You do NOT need the OPC UA test server running to delete the device.
      This script only removes the device registration from Synnax.
"""

import argparse

import synnax as sy

parser = argparse.ArgumentParser(description="Delete an OPC UA server from Synnax")
parser.add_argument(
    "--tls", action="store_true", help="Delete the TLS-encrypted server"
)
parser.add_argument(
    "--tls-auth",
    action="store_true",
    help="Delete the TLS-encrypted server with username/password (port 4843)",
)
args = parser.parse_args()

if args.tls_auth:
    DEVICE_NAME = "OPC UA TLS Auth Server"
elif args.tls:
    DEVICE_NAME = "OPC UA TLS Server"
else:
    DEVICE_NAME = "OPC UA Server"

# Connect to Synnax
client = sy.Synnax()

print("=" * 70)
print("OPC UA Server Deletion Script")
print("=" * 70)
print(f"Target Device Name: {DEVICE_NAME}")
print()

# Check if device exists
existing_device = client.devices.retrieve(name=DEVICE_NAME, ignore_not_found=True)

if existing_device is None:
    print("✓ Device not found - nothing to delete.")
    print()
    print("The device is already disconnected from Synnax.")
    print("=" * 70)
    exit(0)

# Device found - show info and prompt for confirmation
print("Device found:")
print(f"- Name: {existing_device.name}")
print(f"- Key: {existing_device.key}")
print(f"- Location: {existing_device.location}")
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
    print("\nDeleting device...")

    try:
        client.devices.delete([existing_device.key])

        print("✓ Device deleted successfully!")
        print()
        print("The OPC UA server has been disconnected from Synnax.")
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
