#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from typing import Literal
from uuid import uuid4

from pydantic import BaseModel, Field, confloat, conint, field_validator

from synnax import device
from synnax.channel import ChannelKey
from synnax.task import (
    BaseReadTaskConfig,
    BaseWriteTaskConfig,
    JSONConfigMixin,
    StarterStopperMixin,
    Task,
    TaskProtocol,
)
from synnax.telem import CrudeDataType, CrudeRate

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


class HoldingRegisterInputChan(BaseChan):
    """
    Channel configuration for reading from Modbus holding registers (16-bit R/W registers).

    Holding registers are the most universal 16-bit registers, may be read or written,
    and may be used for a variety of things including inputs, outputs, configuration data,
    or any requirement for "holding" data. Function code 03 (Read Holding Registers).

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>

    Example 1: Reading a float32 sensor value
        >>> # Most common: 32-bit float spanning 2 registers
        >>> temp_sensor = HoldingRegisterInputChan(
        ...     address=40001,
        ...     channel=100,
        ...     data_type="float32",  # Spans 2 registers
        ...     swap_bytes=False,     # Match device byte order
        ...     swap_words=False      # Match device word order
        ... )

    Example 2: Reading a 16-bit signed integer
        >>> # Single register value
        >>> pressure_raw = HoldingRegisterInputChan(
        ...     address=40010,
        ...     channel=101,
        ...     data_type="int16",
        ...     enabled=True
        ... )

    Example 3: Reading a 32-bit unsigned integer
        >>> # Accumulated count or large value
        >>> total_count = HoldingRegisterInputChan(
        ...     address=40020,
        ...     channel=102,
        ...     data_type="uint32",   # Spans 2 registers
        ...     swap_bytes=True,      # Device uses different endianness
        ...     swap_words=False
        ... )

    Example 4: Reading a string
        >>> # ASCII string stored in registers
        >>> device_name = HoldingRegisterInputChan(
        ...     address=40100,
        ...     channel=103,
        ...     data_type="STRING",
        ...     string_length=16      # 16 characters (8 registers)
        ... )

    :param address: Modbus register address (0-65535, commonly 40001-49999 in notation)
    :param channel: Synnax channel key to write data to
    :param data_type: Data type interpretation (float32, int16, uint16, int32, uint32, STRING, etc.)
    :param swap_bytes: Swap byte order within each 16-bit word (AB->BA)
    :param swap_words: Swap word order for multi-register values (ABCD->CDAB)
    :param string_length: String length in characters (required for STRING data type)
    :param key: Unique identifier (auto-generated if not provided)
    :param enabled: Whether the channel is enabled for acquisition
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

    Input registers are 16-bit read-only registers used for input data from sensors,
    typically analog values or status information. Function code 04 (Read Input Registers).
    They differ from holding registers in that they cannot be written to.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>

    Example 1: Reading sensor temperature as float32
        >>> # Temperature sensor value
        >>> temperature = InputRegisterChan(
        ...     address=30001,
        ...     channel=200,
        ...     data_type="float32",
        ...     swap_bytes=False,
        ...     swap_words=False
        ... )

    Example 2: Reading analog input as int16
        >>> # Raw ADC value from sensor
        >>> analog_input = InputRegisterChan(
        ...     address=30010,
        ...     channel=201,
        ...     data_type="int16"
        ... )

    Example 3: Reading process variable as uint32
        >>> # Large accumulator or counter value
        >>> flow_total = InputRegisterChan(
        ...     address=30100,
        ...     channel=202,
        ...     data_type="uint32",
        ...     swap_bytes=True,  # Adjust for device endianness
        ...     swap_words=True
        ... )

    :param address: Modbus register address (0-65535, commonly 30001-39999 in notation)
    :param channel: Synnax channel key to write data to
    :param data_type: Data type interpretation (float32, int16, uint16, int32, uint32, STRING, etc.)
    :param swap_bytes: Swap byte order within each 16-bit word (AB->BA)
    :param swap_words: Swap word order for multi-register values (ABCD->CDAB)
    :param string_length: String length in characters (required for STRING data type)
    :param key: Unique identifier (auto-generated if not provided)
    :param enabled: Whether the channel is enabled for acquisition
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

    Coils are 1-bit registers used to control discrete outputs, and may be read or written.
    They represent binary states such as ON/OFF or TRUE/FALSE. Function code 01 (Read Coils).
    Commonly used for relay states, valve positions, motor run status, etc.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>

    Example 1: Reading relay state
        >>> # Monitor a relay or actuator state
        >>> relay_state = CoilInputChan(
        ...     address=1,      # Coil address 1
        ...     channel=300
        ... )

    Example 2: Reading motor run status
        >>> # Check if motor is running
        >>> motor_running = CoilInputChan(
        ...     address=100,
        ...     channel=301,
        ...     enabled=True
        ... )

    Example 3: Reading multiple valve states
        >>> # Monitor several on/off valves
        >>> valve1 = CoilInputChan(address=10, channel=302)
        >>> valve2 = CoilInputChan(address=11, channel=303)
        >>> valve3 = CoilInputChan(address=12, channel=304)

    :param address: Modbus coil address (0-65535, commonly 00001-09999 in notation)
    :param channel: Synnax channel key to write coil state to (0 or 1)
    :param key: Unique identifier (auto-generated if not provided)
    :param enabled: Whether the channel is enabled for acquisition
    """

    type: Literal["coil_input"] = "coil_input"
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."


class DiscreteInputChan(BaseChan):
    """
    Channel configuration for reading from Modbus discrete inputs (1-bit R-only registers).

    Discrete inputs are 1-bit read-only registers used for binary input status.
    They are similar to coils but cannot be written to. Function code 02 (Read Discrete Inputs).
    Commonly used for limit switches, proximity sensors, alarm states, or digital inputs.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>

    Example 1: Reading limit switch state
        >>> # Monitor a mechanical limit switch
        >>> limit_switch = DiscreteInputChan(
        ...     address=10001,
        ...     channel=400
        ... )

    Example 2: Reading alarm status
        >>> # Monitor a system alarm condition
        >>> high_temp_alarm = DiscreteInputChan(
        ...     address=10010,
        ...     channel=401,
        ...     enabled=True
        ... )

    Example 3: Reading multiple safety interlocks
        >>> # Monitor safety system states
        >>> estop_status = DiscreteInputChan(address=10100, channel=402)
        >>> door_closed = DiscreteInputChan(address=10101, channel=403)
        >>> guard_locked = DiscreteInputChan(address=10102, channel=404)

    :param address: Modbus discrete input address (0-65535, commonly 10001-19999 in notation)
    :param channel: Synnax channel key to write discrete state to (0 or 1)
    :param key: Unique identifier (auto-generated if not provided)
    :param enabled: Whether the channel is enabled for acquisition
    """

    type: Literal["discrete_input"] = "discrete_input"
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."


# Union type for all input channels
InputChan = (
    HoldingRegisterInputChan | InputRegisterChan | CoilInputChan | DiscreteInputChan
)


class CoilOutputChan(BaseChan):
    """
    Channel configuration for writing to Modbus coils (1-bit R/W registers).

    Coils are 1-bit registers used to control discrete outputs. This channel
    allows writing boolean values (0/1, False/True) to coil addresses.
    Function code 05 (Write Single Coil) or 15 (Write Multiple Coils).
    Commonly used for controlling relays, solenoid valves, motors, or indicators.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>

    Example 1: Controlling a relay
        >>> # Turn a relay on/off
        >>> relay_control = CoilOutputChan(
        ...     address=1,
        ...     channel=500  # Synnax command channel (write 0 or 1)
        ... )

    Example 2: Controlling a solenoid valve
        >>> # Open/close a pneumatic valve
        >>> valve_control = CoilOutputChan(
        ...     address=100,
        ...     channel=501,
        ...     enabled=True
        ... )

    Example 3: Motor start/stop control
        >>> # Start/stop motor via Modbus coil
        >>> motor_start = CoilOutputChan(
        ...     address=200,
        ...     channel=502
        ... )

    :param address: Modbus coil address to write to (0-65535, commonly 00001-09999 in notation)
    :param channel: Synnax channel key to read command values from (0 or 1)
    :param key: Unique identifier (auto-generated if not provided)
    :param enabled: Whether the channel is enabled for output operations
    """

    type: Literal["coil_output"] = "coil_output"
    channel: ChannelKey
    "The Synnax channel key to read command values from."


class HoldingRegisterOutputChan(BaseChan):
    """
    Channel configuration for writing to Modbus holding registers (16-bit R/W registers).

    Holding registers are 16-bit registers that can be used for outputs, configuration data,
    or any requirement for "holding" data that needs to be both read and written.
    Function code 06 (Write Single Register) or 16 (Write Multiple Registers).
    Commonly used for setpoints, configuration values, or analog control outputs.

    For detailed information, see the Modbus specification:
    <https://www.modbustools.com/modbus.html>

    Example 1: Writing a temperature setpoint (float32)
        >>> # Set desired temperature
        >>> temp_setpoint = HoldingRegisterOutputChan(
        ...     address=40001,
        ...     channel=600,       # Synnax command channel
        ...     data_type="float32",
        ...     swap_bytes=False,
        ...     swap_words=False
        ... )

    Example 2: Writing a speed control value (int16)
        >>> # Set motor speed (0-1000 RPM)
        >>> speed_control = HoldingRegisterOutputChan(
        ...     address=40010,
        ...     channel=601,
        ...     data_type="int16"
        ... )

    Example 3: Writing a large accumulator value (uint32)
        >>> # Reset or set counter value
        >>> counter_preset = HoldingRegisterOutputChan(
        ...     address=40020,
        ...     channel=602,
        ...     data_type="uint32",
        ...     swap_bytes=True,  # Match device endianness
        ...     swap_words=False
        ... )

    Example 4: Writing configuration parameter
        >>> # Set device operating mode
        >>> operating_mode = HoldingRegisterOutputChan(
        ...     address=40100,
        ...     channel=603,
        ...     data_type="uint16",
        ...     enabled=True
        ... )

    :param address: Modbus register address to write to (0-65535, commonly 40001-49999 in notation)
    :param channel: Synnax channel key to read command values from
    :param data_type: Data type to write (float32, int16, uint16, int32, uint32, etc.)
    :param swap_bytes: Swap byte order within each 16-bit word (AB->BA)
    :param swap_words: Swap word order for multi-register values (ABCD->CDAB)
    :param key: Unique identifier (auto-generated if not provided)
    :param enabled: Whether the channel is enabled for output operations
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


class ReadTaskConfig(BaseReadTaskConfig):
    """Configuration for a Modbus TCP read task.

    Inherits common read task fields (sample_rate, stream_rate, data_saving,
    auto_start) from BaseReadTaskConfig and adds Modbus-specific channel configuration
    with Modbus hardware sample rate limits (10 kHz max).
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

    Inherits common write task fields (device, auto_start) from
    BaseWriteTaskConfig and adds Modbus-specific channel configuration.
    Modbus write tasks do not support state feedback, so they do not use
    state_rate or data_saving.
    """

    channels: list[OutputChan]
    "A list of output channel configurations to write to."

    @field_validator("channels")
    def validate_channels_not_empty(cls, v):
        """Validate that at least one channel is provided."""
        if len(v) == 0:
            raise ValueError("Task must have at least one channel")
        return v


class ReadTask(StarterStopperMixin, JSONConfigMixin, TaskProtocol):
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

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration.

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
        return device_client.create(dev)


class WriteTask(StarterStopperMixin, JSONConfigMixin, TaskProtocol):
    """
    A write task for sending commands to Modbus TCP devices. This task is a programmatic
    representation of the Modbus write task configurable within the Synnax console.

    For detailed information on configuring/operating a Modbus write task, see
    https://docs.synnaxlabs.com/reference/driver/modbus/write-task


    :param device: The key of the Synnax Modbus device to write to.
    :param name: A human-readable name for the task.
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
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration.

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
        return device_client.create(dev)


class Device(device.Device):
    """
    Modbus TCP device configuration.

    This class extends the base Device class to provide Modbus-specific configuration
    including TCP connection parameters and byte/word swap settings.

    Example:
        >>> from synnax import modbus
        >>> device = modbus.Device(
        ...     host="192.168.1.100",
        ...     port=502,
        ...     name="Modbus Server",
        ...     location="192.168.1.100:502",
        ...     rack=rack.key,
        ...     swap_bytes=False,
        ...     swap_words=False
        ... )
        >>> client.devices.create(device)

    :param host: The IP address or hostname of the Modbus server
    :param port: The TCP port number (typically 502)
    :param swap_bytes: Whether to swap byte order within 16-bit words
    :param swap_words: Whether to swap word order for 32-bit+ values
    """

    def __init__(
        self,
        *,
        host: str,
        port: int,
        swap_bytes: bool = False,
        swap_words: bool = False,
        name: str = "",
        location: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = False,
    ):
        """
        Initialize a Modbus TCP device.

        Args:
            host: The IP address or hostname of the Modbus server
            port: The TCP port number (typically 502)
            swap_bytes: Whether to swap byte order within 16-bit words
            swap_words: Whether to swap word order for 32-bit+ values
            name: Human-readable name for the device
            location: Physical location or description
            rack: Rack key this device belongs to
            key: Unique key for the device (auto-generated if empty)
            configured: Whether the device has been configured
        """
        # Auto-generate key if not provided
        if not key:
            key = str(uuid4())

        # Build connection properties
        props = {
            "connection": {
                "host": host,
                "port": port,
                "swap_bytes": swap_bytes,
                "swap_words": swap_words,
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
            properties=json.dumps(props),
        )
