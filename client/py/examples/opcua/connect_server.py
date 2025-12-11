#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This script connects to the OPC UA test server by registering it as a device in Synnax.

If the device is already connected, the script exits successfully.
Otherwise, it prompts to create a new OPC UA device connection automatically.

Before running this example:
1. Start the Synnax Driver (if not already running).

2. Start the OPC UA test server:
   poetry run python driver/opc/dev/server_extended.py

3. Login to Synnax (if not already logged in):
   poetry run sy login

4. Run this script:
   poetry run python examples/opcua/connect_opc_server.py

Configuration:
    Modify the constants below to match your OPC UA server configuration.
"""

import json
from uuid import uuid4

import synnax as sy

# Configuration
DEVICE_NAME = "OPC UA Server"
ENDPOINT = "opc.tcp://127.0.0.1:4841/"

# Connect to Synnax
client = sy.Synnax()

print("=" * 70)
print("OPC UA Server Connection Script")
print("=" * 70)
print(f"Target Device Name: {DEVICE_NAME}")
print(f"Endpoint: {ENDPOINT}")
print()

# Check if device already exists
existing_device = client.devices.retrieve(name=DEVICE_NAME, ignore_not_found=True)

if existing_device is not None:
    print("✓ Device already connected!")
    print(f"- Name: {existing_device.name}")
    print(f"- Key: {existing_device.key}")
    print(f"- Location: {existing_device.location}")
    print()
    print("No action needed - device is ready to use.")
    print("=" * 70)
    exit(0)

# Device not found - prompt user for connection method
print("Device not found")
print()
print("Would you like to connect automatically? [Y/N]: ", end="", flush=True)

try:
    response = input().strip().lower()
except (KeyboardInterrupt, EOFError):
    print("\n")
    response = "n"

if response in ("", "y", "yes"):
    print()
    print("Connecting to OPC UA server...")

    try:
        # Get the embedded rack (local driver rack)
        rack = client.racks.retrieve_embedded_rack()
        print(f"Using rack: {rack.name} (key={rack.key})")

        # Create the device with proper connection properties
        device = sy.opcua.Device(
            endpoint=ENDPOINT,
            name=DEVICE_NAME,
            location=ENDPOINT,
            rack=rack.key,
        )

        created_device = client.devices.create(device)

        print("✓ Device connected successfully!")
        print(f"  - Name: {created_device.name}")
        print(f"  - Key: {created_device.key}")
        print(f"  - Location: {created_device.location}")
        print(f"  - Rack: {rack.name}")
        print()
        print("Device is ready to use.")
        print("=" * 70)
        exit(0)

    except Exception as e:
        print(f"✗ Failed to connect device: {e}")
        print()
        print("Please try connecting manually (see instructions below).")
        response = "n"  # Fall through to manual instructions

if response in ("n", "no") or response not in ("", "y", "yes"):
    # Manual connection instructions
    print()
    print("To connect manually:")
    print("1. Open the Synnax Console")
    print("2. Add a new OPC UA device:")
    print(f"   - Name: {DEVICE_NAME}")
    print(f"   - Endpoint: {ENDPOINT}")
    print("   - Make: opc")
    print("=" * 70)
    exit(1)
