#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Diagnostic script to test Modbus connection and task status.
"""

import synnax as sy

# Connect to Synnax
client = sy.Synnax()

print("=" * 70)
print("Modbus Connection Diagnostic")
print("=" * 70)

# Check if device exists
dev = client.devices.retrieve(name="Modbus Server", ignore_not_found=True)

if dev is None:
    print("✗ Device 'Modbus Server' not found")
    print("  Run connect_modbus_server.py first")
    exit(1)

print(f"✓ Device found: {dev.name}")
print(f"  - Key: {dev.key}")
print(f"  - Location: {dev.location}")
print(f"  - Make: {dev.make}")
print(f"  - Properties: {dev.properties}")
print()

# Check if any tasks exist for this device
print("Checking for tasks...")
try:
    # Retrieve task by name
    task = client.tasks.retrieve(name="Modbus Python Example - Read Task")
    print(f"  ✓ Found task: {task.name}")
    print(f"    - Key: {task.key}")
    print(f"    - Type: {task.type}")
    if task.status:
        print(f"    - Status: {task.status.variant}")
        if task.status.message:
            print(f"    - Message: {task.status.message}")
except Exception as e:
    print(f"  ✗ Task not found or error: {e}")

print()

# Check if channels exist
print("Checking for channels...")
expected_channels = [
    "modbus_time",
    "holding_register_0",
    "holding_register_1",
    "input_register_0",
    "coil_0",
    "discrete_input_0",
]

for ch_name in expected_channels:
    try:
        ch = client.channels.retrieve(ch_name)
        print(f"  ✓ {ch_name} (key={ch.key})")
    except Exception:
        print(f"  ✗ {ch_name} - not found")

print()
print("=" * 70)
print("Diagnostic complete")
print()
print("Next steps:")
print("1. Make sure the Modbus server is running:")
print("   poetry run python driver/modbus/dev/server.py")
print("2. Check the Synnax Console for any error messages")
print("3. Check the driver logs for connection errors")
print("=" * 70)
