#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import math
import time

from pymodbus import ModbusDeviceIdentification
from pysy.modbus.datastore import (
    ModbusDeviceContext,
    ModbusSequentialDataBlock,
    ModbusServerContext,
)
from pysy.modbus.server import StartAsyncTcpServer


async def updating_writer(context):
    """Update Modbus registers continuously with simulated sensor data."""
    slave_id = 0x00
    start_ref = time.time()
    i = 0
    RATE = 50  # Hz
    SENSOR_COUNT = 5

    while True:
        i += 1
        current_time = time.time()
        elapsed = current_time - start_ref

        # Generate base sine wave value (same phase for all sensors)
        base_value = math.sin(elapsed)

        # Create offset sine waves (same phase, different offsets)
        # Store as holding registers (function code 3)
        # Scale for UInt8 (0-255 range): sine goes -1 to +1, scaled to fit with offsets
        hr_values = []
        for sensor_idx in range(SENSOR_COUNT):
            # Add offset to base sine wave: base_value + sensor_idx
            # Scale to 0-255 range for UInt8 interpretation
            value = (base_value + sensor_idx) * 25 + 128
            # Clamp to 0-255 and store as 16-bit register
            hr_values.append(max(0, min(255, int(value))))

        # Write to holding registers starting at address 0
        context[slave_id].setValues(3, 0, hr_values)

        # Input registers (function code 4) - similar offset pattern
        ir_values = []
        for sensor_idx in range(SENSOR_COUNT):
            value = (base_value + sensor_idx) * 25 + 128
            ir_values.append(max(0, min(255, int(value))))
        context[slave_id].setValues(4, 0, ir_values)

        # Discrete inputs (function code 2) - simulating digital sensors
        digital_values = [i % 2 == 0, i % 3 == 0, i % 5 == 0, i % 7 == 0]
        context[slave_id].setValues(2, 0, digital_values)

        # Coils (function code 1) - simulating writable digital outputs
        coil_values = [True, False, True, False, True]
        context[slave_id].setValues(1, 0, coil_values)

        await asyncio.sleep(1 / RATE)


async def run_server():
    """Run the Modbus TCP server."""
    # Initialize data store
    store = ModbusDeviceContext(
        di=ModbusSequentialDataBlock(0, [0] * 100),  # Discrete Inputs
        co=ModbusSequentialDataBlock(0, [0] * 100),  # Coils
        hr=ModbusSequentialDataBlock(0, [0] * 100),  # Holding Registers
        ir=ModbusSequentialDataBlock(0, [0] * 100),  # Input Registers
    )

    context = ModbusServerContext(devices=store, single=True)

    # Server identification
    identity = ModbusDeviceIdentification()
    identity.VendorName = "Synnax Labs"
    identity.ProductCode = "MODBUS-SIM"
    identity.VendorUrl = "https://synnaxlabs.com"
    identity.ProductName = "Synnax Modbus Simulator"
    identity.ModelName = "Extended Simulator"
    identity.MajorMinorRevision = "1.0.0"

    # Start the updating task
    task = asyncio.create_task(updating_writer(context))

    # Start Modbus TCP server on localhost:5020
    await StartAsyncTcpServer(
        context=context, identity=identity, address=("127.0.0.1", 5020)
    )


if __name__ == "__main__":
    asyncio.run(run_server())
