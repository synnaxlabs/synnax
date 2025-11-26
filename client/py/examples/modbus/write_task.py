#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to write commands to the test Modbus TCP server (server.py).

Before running this example:
1. Start the test server:
   python driver/modbus/dev/server.py

2. Connect the Modbus device in Synnax:
   - Host: localhost (127.0.0.1)
   - Port: 5020
   - Name the device "Modbus Server" (or update line 30 below)

3. Note: The test server continuously updates its own values, so writes will be
   visible briefly before being overwritten by the server's simulation.
"""

import time

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Retrieve the Modbus device from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.devices.retrieve(name="Modbus Server")

# Create an index channel for command timestamps
modbus_cmd_time = client.channels.create(
    name="modbus_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create command channels for coils (digital outputs)
coil_cmd_0 = client.channels.create(
    name="coil_cmd_0",
    index=modbus_cmd_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)
coil_cmd_1 = client.channels.create(
    name="coil_cmd_1",
    index=modbus_cmd_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# Create command channels for holding registers (analog outputs)
holding_reg_cmd_0 = client.channels.create(
    name="holding_reg_cmd_0",
    index=modbus_cmd_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)
holding_reg_cmd_1 = client.channels.create(
    name="holding_reg_cmd_1",
    index=modbus_cmd_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# Create the Modbus Write Task
# Demonstrates writing to both coils (1-bit digital) and holding registers (16-bit analog)
tsk = sy.modbus.WriteTask(
    name="Modbus Py - Write Task",
    device=dev.key,
    channels=[
        # Coil outputs (1-bit digital) - addresses 0-1
        sy.modbus.CoilOutputChan(
            channel=coil_cmd_0.key,
            address=0,
        ),
        sy.modbus.CoilOutputChan(
            channel=coil_cmd_1.key,
            address=1,
        ),
        # Holding register outputs (16-bit analog) - addresses 2-3
        sy.modbus.HoldingRegisterOutputChan(
            channel=holding_reg_cmd_0.key,
            address=2,
            data_type="uint8",
        ),
        sy.modbus.HoldingRegisterOutputChan(
            channel=holding_reg_cmd_1.key,
            address=3,
            data_type="uint8",
        ),
    ],
)

# Configure the task with Synnax
client.tasks.configure(tsk)
print("✓ Task configured successfully")

print("=" * 70)
print("Starting Modbus Write Task")
print("=" * 70)
print("Sending commands to server.py (localhost:5020)...")
print("Writing 10 command sequences at 1 Hz (10 seconds)\n")

print(f"{'Cycle':<8} {'Coil 0':>10} {'Coil 1':>10} {'HoldReg 2':>12} {'HoldReg 3':>12}")
print("-" * 70)

# Start the write task
with tsk.run():
    # Open a writer to send commands
    with client.open_writer(
        start=sy.TimeStamp.now(),
        channels=[
            modbus_cmd_time.key,
            coil_cmd_0.key,
            coil_cmd_1.key,
            holding_reg_cmd_0.key,
            holding_reg_cmd_1.key,
        ],
        enable_auto_commit=True,
    ) as writer:
        # Write test commands
        for i in range(10):
            # Digital outputs (coils) - alternating ON/OFF
            coil_0_val = i % 2
            coil_1_val = (i + 1) % 2

            # Analog outputs (holding registers) - cycling values in 0-255 range
            hold_reg_0_val = (i * 25) % 256
            hold_reg_1_val = 255 - (i * 20) % 256

            print(
                f"{i+1:<8} {coil_0_val:>10} {coil_1_val:>10} {hold_reg_0_val:>12} {hold_reg_1_val:>12}"
            )

            # Write all commands with timestamp
            writer.write(
                {
                    modbus_cmd_time.key: sy.TimeStamp.now(),
                    coil_cmd_0.key: coil_0_val,
                    coil_cmd_1.key: coil_1_val,
                    holding_reg_cmd_0.key: hold_reg_0_val,
                    holding_reg_cmd_1.key: hold_reg_1_val,
                }
            )
            writer.commit()
            time.sleep(1)

print("-" * 70)
print("✓ Write task completed successfully!")
print("\nCommands sent:")
print("- Coil 0 (address 0): Alternated between OFF and ON")
print("- Coil 1 (address 1): Alternated between ON and OFF")
print("- Holding Register 2 (address 2): Values 0, 25, 50, ..., 225")
print("- Holding Register 3 (address 3): Values 255, 235, 215, ..., 55")
print("=" * 70)
