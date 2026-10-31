#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.modbus.server import STRING_START

import synnax as sy
from tests.driver.modbus_task import ModbusReadTaskCase
from tests.driver.task import assert_streamed_values, create_channel, create_index


class ModbusReadInputRegister(ModbusReadTaskCase):
    task_name = "Modbus Read Input Register"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.ReadChannel]:
        idx = create_index(client, "input_register_index")
        return [
            sy.modbus.InputRegisterReadChannel(
                channel=create_channel(
                    client,
                    name=f"input_register_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
                data_type=sy.DataType.UINT8,
            )
            for i in range(2)
        ]


class ModbusReadHoldingRegister(ModbusReadTaskCase):
    task_name = "Modbus Read Holding Register"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.ReadChannel]:
        idx = create_index(client, "holding_register_index")
        return [
            sy.modbus.HoldingRegisterReadChannel(
                channel=create_channel(
                    client,
                    name=f"holding_register_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
                data_type=sy.DataType.UINT8,
            )
            for i in range(2)
        ]


class ModbusReadDiscreteInput(ModbusReadTaskCase):
    task_name = "Modbus Read Discrete Input"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.ReadChannel]:
        idx = create_index(client, "discrete_input_index")
        return [
            sy.modbus.DiscreteInputReadChannel(
                channel=create_channel(
                    client,
                    name=f"discrete_input_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
            )
            for i in range(2)
        ]


class ModbusReadCoil(ModbusReadTaskCase):
    task_name = "Modbus Read Coil"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.ReadChannel]:
        idx = create_index(client, "coil_input_index")
        return [
            sy.modbus.CoilReadChannel(
                channel=create_channel(
                    client,
                    name=f"coil_input_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
            )
            for i in range(2)
        ]


class ModbusReadMixed(ModbusReadTaskCase):
    task_name = "Modbus Read Mixed"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.ReadChannel]:
        idx = create_index(client, "modbus_mixed_index")
        registers = [
            sy.modbus.InputRegisterReadChannel(
                channel=create_channel(
                    client,
                    name=f"input_register_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
                data_type=sy.DataType.UINT8,
            )
            for i in range(2)
        ]
        inputs = [
            sy.modbus.DiscreteInputReadChannel(
                channel=create_channel(
                    client,
                    name=f"discrete_input_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
            )
            for i in range(2)
        ]
        return [*registers, *inputs]


UINT8 = sy.DataType.UINT8
UINT16 = sy.DataType.UINT16
HR = sy.modbus.HoldingRegisterReadChannel
IR = sy.modbus.InputRegisterReadChannel
COIL = sy.modbus.CoilReadChannel
DI = sy.modbus.DiscreteInputReadChannel
HR_BIT = sy.modbus.HoldingRegisterBitReadChannel
IR_BIT = sy.modbus.InputRegisterBitReadChannel

# The simulator's fixed values make these known. Each register entry is (name, channel
# type, address, register data type, channel data type, value).
REGISTER_VALUES: list[
    tuple[str, type[HR] | type[IR], int, sy.DataType, sy.DataType, float]
] = [
    # 100 and 299 are 200 registers apart, more than one request can read.
    ("hr_100", HR, 100, UINT16, UINT16, 100),
    ("hr_299", HR, 299, UINT16, UINT16, 299),
    ("ir_100", IR, 100, UINT16, UINT16, 100),
    ("ir_299", IR, 299, UINT16, UINT16, 299),
    ("hr_float32", HR, 300, sy.DataType.FLOAT32, sy.DataType.FLOAT32, 1.5),
    ("hr_float32_high", HR, 301, UINT16, UINT16, 0x3FC0),
    ("hr_int16", HR, 302, sy.DataType.INT16, sy.DataType.INT16, -2),
    ("hr_uint32", HR, 303, sy.DataType.UINT32, sy.DataType.UINT32, 70000),
    ("hr_uint16_as_uint8", HR, 150, UINT16, UINT8, 150),
]
# (name, channel type, address, value).
BIT_VALUES: list[tuple[str, type[COIL] | type[DI], int, float]] = [
    ("coil_111", COIL, 111, 1),
    ("coil_112", COIL, 112, 0),
    ("di_101", DI, 101, 1),
    ("di_102", DI, 102, 0),
]
# (name, channel type, address, bit, value). Register 165 holds 0b10100101.
REGISTER_BIT_VALUES: list[tuple[str, type[HR_BIT] | type[IR_BIT], int, int, float]] = [
    ("hr_165_bit_0", HR_BIT, 165, 0, 1),
    ("hr_165_bit_1", HR_BIT, 165, 1, 0),
    ("hr_256_bit_8", HR_BIT, 256, 8, 1),
    ("ir_165_bit_7", IR_BIT, 165, 7, 1),
    ("ir_165_bit_6", IR_BIT, 165, 6, 0),
]


class ModbusReadValues(ModbusReadTaskCase):
    """Reads the simulator's fixed values: a map wider than one request, each register
    data type, overlapping registers, a register cast into a narrower channel, and
    register bits."""

    task_name = "Modbus Read Values"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.ReadChannel]:
        idx = create_index(client, "modbus_values_index")

        def create(name: str, data_type: sy.DataType) -> int:
            return create_channel(
                client, name=f"modbus_value_{name}", data_type=data_type, index=idx.key
            )

        registers: list[sy.modbus.ReadChannel] = [
            cls(channel=create(name, ch_type), address=address, data_type=reg_type)
            for name, cls, address, reg_type, ch_type, _ in REGISTER_VALUES
        ]
        bits: list[sy.modbus.ReadChannel] = [
            cls(channel=create(name, UINT8), address=address)
            for name, cls, address, _ in BIT_VALUES
        ]
        register_bits: list[sy.modbus.ReadChannel] = [
            cls(channel=create(name, UINT8), address=address, bit=bit)
            for name, cls, address, bit, _ in REGISTER_BIT_VALUES
        ]
        return registers + bits + register_bits

    def run(self) -> None:
        assert self.tsk is not None
        by_name = {f"modbus_value_{v[0]}": v[-1] for v in REGISTER_VALUES}
        by_name.update({f"modbus_value_{v[0]}": v[-1] for v in BIT_VALUES})
        by_name.update({f"modbus_value_{v[0]}": v[-1] for v in REGISTER_BIT_VALUES})
        channels = self.client.channels.retrieve(self._channel_keys(self.tsk))
        expected = {ch.key: by_name[ch.name] for ch in channels}
        with self.client.open_streamer(list(expected)) as streamer, self.tsk.run():
            assert_streamed_values(
                self.client, streamer, expected=expected, task_name=self.task_name
            )


# (name, string length, bytes swapped, value), all from STRING_START.
STRING_VALUES: list[tuple[str, int, bool, str]] = [
    # The NUL ends the string before its length.
    ("string_nul", 8, False, "SYNNAX"),
    ("string_odd", 5, False, "SYNNA"),
    ("string_swapped", 6, True, "YSNNXA"),
]


class ModbusReadString(ModbusReadTaskCase):
    """Reads strings from holding registers."""

    task_name = "Modbus Read String"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.ReadChannel]:
        idx = create_index(client, "modbus_string_index")
        return [
            HR(
                channel=create_channel(
                    client,
                    name=f"modbus_{name}",
                    data_type=sy.DataType.STRING,
                    index=idx.key,
                ),
                address=STRING_START,
                data_type=sy.DataType.STRING,
                string_length=length,
                bytes_swapped=swapped,
            )
            for name, length, swapped, _ in STRING_VALUES
        ]

    def run(self) -> None:
        assert self.tsk is not None
        by_name = {f"modbus_{v[0]}": v[-1] for v in STRING_VALUES}
        channels = self.client.channels.retrieve(self._channel_keys(self.tsk))
        expected = {ch.key: by_name[ch.name] for ch in channels}
        received: dict[int, object] = {}
        timer = sy.Timer()
        with self.client.open_streamer(list(expected)) as streamer, self.tsk.run():
            while received.keys() != expected.keys():
                assert timer.elapsed() < 10 * sy.TimeSpan.SECOND, (
                    f"Strings should stream within 10s, got {received}"
                )
                frame = streamer.read(timeout=1)
                if frame is None:
                    continue
                for key in expected:
                    if key in frame and len(frame[key]) > 0:
                        received[key] = frame[key][-1]
        assert received == expected, f"Strings should be {expected}, got {received}"
