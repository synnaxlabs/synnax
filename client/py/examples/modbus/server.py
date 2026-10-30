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
from pymodbus.client import ModbusTcpClient
from pymodbus.constants import ExcCodes
from pymodbus.datastore import ModbusServerContext
from pymodbus.exceptions import NoSuchIdException
from pymodbus.server import ModbusTcpServer
from pymodbus.simulator import DataType, SimData, SimDevice
from pymodbus.simulator.simcore import SimCore

import synnax as sy
from examples.simulators.device_sim import DeviceSim
from synnax import modbus

SLAVE_ID = 0x00
SENSOR_COUNT = 5
# From FIXED_START to TYPED_START, register n holds n, and bit n is on when n is odd.
FIXED_START = 100
TYPED_START = 300
GATEWAY_UNITS = (1, 2)
# From TYPED_START: float32 1.5, int16 -2, and uint32 70000, low word first as the
# Driver reads by default.
TYPED_REGISTERS = [0x0000, 0x3FC0, 0xFFFE, 0x1170, 0x0001]


async def updating_writer(
    context: ModbusServerContext | SimCore, rate: sy.Rate = 50 * sy.Rate.HZ
) -> None:
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
    channel_names = (f"{device_name.replace(' ', '_')}_time",)

    async def _run_server(self) -> None:
        await run_server(self.host, self.port, rate=self.rate)

    def _wait_for_ready(self) -> None:
        """Poll until the server answers a coil read."""
        super()._wait_for_ready()
        timer = sy.Timer()
        while True:
            client = ModbusTcpClient(self.host, port=self.port, timeout=1)
            try:
                if (
                    client.connect()
                    and not client.read_coils(0, count=1, device_id=SLAVE_ID).isError()
                ):
                    return
            except Exception:
                pass
            finally:
                client.close()
            if timer.elapsed() > self.startup_timeout:
                raise RuntimeError(
                    f"Modbus server on {self.host}:{self.port} not answering after "
                    f"{self.startup_timeout}"
                )
            sy.sleep(0.1)

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
    fixed = range(FIXED_START, TYPED_START)
    bits = [False] * FIXED_START + [n % 2 == 1 for n in fixed]
    registers = [0] * FIXED_START + list(fixed) + TYPED_REGISTERS
    dev = SimDevice(
        id=SLAVE_ID,
        simdata=(
            [SimData(0, values=bits, datatype=DataType.BITS)],
            [SimData(0, values=bits, datatype=DataType.BITS)],
            [SimData(0, values=registers, datatype=DataType.REGISTERS)],
            [SimData(0, values=registers, datatype=DataType.REGISTERS)],
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
    async with asyncio.TaskGroup() as tg:
        tg.create_task(updating_writer(server.context, rate))
        await server.serve_forever()


class ModbusGatewaySim(DeviceSim):
    """Modbus TCP gateway simulator with units 1 and 2 on port 5022."""

    description = "Modbus TCP gateway simulator on port 5022"
    host = "127.0.0.1"
    port = 5022
    device_name = "Modbus Gateway Unit 1"

    async def _run_server(self) -> None:
        await run_gateway(self.host, self.port)

    @staticmethod
    def create_device(rack_key: int, unit_id: int = 1) -> modbus.Device:
        return modbus.Device(
            host=ModbusGatewaySim.host,
            port=ModbusGatewaySim.port,
            unit_id=unit_id,
            name=f"Modbus Gateway Unit {unit_id}",
            location=f"{ModbusGatewaySim.host}:{ModbusGatewaySim.port}",
            rack=rack_key,
        )


class _Gateway(SimCore):
    """Answers only its own units, and counts its open connections in input register 0
    of each unit. A request for another unit gets no reply, as from a unit that is
    off."""

    connections = 0

    def trace_connect(self, connected: bool) -> None:
        self.connections += 1 if connected else -1

    def _require(self, device_id: int) -> None:
        if device_id not in self.devices:
            raise NoSuchIdException(f"unit {device_id}")

    async def async_getValues(
        self, device_id: int, func_code: int, address: int, count: int = 1
    ) -> list[int] | list[bool] | ExcCodes:
        self._require(device_id)
        if func_code == 4 and address == 0 and count == 1:
            return [self.connections]
        return await super().async_getValues(device_id, func_code, address, count)

    async def async_setValues(
        self,
        device_id: int,
        func_code: int,
        address: int,
        values: list[int] | list[bool],
    ) -> ExcCodes | None:
        self._require(device_id)
        return await super().async_setValues(device_id, func_code, address, values)


async def run_gateway(
    host: str = ModbusGatewaySim.host, port: int = ModbusGatewaySim.port
) -> None:
    """Run a Modbus TCP gateway with units 1 and 2. Holding register n of unit u holds
    100 * u + n, and input register 0 of each unit holds the number of open
    connections."""
    devices = [
        SimDevice(
            id=unit,
            simdata=(
                [SimData(0, values=[False] * 16, datatype=DataType.BITS)],
                [SimData(0, values=[False] * 16, datatype=DataType.BITS)],
                [
                    SimData(
                        0,
                        values=[100 * unit + n for n in range(10)],
                        datatype=DataType.REGISTERS,
                    )
                ],
                [SimData(0, values=[0] * 10, datatype=DataType.REGISTERS)],
            ),
        )
        for unit in GATEWAY_UNITS
    ]
    gateway = _Gateway(devices)
    server = ModbusTcpServer(
        context=devices,
        address=(host, port),
        ignore_missing_devices=True,
        trace_connect=gateway.trace_connect,
    )
    server.context = gateway
    await server.serve_forever()


if __name__ == "__main__":
    asyncio.run(run_server())
