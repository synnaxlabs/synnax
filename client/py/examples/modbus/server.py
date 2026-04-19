#  Copyright 2026 Synnax Labs, Inc.
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
from pymodbus.datastore.context import SimDevice
from pymodbus.server import ModbusTcpServer
from pymodbus.simulator.simdata import DataType, SimData

import synnax as sy
from examples.simulators.device_sim import DeviceSim
from synnax import modbus

SLAVE_ID = 0x00
SENSOR_COUNT = 5


async def updating_writer(context, rate: sy.Rate = 50 * sy.Rate.HZ):
    """Update Modbus registers continuously with simulated sensor data."""
    start_ref = time.time()
    i = 0

    while True:
        i += 1
        current_time = time.time()
        elapsed = current_time - start_ref

        base_value = math.sin(elapsed)

        hr_values = []
        for sensor_idx in range(SENSOR_COUNT):
            value = (base_value + sensor_idx) * 25 + 128
            hr_values.append(max(0, min(255, int(value))))

        await context.async_setValues(SLAVE_ID, 3, 0, hr_values)

        ir_values = []
        for sensor_idx in range(SENSOR_COUNT):
            value = (base_value + sensor_idx) * 25 + 128
            ir_values.append(max(0, min(255, int(value))))
        await context.async_setValues(SLAVE_ID, 4, 0, ir_values)

        digital_values = [i % 2 == 0, i % 3 == 0, i % 5 == 0, i % 7 == 0]
        await context.async_setValues(SLAVE_ID, 2, 0, digital_values)

        coil_values = [True, False, True, False, True]
        await context.async_setValues(SLAVE_ID, 1, 0, coil_values)

        await asyncio.sleep(1 / rate)


class ModbusSim(DeviceSim):
    """Modbus TCP device simulator on port 5020."""

    description = "Modbus TCP simulator on port 5020"
    host = "127.0.0.1"
    port = 5020
    device_name = "Modbus TCP Test Server"

    async def _run_server(self) -> None:
        await run_server(self.host, self.port, rate=self.rate)

    @staticmethod
    def create_device(rack_key: int) -> modbus.Device:
        return modbus.Device(
            host=ModbusSim.host,
            port=ModbusSim.port,
            name=ModbusSim.device_name,
            location=f"{ModbusSim.host}:{ModbusSim.port}",
            rack=rack_key,
            swap_bytes=False,
            swap_words=False,
        )


async def run_server(
    host: str = ModbusSim.host,
    port: int = ModbusSim.port,
    rate: sy.Rate = 50 * sy.Rate.HZ,
) -> None:
    """Run the Modbus TCP server."""
    dev = SimDevice(
        id=SLAVE_ID,
        simdata=(
            [SimData(0, values=[False] * 100, datatype=DataType.BITS)],
            [SimData(0, values=[False] * 100, datatype=DataType.BITS)],
            [SimData(0, values=[0] * 100, datatype=DataType.REGISTERS)],
            [SimData(0, values=[0] * 100, datatype=DataType.REGISTERS)],
        ),
    )

    identity = ModbusDeviceIdentification()
    identity.VendorName = "Synnax Labs"
    identity.ProductCode = "MODBUS-SIM"
    identity.VendorUrl = "https://synnaxlabs.com"
    identity.ProductName = "Synnax Modbus Simulator"
    identity.ModelName = "Extended Simulator"
    identity.MajorMinorRevision = "1.0.0"

    server = ModbusTcpServer(context=dev, identity=identity, address=(host, port))
    asyncio.create_task(updating_writer(server.context, rate))
    await server.serve_forever()


if __name__ == "__main__":
    asyncio.run(run_server())
