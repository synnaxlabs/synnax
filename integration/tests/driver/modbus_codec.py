#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Checks the values Modbus tasks read and write against the simulator.

A pymodbus client lays out and reads back the registers with its own encoding, so each
test pins the register layout of every data type and swap setting.
"""

import struct
from collections.abc import Callable, Mapping
from dataclasses import dataclass

from examples.modbus import ModbusSim
from pymodbus.client import ModbusTcpClient

import synnax as sy
from synnax import modbus
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import cleanup_task, create_channel, create_index

DEVICE_ID = 0
FORMATS = {
    "int16": "h",
    "uint16": "H",
    "int32": "i",
    "uint32": "I",
    "float32": "f",
    "int64": "q",
    "float64": "d",
}
TIMEOUT = 10 * sy.TimeSpan.SECOND


@dataclass
class Register:
    """A value at a holding register address, laid out with a data type and swaps."""

    address: int
    data_type: str
    value: float
    bytes_swapped: bool = False
    words_swapped: bool = False

    @property
    def registers(self) -> list[int]:
        """Returns the registers that hold the value. They hold the low word first
        unless words_swapped, and each register is big-endian unless bytes_swapped."""
        raw = struct.pack(f">{FORMATS[self.data_type]}", self.value)
        words = [raw[i : i + 2] for i in range(0, len(raw), 2)]
        if not self.words_swapped:
            words.reverse()
        if self.bytes_swapped:
            words = [w[::-1] for w in words]
        return [int.from_bytes(w, "big") for w in words]


READ_REGISTERS = [
    Register(40, "int16", -1234),
    Register(41, "uint16", 51234, bytes_swapped=True),
    Register(42, "int32", -123456789),
    Register(44, "uint32", 3_000_000_000, words_swapped=True),
    Register(46, "float32", 3.25),
    Register(48, "float32", -1.5, bytes_swapped=True, words_swapped=True),
    Register(50, "float64", 12345.678),
    Register(54, "int64", -1_234_567_890_123, bytes_swapped=True, words_swapped=True),
]

WRITE_REGISTERS = [
    Register(70, "int16", -4321),
    Register(72, "uint32", 4_000_000_000, words_swapped=True),
    Register(74, "float32", -6.75, bytes_swapped=True),
    Register(76, "float64", -9876.5),
    Register(81, "int32", 987654321, bytes_swapped=True, words_swapped=True),
]
# Addresses inside the write block that no channel writes.
REGISTER_GAPS = {71: 0xBEEF, 80: 0x1234}

# ModbusSim holds coils 0 to 4 at these values.
SIM_COILS = [1, 0, 1, 0, 1]
# Coil 92 is between channels and starts on, so the write must keep it on.
WRITE_COILS = {90: 1, 91: 0, 93: 1}


class ModbusCodec(SimulatorCase):
    """Reads and writes holding registers and coils through Modbus tasks.

    Tests (run sequentially):
        1. Read registers: every data type and swap setting decodes to its value.
        2. Read coils: the simulator's fixed coils decode to 0 and 1.
        3. Write registers: every command lands in the device's layout, and
           registers between channels keep their values.
        4. Write coils: every command lands, and coils between channels keep theirs.
    """

    sim_classes = [ModbusSim]

    def setup(self) -> None:
        self.modbus = ModbusTcpClient(ModbusSim.host, port=ModbusSim.port, timeout=2)
        super().setup()
        self.device = self.client.devices.retrieve(name=self.device_name)
        if not self.modbus.connect():  # type: ignore[no-untyped-call]
            self.fail(f"Could not connect to {ModbusSim.host}:{ModbusSim.port}")

    def teardown(self) -> None:
        try:
            self.modbus.close()  # type: ignore[no-untyped-call]
        finally:
            super().teardown()

    def run(self) -> None:
        self.test_read_registers()
        self.test_read_coils()
        self.test_write_registers()
        self.test_write_coils()

    def test_read_registers(self) -> None:
        self.log("Testing: Read registers")
        for r in READ_REGISTERS:
            self._check(
                self.modbus.write_registers(r.address, r.registers, device_id=DEVICE_ID)
            )
        idx = create_index(self.client, "modbus_codec_read_time")
        channels = [
            modbus.HoldingRegisterReadChannel(
                channel=create_channel(
                    self.client,
                    name=f"modbus_codec_read_{r.address}",
                    data_type=sy.DataType(r.data_type),
                    index=idx.key,
                ),
                address=r.address,
                data_type=r.data_type,
                bytes_swapped=r.bytes_swapped,
                words_swapped=r.words_swapped,
            )
            for r in READ_REGISTERS
        ]
        expected = {ch.channel: r.value for ch, r in zip(channels, READ_REGISTERS)}
        self._assert_read(channels, expected, "Modbus Codec Read Registers")

    def test_read_coils(self) -> None:
        self.log("Testing: Read coils")
        idx = create_index(self.client, "modbus_codec_coil_time")
        channels = [
            modbus.CoilReadChannel(
                channel=create_channel(
                    self.client,
                    name=f"modbus_codec_coil_{address}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=address,
            )
            for address in range(len(SIM_COILS))
        ]
        expected = {ch.channel: v for ch, v in zip(channels, SIM_COILS)}
        self._assert_read(channels, expected, "Modbus Codec Read Coils")

    def test_write_registers(self) -> None:
        self.log("Testing: Write registers")
        start = WRITE_REGISTERS[0].address
        count = WRITE_REGISTERS[-1].address + len(WRITE_REGISTERS[-1].registers) - start
        self._check(
            self.modbus.write_registers(start, [0] * count, device_id=DEVICE_ID)
        )
        for address, value in REGISTER_GAPS.items():
            self._check(
                self.modbus.write_registers(address, [value], device_id=DEVICE_ID)
            )
        idx = create_index(self.client, "modbus_codec_hr_cmd_time")
        channels = [
            modbus.HoldingRegisterWriteChannel(
                channel=create_channel(
                    self.client,
                    name=f"modbus_codec_hr_cmd_{r.address}",
                    data_type=sy.DataType(r.data_type),
                    index=idx.key,
                ),
                address=r.address,
                data_type=r.data_type,
                bytes_swapped=r.bytes_swapped,
                words_swapped=r.words_swapped,
            )
            for r in WRITE_REGISTERS
        ]
        want = [0] * count
        for r in WRITE_REGISTERS:
            want[r.address - start : r.address - start + len(r.registers)] = r.registers
        for address, value in REGISTER_GAPS.items():
            want[address - start] = value

        def device_state() -> list[int]:
            res = self.modbus.read_holding_registers(
                start, count=count, device_id=DEVICE_ID
            )
            self._check(res)
            return list(res.registers)

        commands = {ch.channel: r.value for ch, r in zip(channels, WRITE_REGISTERS)}
        self._assert_written(
            channels, commands, device_state, want, "Modbus Codec Write Registers"
        )

    def test_write_coils(self) -> None:
        self.log("Testing: Write coils")
        start = min(WRITE_COILS)
        count = max(WRITE_COILS) - start + 1
        initial = [not WRITE_COILS.get(start + i, 0) for i in range(count)]
        self._check(self.modbus.write_coils(start, initial, device_id=DEVICE_ID))
        idx = create_index(self.client, "modbus_codec_coil_cmd_time")
        channels = [
            modbus.CoilWriteChannel(
                channel=create_channel(
                    self.client,
                    name=f"modbus_codec_coil_cmd_{address}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=address,
            )
            for address in WRITE_COILS
        ]
        want = [int(WRITE_COILS.get(start + i, initial[i])) for i in range(count)]

        def device_state() -> list[int]:
            res = self.modbus.read_coils(start, count=count, device_id=DEVICE_ID)
            self._check(res)
            return [int(b) for b in res.bits[:count]]

        commands = {ch.channel: WRITE_COILS[ch.address] for ch in channels}
        self._assert_written(
            channels, commands, device_state, want, "Modbus Codec Write Coils"
        )

    def _assert_read(
        self,
        channels: list[modbus.ReadChannel],
        expected: Mapping[int, float],
        name: str,
        samples: int = 5,
    ) -> None:
        """Runs a read task over channels until each has samples values, and asserts
        that every value equals the expected one."""
        task = modbus.ReadTask(
            name=name,
            device=self.device.key,
            sample_rate=20 * sy.Rate.HZ,
            stream_rate=10 * sy.Rate.HZ,
            channels=channels,
        )
        self.client.tasks.configure(task)
        received: dict[int, list[float]] = {key: [] for key in expected}
        try:
            with self.client.open_streamer(list(expected)) as streamer:
                with task.run():
                    timer = sy.Timer()
                    while any(len(v) < samples for v in received.values()):
                        if timer.elapsed() > TIMEOUT:
                            counts = {k: len(v) for k, v in received.items()}
                            self.fail(f"{name}: expected {samples} samples, {counts}")
                        frame = streamer.read(timeout=1)
                        if frame is None:
                            continue
                        for key in expected:
                            if key in frame:
                                received[key].extend(frame[key].to_numpy().tolist())
        finally:
            cleanup_task(self.client, task)
        for ch in channels:
            got = received[ch.channel]
            if any(v != expected[ch.channel] for v in got):
                self.fail(
                    f"{name}: address {ch.address} expected {expected[ch.channel]!r}, "
                    f"got {got}"
                )
        self.log(f"  {len(channels)} channels decoded correctly")

    def _assert_written(
        self,
        channels: list[modbus.WriteChannel],
        commands: Mapping[int, float],
        device_state: Callable[[], list[int]],
        want: list[int],
        name: str,
        attempts: int = 3,
    ) -> None:
        """Runs a write task over channels, sends commands, and asserts that the
        device state becomes want. A write task acknowledges its start before its
        streamer opens, so a command can be lost and is sent again."""
        task = modbus.WriteTask(name=name, device=self.device.key, channels=channels)
        self.client.tasks.configure(task)
        indexes = list(
            {ch.index for ch in self.client.channels.retrieve(list(commands))}
        )
        got: list[int] = []
        try:
            with task.run():
                for _ in range(attempts):
                    with self.client.open_writer(
                        start=sy.TimeStamp.now(),
                        channels=[*commands, *indexes],
                        name=f"{name} commands",
                    ) as writer:
                        now = sy.TimeStamp.now()
                        writer.write({**commands, **{k: now for k in indexes}})
                    timer = sy.Timer()
                    while timer.elapsed() < 3 * sy.TimeSpan.SECOND:
                        got = device_state()
                        if got == want:
                            self.log(f"  {len(channels)} channels encoded correctly")
                            return
                        sy.sleep(0.1)
        finally:
            cleanup_task(self.client, task)
        self.fail(f"{name}: expected device state {want}, got {got}")

    def _check(self, res: object) -> None:
        if getattr(res, "isError", lambda: False)():
            self.fail(f"Modbus request failed: {res}")
