#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates task management features in Synnax.

The principles shown here apply to ALL hardware task types (NI, LabJack, OPC UA, Modbus, etc.),
but this example uses Modbus for demonstration purposes.

Key features demonstrated:
- Creating tasks
- Retrieving tasks by name
- Listing all tasks on a rack
- Copying tasks
- Modifying task configurations
- Starting/stopping tasks

Before running this example:
1. Start the Modbus simulator server:
   poetry run python driver/modbus/dev/server.py

2. Connect the Modbus device in Synnax Console:
   - Host: localhost (127.0.0.1)
   - Port: 5020
   - Name: "Modbus Server"

Note: The task management API is identical for all hardware types - only the task
configuration specifics differ between hardware integrations.
"""

import synnax as sy
from synnax import modbus

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Retrieve the Modbus device from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.devices.retrieve(name="Modbus Server")

print("=" * 70)
print("Modbus Task Management Example")
print()

# ============================================================================
# Step 1: Create channels and an initial Modbus read task
# ============================================================================
print("Step 1: Creating initial Modbus read task")
print("-" * 70)

# Create an index channel for timestamps
modbus_time = client.channels.create(
    name="modbus_mgmt_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create data channels for two input registers
input_reg_0 = client.channels.create(
    name="modbus_mgmt_reg_0",
    index=modbus_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

input_reg_1 = client.channels.create(
    name="modbus_mgmt_reg_1",
    index=modbus_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# Create the initial Modbus read task
original_task = sy.modbus.ReadTask(
    name="Modbus Read Task",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 10,
    stream_rate=sy.Rate.HZ * 10,
    data_saving=True,
    channels=[
        sy.modbus.InputRegisterChan(
            channel=input_reg_0.key, address=0, data_type="uint8"
        ),
        sy.modbus.InputRegisterChan(
            channel=input_reg_1.key, address=1, data_type="uint8"
        ),
    ],
)

# Configure the task with Synnax
client.tasks.configure(original_task)
print(f"✓ Created task: '{original_task.name}'")
print(f"  Task key: {original_task.key}")
print(f"  Sample rate: 10 Hz")
print(f"  Stream rate: 10 Hz")
print(f"  Channels: 2 input registers")
print()

# ============================================================================
# Step 2: Copy the task and modify its configuration
# ============================================================================
print("\nStep 2: Copying task and modifying configuration")
print("-" * 70)

# Retrieve the original task by name
retrieved_task = client.tasks.retrieve(name="Modbus Read Task")
print(f"\nRetrieved task by name: '{retrieved_task.name}' (key: {retrieved_task.key})")

# Copy the original task with a new name
# This creates a new, independent task with the same configuration
copied_task_raw = client.tasks.copy(
    key=retrieved_task.key,
    name="Modbus Read Task Copy",
)

# Convert the raw task to a ReadTask object so we can modify the configuration
copied_task = sy.modbus.ReadTask(internal=copied_task_raw)

# Modify the stream rate and enable auto-start
# Convert Rate to int to avoid Pydantic serialization warnings
copied_task.config.stream_rate = int(sy.Rate.HZ * 5)
copied_task.config.auto_start = True

# Reconfigure the task with the new settings
# This will also start the task automatically since auto_start is True
client.tasks.configure(copied_task)

print(f"\n✓ Copied and modified task: '{copied_task.name}'")
print(f"  Original task:")
print(f"    Key:          {original_task.key}")
print(f"    Stream rate:  10 Hz")
print(f"    Auto-start:   Disabled")
print(f"  Copied task:")
print(f"    Key:          {copied_task.key}")
print(f"    Stream rate:  5 Hz (modified)")
print(f"    Auto-start:   Enabled (task is now running)")

# Keep the copied task running for now - we'll verify it below
print()

# ============================================================================
# Step 3: List all tasks to see the original and copy
# ============================================================================
print("\nStep 3: Listing all tasks on the rack")
print("-" * 70)

# List all tasks again to see the original and copy
all_tasks_updated = client.tasks.list()
print(f"✓ Found {len(all_tasks_updated)} task(s) on the rack:\n")

# Group tasks by type for better visualization
modbus_tasks = [t for t in all_tasks_updated if t.type == "modbus_read"]
other_tasks = [t for t in all_tasks_updated if t.type != "modbus_read"]

if modbus_tasks:
    print("  Modbus Read Tasks:")
    for task in modbus_tasks:
        print(f"    • {task.name}")
        print(f"      Key: {task.key}")

if other_tasks:
    print("\n  Other Tasks:")
    for task in other_tasks:
        print(f"    • {task.name}")
        print(f"      Key: {task.key}, Type: {task.type}")

print()

# Verify the copied task is running by reading some data
print("  Verifying copied task is running (5 second test)...")
with client.open_streamer([input_reg_0.key, input_reg_1.key]) as streamer:
    start_time = sy.TimeStamp.now()
    sample_count = 0
    while sy.TimeStamp.now().span(start_time).seconds < 5:
        frame = streamer.read()
        if frame and input_reg_0.key in frame:
            sample_count += len(frame[input_reg_0.key])

if sample_count > 0:
    print(f"  ✓ Copied task is running (received {sample_count} samples)")
else:
    print(f"  ✗ No samples received from copied task")

# Stop the copied task to clean up
print("  Stopping copied task...")
copied_task.stop()
print(f"  ✓ Copied task stopped")

# ============================================================================
# Step 4: Delete the copied task
# ============================================================================
print("\nStep 4: Deleting the copied task")
print("-" * 70)

# Delete the copied task by key
client.tasks.delete(copied_task.key)
print(f"✓ Deleted copied task (key: {copied_task.key})")

# Verify deletion by listing tasks again
remaining_tasks = client.tasks.list()
remaining_modbus = [t for t in remaining_tasks if t.type == "modbus_read"]
print(f"✓ Remaining Modbus tasks: {len(remaining_modbus)}")
for task in remaining_modbus:
    print(f"  • {task.name} (key: {task.key})")

print("\n✓ Example completed successfully!")
print("=" * 70)
