#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This script connects a LabJack device by registering it in Synnax.

If the device is already connected, the script exits successfully.
Otherwise, it prompts to create a new LabJack device connection automatically.

Before running this example:
1. Start the Synnax Driver (if not already running).

2. Connect your LabJack device via USB, Ethernet, or WiFi.

3. Login to Synnax (if not already logged in):
   poetry run sy login

4. Run this script:
   poetry run python examples/labjack/connect_device.py

Configuration:
    Modify the constants below to match your LabJack device configuration.
"""

import synnax as sy

# Configuration
DEVICE_NAME = "My LabJack T7"
MODEL = sy.labjack.T7  # Options: T4, T7, T8
IDENTIFIER = "ANY"  # Options: serial number, IP address, device name, or "ANY"
CONNECTION_TYPE = "ANY"  # Options: "ANY", "USB", "TCP", "ETHERNET", "WIFI"

# Connect to Synnax
client = sy.Synnax()

print("=" * 70)
print("LabJack Device Connection Script")
print("=" * 70)
print(f"Target Device Name: {DEVICE_NAME}")
print(f"Model: {MODEL}")
print(f"Identifier: {IDENTIFIER}")
print(f"Connection Type: {CONNECTION_TYPE}")
print()

# Check if device already exists
existing_device = client.devices.retrieve(name=DEVICE_NAME, ignore_not_found=True)

if existing_device is not None:
    print("✓ Device already connected!")
    print(f"- Name: {existing_device.name}")
    print(f"- Model: {existing_device.model}")
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
    print("Connecting to LabJack device...")

    try:
        # Get the embedded rack (local driver rack)
        rack = client.racks.retrieve_embedded_rack()
        print(f"Using rack: {rack.name} (key={rack.key})")

        # Determine location string based on connection type
        if IDENTIFIER == "ANY":
            location = CONNECTION_TYPE
        else:
            location = IDENTIFIER

        device = sy.labjack.Device(
            model=MODEL,
            identifier=IDENTIFIER,
            name=DEVICE_NAME,
            location=location,
            rack=rack.key,
            connection_type=CONNECTION_TYPE,
        )

        created_device = client.devices.create(device)

        print("✓ Device connected successfully!")
        print(f"  - Name: {created_device.name}")
        print(f"  - Model: {created_device.model}")
        print(f"  - Key: {created_device.key}")
        print(f"  - Location: {created_device.location}")
        print(f"  - Rack: {rack.name}")
        print()
        print("Device is ready to use.")
        print()
        print("Next steps:")
        print("  - Run examples/labjack/read_task.py to read data")
        print("  - Run examples/labjack/write_task.py to control outputs")
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
    print("1. Open Synnax Console")
    print("2. Go to Resources → Devices")
    print("3. Add a new LabJack device:")
    print(f"   - Name: {DEVICE_NAME}")
    print(f"   - Model: {MODEL}")
    print(f"   - Identifier: {IDENTIFIER}")
    print(f"   - Connection Type: {CONNECTION_TYPE}")
    print()
    print("Identifier options:")
    print("  - 'ANY': Connect to any available device")
    print("  - Serial number (e.g., '470012345')")
    print("  - IP address (e.g., '192.168.1.100')")
    print("  - Device name (e.g., 'MyLabJack')")
    print()
    print("Connection Type options:")
    print("  - 'ANY': Use any available connection")
    print("  - 'USB': Connect via USB only")
    print("  - 'TCP' or 'ETHERNET': Connect via Ethernet")
    print("  - 'WIFI': Connect via WiFi")
    print("=" * 70)
    exit(1)
