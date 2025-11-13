#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This script connects to the Modbus TCP test server by registering it in Synnax.

If the server is already connected, the script exits successfully.
Otherwise, it prompts to create a new Modbus server connection automatically.

Before running this example:
1. Start the Synnax Driver (if not already running).

2. Start the Modbus TCP test server:
   python driver/modbus/dev/server.py

3. Login to Synnax (if not already logged in):
   poetry run sy login

4. Run this script:
   poetry run python examples/modbus/connect_modbus_server.py

Configuration:
    Modify the constants below to match your Modbus server configuration.
"""

import json

import synnax as sy
from synnax.hardware import modbus

# Configuration
DEVICE_NAME = "Modbus Server"
HOST = "127.0.0.1"
PORT = 5020

# Connect to Synnax
client = sy.Synnax()

print("=" * 70)
print("Modbus TCP Server Connection Script")
print("=" * 70)
print(f"Target Server Name: {DEVICE_NAME}")
print(f"Host: {HOST}")
print(f"Port: {PORT}")
print()

# Check if server already exists
existing_device = client.hardware.devices.retrieve(
    name=DEVICE_NAME, ignore_not_found=True
)

if existing_device is not None:
    print("✓ Server already connected!")
    print(f"- Name: {existing_device.name}")
    print(f"- Key: {existing_device.key}")
    print(f"- Location: {existing_device.location}")
    print()
    print("No action needed - server is ready to use.")
    print("=" * 70)
    exit(0)

# Server not found - prompt user for connection method
print("Server not found")
print()
print("Would you like to connect automatically? [Y/N]: ", end="", flush=True)

try:
    response = input().strip().lower()
except (KeyboardInterrupt, EOFError):
    print("\n")
    response = "n"

if response in ("", "y", "yes"):
    print()
    print("Connecting to Modbus TCP server...")

    try:
        # Get the embedded rack (local driver rack)
        rack = client.hardware.racks.retrieve_embedded_rack()
        print(f"Using rack: {rack.name} (key={rack.key})")

        device = modbus.Device(
            host=HOST,
            port=PORT,
            name=DEVICE_NAME,
            location=f"{HOST}:{PORT}",
            rack=rack.key,
            swap_bytes=False,
            swap_words=False,
        )

        created_device = client.hardware.devices.create(device)

        print("✓ Server connected successfully!")
        print(f"  - Name: {created_device.name}")
        print(f"  - Key: {created_device.key}")
        print(f"  - Location: {created_device.location}")
        print(f"  - Rack: {rack.name}")
        print()
        print("Server is ready to use.")
        print("=" * 70)
        exit(0)

    except Exception as e:
        print(f"✗ Failed to connect server: {e}")
        print()
        print("Please try connecting manually (see instructions below).")
        response = "n"  # Fall through to manual instructions

if response in ("n", "no") or response not in ("", "y", "yes"):
    # Manual connection instructions
    print()
    print("To connect manually:")
    print("1. Open the Synnax Console")
    print("2. Add a new Modbus server:")
    print(f"   - Name: {DEVICE_NAME}")
    print(f"   - Host: {HOST}")
    print(f"   - Port: {PORT}")
    print("=" * 70)
    exit(1)
