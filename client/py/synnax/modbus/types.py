#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal, get_args
from uuid import uuid4

from pydantic import JsonValue

from synnax import device, task
from synnax.exceptions import PathError, ValidationError
from synnax.modbus.types_gen import (
    ReadChannel,
    ReadConfig,
    RegisterBit,
    WriteChannel,
    WriteConfig,
)
from synnax.telem import CrudeRate, Rate

# Device identifiers - must match Console expectations
MAKE = "Modbus"
MODEL = "Modbus"

# Device map keys keep the released type spellings, so channels created before the
# labels were renamed keep matching.
_READ_NAME_TYPES = {
    "coil": "coil_input",
    "discrete_input": "discrete_input",
    "holding_register": "holding_register_input",
    "input_register": "register_input",
    "holding_register_bit": "holding_register_bit_input",
    "input_register_bit": "register_bit_input",
}

_WRITE_NAME_TYPES = {
    "coil": "coil_output",
    "holding_register": "holding_register_output",
    "holding_register_bit": "holding_register_bit_output",
}

# The rates libmodbus sets on Linux, macOS, and Windows. The Driver and the Console
# accept the same list.
_BaudRate = Literal[1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200, 230400]
_BAUD_RATES: tuple[int, ...] = get_args(_BaudRate)


class ReadTask(
    task.StarterStopperMixin, task.JSONConfigMixin[ReadConfig], task.Protocol
):
    """A read task for sampling data from Modbus TCP devices and writing the data to a
    Synnax cluster. For detailed information on configuring and operating a Modbus read
    task, see https://docs.synnaxlabs.com/reference/hardware/modbus/read-task

    :param device: The key of the Synnax Modbus device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the Modbus device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The input channels to acquire data from (ReadChannel variants:
        CoilReadChannel, DiscreteInputReadChannel, HoldingRegisterReadChannel,
        InputRegisterReadChannel, HoldingRegisterBitReadChannel,
        InputRegisterBitReadChannel).
    """

    TYPE = "modbus_read"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        sample_rate: CrudeRate = 10,
        stream_rate: CrudeRate = 5,
        data_saving_disabled: bool = False,
        auto_start: bool = False,
        channels: list[ReadChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = ReadConfig(
            device=device,
            sample_rate=Rate(sample_rate),
            stream_rate=Rate(stream_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration.

        The key format follows Console's readMapKey convention:
        "{type}-{address}" for fixed-density channels, "{type}-{address}-{dataType}"
        for register channels, and "{type}-{address}-{bit}" for register bit
        channels, with underscores replaced by hyphens.
        """
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties)
        if "read" not in props:
            props["read"] = {"index": 0, "channels": {}}
        channels = device.ensure_object(props, "read", "channels")
        for ch in self.config.channels:
            key = f"{_READ_NAME_TYPES[ch.type]}-{ch.address}"
            if hasattr(ch, "data_type"):
                key += f"-{ch.data_type}"
            if isinstance(ch, RegisterBit):
                key += f"-{ch.bit}"
            key = key.replace("_", "-")
            channels[key] = ch.channel
        dev.properties = props
        return device_client.create(dev)


class WriteTask(
    task.StarterStopperMixin, task.JSONConfigMixin[WriteConfig], task.Protocol
):
    """A write task for sending commands to Modbus TCP devices. For detailed
    information on configuring and operating a Modbus write task, see
    https://docs.synnaxlabs.com/reference/hardware/modbus/write-task

    :param device: The key of the Synnax Modbus device to write to.
    :param name: A human-readable name for the task.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The output channels to write to (WriteChannel variants:
        CoilWriteChannel, HoldingRegisterWriteChannel,
        HoldingRegisterBitWriteChannel).
    """

    TYPE = "modbus_write"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        auto_start: bool = False,
        channels: list[WriteChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            device=device,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration.

        The key format follows Console's writeMapKey convention: "{type}-{address}"
        for coils, "{type}-{address}-{dataType}" for holding registers, and
        "{type}-{address}-{bit}" for register bits, with underscores replaced by
        hyphens.
        """
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties)
        channels = device.ensure_object(props, "write", "channels")
        for ch in self.config.channels:
            key = f"{_WRITE_NAME_TYPES[ch.type]}-{ch.address}"
            if hasattr(ch, "data_type"):
                key += f"-{ch.data_type}"
            if isinstance(ch, RegisterBit):
                key += f"-{ch.bit}"
            channels[key.replace("_", "-")] = ch.channel
        dev.properties = props
        return device_client.create(dev)


class Device(device.Device):
    """A unit on a Modbus TCP server or a Modbus RTU serial line. Devices with the
    same host and port, or the same serial port, share one connection.

    :param transport: "tcp" for Modbus TCP, or "rtu" for Modbus RTU on a serial port.
    :param host: The IP address or hostname of the Modbus server. Required for TCP.
    :param port: The TCP port number, typically 502. Required for TCP.
    :param serial_port: The serial port, such as /dev/ttyUSB0 or COM3. Required for
        RTU.
    :param baud_rate: The baud rate of the serial line: 1200, 2400, 4800, 9600, 19200,
        38400, 57600, 115200, or 230400.
    :param parity: The parity of the serial line.
    :param stop_bits: The stop bits of each character on the serial line.
    :param swap_bytes: Whether register channels swap the byte order within each
        16-bit word. A channel's bytes_swapped overrides it.
    :param swap_words: Whether register channels swap the word order of 32-bit and
        larger values. A channel's words_swapped overrides it.
    :param unit_id: The unit the device addresses. For TCP, from 0 to 255, and 255 by
        default: 255 addresses the server itself, and a gateway needs the unit behind
        it. For RTU, from 1 to 247, and 1 by default.
    :param timeout_ms: The connect and response timeout in milliseconds.
    :param retries: The immediate retries of a request after a timeout or another
        temporary fault, from 0 to 10.
    :param max_registers: The most registers one read covers, from 1 to 125.
    :param max_gap: The most unused addresses one read can span between two channels.
        0 reads only addresses that channels cover.
    :param write_codes: "multiple" writes with function codes 15 and 16. "single" writes
        each coil with 05 and each register with 06.
    :param mask_write_supported: Whether the server takes function code 22, so a
        register bit write is one request. Otherwise the Driver reads the register and
        writes it back.
    :param delay_ms: The wait in milliseconds between requests on the connection,
        from 0 to 5000.
    :param name: Human-readable name for the device.
    :param location: Physical location or description.
    :param rack: Rack key this device belongs to.
    :param key: Unique key for the device. Auto-generated if empty.
    :param configured: Whether the device has been configured.
    :raises ValueError: If a field of the other transport is set, or a required field
        of this one is missing.
    :raises PathError: If baud_rate is not one of the supported rates.
    """

    def __init__(
        self,
        *,
        transport: Literal["tcp", "rtu"] = "tcp",
        host: str | None = None,
        port: int | None = None,
        serial_port: str | None = None,
        baud_rate: _BaudRate = 19200,
        parity: Literal["none", "even", "odd"] = "even",
        stop_bits: Literal[1, 2] = 1,
        swap_bytes: bool = False,
        swap_words: bool = False,
        unit_id: int | None = None,
        timeout_ms: int = 1000,
        retries: int = 1,
        max_registers: int = 125,
        max_gap: int = 125,
        write_codes: Literal["multiple", "single"] = "multiple",
        mask_write_supported: bool = False,
        delay_ms: int = 0,
        name: str = "",
        location: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = False,
    ):
        if not key:
            key = str(uuid4())
        connection: dict[str, JsonValue] = {"transport": transport}
        if transport == "tcp":
            if host is None or port is None:
                raise ValueError("a TCP device needs a host and a port")
            if serial_port is not None:
                raise ValueError("a TCP device has no serial port")
            connection.update(host=host, port=port)
            if unit_id is None:
                unit_id = 255
        else:
            if serial_port is None:
                raise ValueError("an RTU device needs a serial port")
            if host is not None or port is not None:
                raise ValueError("an RTU device has no host or port")
            if baud_rate not in _BAUD_RATES:
                rates = ", ".join(str(rate) for rate in _BAUD_RATES)
                raise PathError("baud_rate", ValidationError(f"must be one of {rates}"))
            connection.update(
                serial_port=serial_port,
                baud_rate=baud_rate,
                parity=parity,
                stop_bits=stop_bits,
            )
            if unit_id is None:
                unit_id = 1
        connection.update(
            swap_bytes=swap_bytes,
            swap_words=swap_words,
            unit_id=unit_id,
            timeout_ms=timeout_ms,
            retries=retries,
        )
        props: dict[str, JsonValue] = {
            "connection": connection,
            "requests": {
                "max_registers": max_registers,
                "max_gap": max_gap,
                "write_codes": write_codes,
                "mask_write_supported": mask_write_supported,
                "delay_ms": delay_ms,
            },
            "read": {"index": 0, "channels": {}},
            "write": {"channels": {}},
        }
        super().__init__(
            key=key,
            location=location,
            rack=rack,
            name=name,
            make=MAKE,
            model=MODEL,
            configured=configured,
            properties=props,
        )
