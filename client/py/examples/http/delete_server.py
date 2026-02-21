#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This script deletes an HTTP mock server from Synnax.

If the server is not found, the script exits with an appropriate message.
Otherwise, it prompts for confirmation before deleting.

Before running this example:
1. Login to Synnax (if not already logged in):
   uv run sy login

2. Run this script:
   uv run python examples/http/delete_server.py

Note: You do NOT need the HTTP mock server running to delete the server registration.
      This script only removes the server registration from Synnax.

Configuration:
    Modify the constants below to match your HTTP server name.
"""

import synnax as sy

DEVICE_NAME = "HTTP Mock Server"

client = sy.Synnax()

print("=" * 70)
print("HTTP Mock Server Deletion Script")
print("=" * 70)
print(f"Target Server Name: {DEVICE_NAME}")
print()

existing_device = client.devices.retrieve(name=DEVICE_NAME, ignore_not_found=True)

if existing_device is None:
    print("✓ Server not found - nothing to delete.")
    print()
    print("The server is already disconnected from Synnax.")
    print("=" * 70)
    exit(0)

print("Server found:")
print(f"- Name: {existing_device.name}")
print(f"- Key: {existing_device.key}")
print(f"- Location: {existing_device.location}")
print()
print("Are you sure you want to delete this server? [Y/N]: ", end="", flush=True)

try:
    response = input().strip().lower()
except (KeyboardInterrupt, EOFError):
    print("\n")
    response = "n"

if response in ("y", "yes"):
    print()
    print("Deleting server...")

    try:
        client.devices.delete([existing_device.key])

        print("✓ Server deleted successfully!")
        print()
        print("The HTTP server has been disconnected from Synnax.")
        print("=" * 70)
        exit(0)

    except Exception as e:
        print(f"✗ Failed to delete server: {e}")
        print()
        print("Please try deleting manually in the Synnax Console:")
        print("1. Open Synnax Console")
        print("2. Go to Resources → Devices")
        print(f"3. Find '{DEVICE_NAME}' and delete it")
        print("=" * 70)
        exit(1)

else:
    print()
    print("Deletion cancelled - server preserved.")
    print("=" * 70)
    exit(0)
