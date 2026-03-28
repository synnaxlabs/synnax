#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This script connects to an HTTP server by registering it as a device in Synnax.

If the server is already connected, the script exits successfully.
Otherwise, it prompts to create a new HTTP server connection automatically.

Before running this example:
1. Start the Synnax Driver (if not already running).

2. Start the mock HTTP server:
   uv run python examples/http_sim/server.py

3. Login to Synnax (if not already logged in):
   uv run sy login

4. Run this script:
   uv run python examples/http_sim/connect_server.py

Configuration:
    Modify the constants below to match your HTTP server.
"""

import synnax as sy

# Configuration
DEVICE_NAME = "HTTP Server"
HOST = "127.0.0.1:8081"

# Connect to Synnax
client = sy.Synnax()

print("=" * 70)
print("HTTP Server Connection Script")
print("=" * 70)
print(f"Target Server Name: {DEVICE_NAME}")
print(f"Host: {HOST}")
print()

# Check if server already exists
existing_device = client.devices.retrieve(name=DEVICE_NAME, ignore_not_found=True)

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
    print("Connecting to HTTP server...")

    try:
        # Get the embedded rack (local driver rack)
        rack = client.racks.retrieve_embedded_rack()
        print(f"Using rack: {rack.name} (key={rack.key})")

        device = sy.http.Device(
            host=HOST,
            secure=False,
            name=DEVICE_NAME,
            rack=rack.key,
            health_check=sy.http.HealthCheck(path="/health"),
        )

        created_device = client.devices.create(device)

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
    print("2. Add a new HTTP server:")
    print(f"   - Name: {DEVICE_NAME}")
    print(f"   - Host: {HOST}")
    print("=" * 70)
    exit(1)
