#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal
from uuid import uuid4

from pydantic import BaseModel, Field, confloat, conint, field_validator

from synnax.channel import ChannelKey
from synnax.hardware.task import (
    BaseReadTaskConfig,
    BaseWriteTaskConfig,
    JSONConfigMixin,
    MetaTask,
    StarterStopperMixin,
    Task,
)
from synnax.telem import CrudeRate

# Device identifiers - must match Console expectations
MAKE = "Modbus"
MODEL = "Modbus"


class BaseChan(BaseModel):
    """Base class for all Modbus channels."""

    key: str = ""
    "A unique key to identify this channel."
    enabled: bool = True
    "Whether the channel is enabled."
    address: int = Field(ge=0, le=65535)
    "The Modbus register address (0-65535)."

    def __init__(self, **data):
        if "key" not in data or not data["key"]:
            data["key"] = str(uuid4())
        super().__init__(**data)


# ================================ READ CHANNELS ================================


class HoldingRegisterInputChan(BaseChan):
    """
    Channel configuration for reading from Modbus holding registers (16-bit R/W registers).

    Holding registers are the most universal 16-bit registers, may be read or written,
    and may be used for a variety of things including inputs, outputs, configuration data,
    or any requirement for "holding" data.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>
    """

    type: Literal["holding_register_input"] = "holding_register_input"
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."
    data_type: str = "float32"
    "The data type to interpret the register(s) as (e.g., 'float32', 'int16', 'uint32')."
    swap_bytes: bool = False
    "Whether to swap the byte order for multi-register values."
    swap_words: bool = False
    "Whether to swap the word order for multi-register values."
    string_length: int = Field(default=0, ge=0)
    "String length for STRING data type. Ignored for other data types."


class InputRegisterChan(BaseChan):
    """
    Channel configuration for reading from Modbus input registers (16-bit R-only registers).

    Input registers are 16-bit registers used for input, and may only be read.
    They are typically used for analog input data from sensors.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>
    """

    type: Literal["register_input"] = "register_input"
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."
    data_type: str = "float32"
    "The data type to interpret the register(s) as (e.g., 'float32', 'int16', 'uint32')."
    swap_bytes: bool = False
    "Whether to swap the byte order for multi-register values."
    swap_words: bool = False
    "Whether to swap the word order for multi-register values."
    string_length: int = Field(default=0, ge=0)
    "String length for STRING data type. Ignored for other data types."


class CoilInputChan(BaseChan):
    """
    Channel configuration for reading from Modbus coils (1-bit R/W registers).

    Coils are 1-bit registers, are used to control discrete outputs, and may be read or written.
    They represent binary states such as ON/OFF or TRUE/FALSE.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>
    """

    type: Literal["coil_input"] = "coil_input"
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."


class DiscreteInputChan(BaseChan):
    """
    Channel configuration for reading from Modbus discrete inputs (1-bit R-only registers).

    Discrete inputs are 1-bit registers used as inputs, and may only be read.
    They are similar to coils but cannot be written to.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>
    """

    type: Literal["discrete_input"] = "discrete_input"
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."


# Union type for all input channels
InputChan = (
    HoldingRegisterInputChan | InputRegisterChan | CoilInputChan | DiscreteInputChan
)


# ================================ WRITE CHANNELS ================================


class CoilOutputChan(BaseChan):
    """
    Channel configuration for writing to Modbus coils (1-bit R/W registers).

    Coils are 1-bit registers used to control discrete outputs. This channel
    allows writing boolean values (0/1, False/True) to coil addresses.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>
    """

    type: Literal["coil_output"] = "coil_output"
    channel: ChannelKey
    "The Synnax channel key to read command values from."


class HoldingRegisterOutputChan(BaseChan):
    """
    Channel configuration for writing to Modbus holding registers (16-bit R/W registers).

    Holding registers are 16-bit registers that can be used for outputs, configuration data,
    or any requirement for "holding" data that needs to be both read and written.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>
    """

    type: Literal["holding_register_output"] = "holding_register_output"
    channel: ChannelKey
    "The Synnax channel key to read command values from."
    data_type: str = "float32"
    "The data type to interpret the register(s) as (e.g., 'float32', 'int16', 'uint32')."
    swap_bytes: bool = False
    "Whether to swap the byte order for multi-register values."
    swap_words: bool = False
    "Whether to swap the word order for multi-register values."


# Union type for all output channels
OutputChan = CoilOutputChan | HoldingRegisterOutputChan


# ================================ TASK CONFIGURATIONS ================================


class ReadTaskConfig(BaseReadTaskConfig):
    """Configuration for a Modbus TCP read task.

    Inherits common read task fields (sample_rate, stream_rate, data_saving,
    auto_start) from BaseReadTaskConfig and adds Modbus-specific channel configuration
    with Modbus hardware sample rate limits (10kHz max).
    """

    device: str = Field(min_length=1)
    "The key of the Synnax Modbus device to read from."
    sample_rate: conint(ge=0, le=10000)
    stream_rate: conint(ge=0, le=10000)
    channels: list[InputChan]
    "A list of input channel configurations to acquire data from."

    @field_validator("channels")
    def validate_channels_not_empty(cls, v):
        """Validate that at least one channel is provided."""
        if len(v) == 0:
            raise ValueError("Task must have at least one channel")
        return v


class WriteTaskConfig(BaseWriteTaskConfig):
    """Configuration for a Modbus TCP write task.

    Inherits common write task fields (device, data_saving, auto_start) from
    BaseWriteTaskConfig and adds Modbus-specific channel configuration.
    Modbus write tasks do not use state_rate.
    """

    channels: list[OutputChan]
    "A list of output channel configurations to write to."

    @field_validator("channels")
    def validate_channels_not_empty(cls, v):
        """Validate that at least one channel is provided."""
        if len(v) == 0:
            raise ValueError("Task must have at least one channel")
        return v



class ReadTask(StarterStopperMixin, JSONConfigMixin, MetaTask):
    """
    A read task for sampling data from Modbus TCP devices and writing the data to a
    Synnax cluster. This task is a programmatic representation of the Modbus read
    task configurable within the Synnax console.

    For detailed information on configuring/operating a Modbus read task, see
    https://docs.synnaxlabs.com/reference/driver/modbus/read-task


    :param device: The key of the Synnax Modbus device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the Modbus device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster. For example, a sample rate of 100Hz and a stream rate of 25Hz will
        result in groups of 4 samples being streamed to the cluster every 40ms.
    :param data_saving: Whether to save data permanently within Synnax, or just stream
        it for real-time consumption.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: A list of input channel configurations (InputChan subtypes:
        HoldingRegisterInputChan, InputRegisterChan, CoilInputChan, DiscreteInputChan).
    """

    TYPE = "modbus_read"
    config: ReadTaskConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        device: str = "",
        name: str = "",
        sample_rate: CrudeRate = 0,
        stream_rate: CrudeRate = 0,
        data_saving: bool = False,
        auto_start: bool = False,
        channels: list[InputChan] = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadTaskConfig.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = ReadTaskConfig(
            device=device,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )

    def _update_device_properties(self, device_client):
        """Internal: Update device properties before task configuration.

        This method synchronizes channel configurations with device properties
        so that the Console can properly map Modbus register addresses to Synnax channels.

        The key format follows Console's readMapKey convention:
        - For fixed-density channels (coils, discrete inputs): "{type}-{address}"
          Example: "coil-input-100" for a coil at address 100
        - For variable-density channels (registers): "{type}-{address}-{dataType}"
          Example: "holding-register-input-40001-float32" for a float32 at address 40001

        Keys use hyphens instead of underscores to match Console's naming convention.
        """
        import json

        dev = device_client.retrieve(key=self.config.device)
        props = (
            json.loads(dev.properties)
            if isinstance(dev.properties, str)
            else dev.properties
        )

        if "read" not in props:
            props["read"] = {"index": 0, "channels": {}}

        for ch in self.config.channels:
            # Generate key matching Console's readMapKey format
            key = f"{ch.type}-{ch.address}"
            # Variable density channels (holding_register_input, register_input) include dataType
            # because the same address can represent different data types depending on interpretation
            if hasattr(ch, "data_type"):
                key += f"-{ch.data_type}"
            # Replace underscores with hyphens
            key = key.replace("_", "-")

            props["read"]["channels"][key] = ch.channel

        dev.properties = json.dumps(props)
        device_client.create(dev)


class WriteTask(StarterStopperMixin, JSONConfigMixin, MetaTask):
    """
    A write task for sending commands to Modbus TCP devices. This task is a programmatic
    representation of the Modbus write task configurable within the Synnax console.

    For detailed information on configuring/operating a Modbus write task, see
    https://docs.synnaxlabs.com/reference/driver/modbus/write-task


    :param device: The key of the Synnax Modbus device to write to.
    :param name: A human-readable name for the task.
    :param data_saving: Whether to save data permanently within Synnax, or just stream
        it for real-time consumption.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: A list of output channel configurations (OutputChan subtypes:
        CoilOutputChan, HoldingRegisterOutputChan).
    """

    TYPE = "modbus_write"
    config: WriteTaskConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        device: str = "",
        name: str = "",
        data_saving: bool = False,
        auto_start: bool = False,
        channels: list[OutputChan] = None,
    ):
        if internal is not None:
            self._internal = internal
            self.config = WriteTaskConfig.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = WriteTaskConfig(
            device=device,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )

    def _update_device_properties(self, device_client):
        """Internal: Update device properties before task configuration.

        This method synchronizes channel configurations with device properties
        so that the Console can properly map Modbus register addresses to Synnax channels.

        The key format follows Console's writeMapKey convention:
        - Format: "{type}-{address}" (NO dataType suffix for write channels)
          Example: "coil-output-100" for a coil at address 100
          Example: "holding-register-output-40001" for a register at address 40001

        Write channels omit the dataType because writes are unambiguous - the driver
        converts the incoming value to the appropriate Modbus format based on register type.

        Keys use hyphens instead of underscores to match Console's naming convention.
        """
        import json

        dev = device_client.retrieve(key=self.config.device)
        props = (
            json.loads(dev.properties)
            if isinstance(dev.properties, str)
            else dev.properties
        )

        if "write" not in props:
            props["write"] = {"channels": {}}

        for ch in self.config.channels:
            # Generate key matching Console's writeMapKey format
            # Write channels don't include dataType in the key because writes are unambiguous
            key = f"{ch.type}-{ch.address}".replace("_", "-")

            # Map the generated key to the Synnax channel that will send command values
            props["write"]["channels"][key] = ch.channel

        dev.properties = json.dumps(props)
        device_client.create(dev)


# ================================ DEVICE HELPERS ================================


def device_props(
    host: str,
    port: int,
    swap_bytes: bool = False,
    swap_words: bool = False,
) -> dict:
    """
    Create device properties for a Modbus TCP connection.

    Args:
        host: The IP address or hostname of the Modbus server
        port: The TCP port number (typically 502)
        swap_bytes: Whether to swap byte order within 16-bit words
        swap_words: Whether to swap word order for 32-bit+ values

    Returns:
        Dictionary of device properties with the correct structure for Console
    """
    return {
        "connection": {
            "host": host,
            "port": port,
            "swap_bytes": swap_bytes,
            "swap_words": swap_words,
        },
        "read": {"index": 0, "channels": {}},
        "write": {"channels": {}},
    }


def create_device(client, **kwargs):
    """
    Create a Modbus device with make, model, and key automatically set.

    This is a thin wrapper around client.hardware.devices.create() that
    automatically fills in:
    - make: "Modbus"
    - model: "Modbus"
    - key: auto-generated UUID if not provided

    All other parameters are passed through unchanged.

    Example:
        >>> device = modbus.create_device(
        ...     client=client,
        ...     name="Modbus Server",
        ...     location="127.0.0.1:502",
        ...     rack=rack.key,
        ...     properties=json.dumps({...})
        ... )
    """
    from uuid import uuid4

    # Auto-generate key if not provided
    if "key" not in kwargs:
        kwargs["key"] = str(uuid4())

    kwargs["make"] = MAKE
    kwargs["model"] = MODEL
    return client.hardware.devices.create(**kwargs)
