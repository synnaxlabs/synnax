#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""EtherCAT driver types for Synnax Python SDK.

This module provides Python types for configuring EtherCAT read and write tasks
that communicate with EtherCAT slave devices via the Synnax driver.

EtherCAT channels can be configured in two modes:
- **Automatic**: Reference PDOs by name (e.g., "Position actual value"). The driver
  looks up the PDO address from the slave device properties discovered during scanning.
- **Manual**: Specify PDO addresses directly (index, subindex, bit_length, data_type).

Example:
    >>> import synnax as sy
    >>> from synnax import ethercat
    >>>
    >>> # Create an automatic input channel (PDO resolved by name)
    >>> position_ch = ethercat.AutomaticInputChan(
    ...     device="slave-device-key",
    ...     pdo="Position actual value",
    ...     channel=position_channel.key,
    ... )
    >>>
    >>> # Create a read task
    >>> read_task = ethercat.ReadTask(
    ...     name="EtherCAT Read",
    ...     sample_rate=1000,
    ...     stream_rate=100,
    ...     channels=[position_ch],
    ... )
    >>> client.tasks.configure(read_task)
"""

import json
from typing import Literal
from uuid import uuid4

from pydantic import BaseModel, Field, conint, field_validator

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
from synnax.telem import CrudeRate

# Device identifiers - must match driver expectations
MAKE = "EtherCAT"
MODEL = "Slave"


class PDOEntry(BaseModel):
    """Information about a single PDO entry from device discovery.

    This model represents a PDO (Process Data Object) entry stored in
    slave device properties after scanning. Note that this represents an
    individual entry within a PDO, not the PDO container itself.

    :param name: Human-readable name of the PDO entry (e.g., "Position actual value").
    :param pdo_index: Parent PDO index (e.g., 0x1A00 for TxPDO, 0x1600 for RxPDO).
    :param index: CoE object dictionary index (e.g., 0x6064 = 24676).
    :param sub_index: CoE object dictionary subindex.
    :param bit_length: Size of the data in bits.
    :param data_type: Data type string (e.g., "uint16", "int32", "float32").
    """

    name: str
    pdo_index: int = Field(default=0, ge=0, le=65535)
    index: int = Field(ge=0, le=65535)
    sub_index: int = Field(ge=0, le=255)
    bit_length: int = Field(ge=1, le=64)
    data_type: str


class BaseChan(BaseModel):
    """Base class for all EtherCAT channels."""

    key: str = ""
    "A unique key to identify this channel configuration."
    enabled: bool = True
    "Whether the channel is enabled for data exchange."
    device: str = Field(min_length=1)
    "The key of the Synnax slave device this channel belongs to."

    def __init__(self, **data):
        if "key" not in data or not data["key"]:
            data["key"] = str(uuid4())
        super().__init__(**data)


class AutomaticInputChan(BaseChan):
    """Input channel that resolves PDO address from slave device properties by name.

    Automatic channels reference PDOs by their human-readable name. The driver
    looks up the PDO address (index, subindex, bit_length, data_type) from the
    slave device properties that were populated during device scanning.

    This is the recommended approach as it's more readable and less error-prone
    than specifying raw PDO addresses.

    Example:
        >>> # Read position from a servo drive
        >>> position_ch = AutomaticInputChan(
        ...     device="servo-drive-key",
        ...     pdo="Position actual value",
        ...     channel=position_channel.key,
        ... )

    :param device: The key of the Synnax slave device this channel belongs to.
    :param pdo: The name of the PDO to read (must match a name in device properties).
    :param channel: The Synnax channel key that data will be written to.
    :param enabled: Whether the channel is enabled for data acquisition.
    :param key: Unique identifier (auto-generated if not provided).
    """

    type: Literal["automatic"] = "automatic"
    pdo: str = Field(min_length=1)
    "The name of the PDO to look up in slave device properties."
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."


class ManualInputChan(BaseChan):
    """Input channel with PDO address specified directly.

    Manual channels allow specifying the exact PDO address when the PDO name
    is not known or when working with devices that don't have properly
    populated PDO names.

    Example:
        >>> # Read a PDO at index 0x6064, sub_index 0
        >>> position_ch = ManualInputChan(
        ...     device="servo-drive-key",
        ...     index=0x6064,
        ...     sub_index=0,
        ...     bit_length=32,
        ...     data_type="int32",
        ...     channel=position_channel.key,
        ... )

    :param device: The key of the Synnax slave device this channel belongs to.
    :param index: CoE object dictionary index (0-65535).
    :param sub_index: CoE object dictionary subindex (0-255).
    :param bit_length: Size of the data in bits (1-64).
    :param data_type: Data type string (e.g., "uint16", "int32", "float32").
    :param channel: The Synnax channel key that data will be written to.
    :param enabled: Whether the channel is enabled for data acquisition.
    :param key: Unique identifier (auto-generated if not provided).
    """

    type: Literal["manual"] = "manual"
    index: int = Field(ge=0, le=65535)
    "CoE object dictionary index (e.g., 0x6064 = 24676)."
    sub_index: int = Field(ge=0, le=255)
    "CoE object dictionary subindex."
    bit_length: int = Field(ge=1, le=64)
    "Size of the data in bits."
    data_type: str
    "Data type string (e.g., 'uint8', 'int16', 'uint32', 'int32', 'float32')."
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."


# Union type for all input channels
InputChan = AutomaticInputChan | ManualInputChan


class AutomaticOutputChan(BaseChan):
    """Output channel that resolves PDO address from slave device properties by name.

    Automatic output channels reference PDOs by their human-readable name. The driver
    looks up the PDO address from the slave device properties.

    Example:
        >>> # Control velocity on a servo drive
        >>> velocity_cmd = AutomaticOutputChan(
        ...     device="servo-drive-key",
        ...     pdo="Target velocity",
        ...     cmd_channel=velocity_command.key,
        ...     state_channel=velocity_state.key,  # Optional state feedback
        ... )

    :param device: The key of the Synnax slave device this channel belongs to.
    :param pdo: The name of the PDO to write (must match a name in device properties).
    :param cmd_channel: The Synnax channel key to receive command values from.
    :param state_channel: The Synnax channel key to write state feedback to (optional).
    :param enabled: Whether the channel is enabled for output operations.
    :param key: Unique identifier (auto-generated if not provided).
    """

    type: Literal["automatic"] = "automatic"
    pdo: str = Field(min_length=1)
    "The name of the PDO to look up in slave device properties."
    cmd_channel: ChannelKey
    "The Synnax channel key to receive command values from."
    state_channel: ChannelKey = 0
    "The Synnax channel key to write state feedback to (0 to disable)."


class ManualOutputChan(BaseChan):
    """Output channel with PDO address specified directly.

    Manual output channels allow specifying the exact PDO address when the PDO name
    is not known or when working with devices that don't have properly populated
    PDO names.

    Example:
        >>> # Write to a PDO at index 0x60FF, sub_index 0
        >>> velocity_cmd = ManualOutputChan(
        ...     device="servo-drive-key",
        ...     index=0x60FF,
        ...     sub_index=0,
        ...     bit_length=32,
        ...     data_type="int32",
        ...     cmd_channel=velocity_command.key,
        ... )

    :param device: The key of the Synnax slave device this channel belongs to.
    :param index: CoE object dictionary index (0-65535).
    :param sub_index: CoE object dictionary subindex (0-255).
    :param bit_length: Size of the data in bits (1-64).
    :param data_type: Data type string (e.g., "uint16", "int32", "float32").
    :param cmd_channel: The Synnax channel key to receive command values from.
    :param state_channel: The Synnax channel key to write state feedback to (optional).
    :param enabled: Whether the channel is enabled for output operations.
    :param key: Unique identifier (auto-generated if not provided).
    """

    type: Literal["manual"] = "manual"
    index: int = Field(ge=0, le=65535)
    "CoE object dictionary index (e.g., 0x60FF = 24831)."
    sub_index: int = Field(ge=0, le=255)
    "CoE object dictionary subindex."
    bit_length: int = Field(ge=1, le=64)
    "Size of the data in bits."
    data_type: str
    "Data type string (e.g., 'uint8', 'int16', 'uint32', 'int32', 'float32')."
    cmd_channel: ChannelKey
    "The Synnax channel key to receive command values from."
    state_channel: ChannelKey = 0
    "The Synnax channel key to write state feedback to (0 to disable)."


# Union type for all output channels
OutputChan = AutomaticOutputChan | ManualOutputChan


class ReadTaskConfig(BaseReadTaskConfig):
    """Configuration for an EtherCAT read task.

    Inherits common read task fields (sample_rate, stream_rate, data_saving,
    auto_start) from BaseReadTaskConfig and adds EtherCAT-specific channel
    configuration.

    The sample_rate must be divisible by stream_rate. For example, a sample_rate
    of 1000 Hz and stream_rate of 100 Hz will result in batches of 10 samples
    being streamed to Synnax every 10ms.

    EtherCAT supports high-frequency sampling. The default rate limits are set
    to 50 kHz, but actual achievable rates depend on the number of slaves and
    PDOs configured.

    :param sample_rate: The rate at which to sample data from EtherCAT slaves (Hz).
    :param stream_rate: The rate at which data is streamed to Synnax (Hz).
    :param data_saving: Whether to persist data to disk (True) or only stream (False).
    :param auto_start: Whether to start the task automatically when configured.
    :param channels: List of input channel configurations (automatic or manual).
    """

    channels: list[InputChan]
    "A list of input channel configurations to acquire data from."

    @field_validator("channels")
    def validate_channels_not_empty(cls, v):
        """Validate that at least one channel is provided."""
        if len(v) == 0:
            raise ValueError("Task must have at least one channel")
        return v


class WriteTaskConfig(BaseWriteTaskConfig):
    """Configuration for an EtherCAT write task.

    Inherits common write task fields (device, auto_start) from BaseWriteTaskConfig
    and adds EtherCAT-specific fields for state feedback and execution rate.

    Note: The `device` field from BaseWriteTaskConfig is not used for EtherCAT
    write tasks since channels can span multiple slave devices. Each channel
    specifies its own device.

    :param state_rate: Rate at which state feedback is written to Synnax (Hz).
    :param execution_rate: Rate at which commands are executed on the bus (Hz).
    :param data_saving: Whether to persist state data to disk.
    :param auto_start: Whether to start the task automatically when configured.
    :param channels: List of output channel configurations (automatic or manual).
    """

    device: str = ""
    "Not used for EtherCAT - each channel specifies its own device."
    state_rate: float = Field(default=1.0, ge=0, le=10000)
    "Rate at which state feedback is written to Synnax (Hz)."
    execution_rate: float = Field(default=1000.0, ge=0, le=50000)
    "Rate at which commands are executed on the EtherCAT bus (Hz)."
    data_saving: bool = True
    "Whether to persist state data to disk."
    channels: list[OutputChan]
    "A list of output channel configurations to write to."

    @field_validator("channels")
    def validate_channels_not_empty(cls, v):
        """Validate that at least one channel is provided."""
        if len(v) == 0:
            raise ValueError("Task must have at least one channel")
        return v


class ReadTask(StarterStopperMixin, JSONConfigMixin, TaskProtocol):
    """A read task for sampling data from EtherCAT slave devices.

    This task configures the Synnax driver to perform cyclic PDO exchange with
    EtherCAT slaves and write the acquired data to Synnax channels.

    The task supports both automatic channels (PDO resolved by name) and manual
    channels (PDO address specified directly).

    Example:
        >>> import synnax as sy
        >>> from synnax import ethercat
        >>>
        >>> # Create channels for time and position
        >>> time_ch = client.channels.create(
        ...     name="ecat_time",
        ...     data_type=sy.DataType.TIMESTAMP,
        ...     is_index=True,
        ... )
        >>> position_ch = client.channels.create(
        ...     name="position",
        ...     data_type=sy.DataType.INT32,
        ...     index=time_ch.key,
        ... )
        >>>
        >>> # Create the read task
        >>> read_task = ethercat.ReadTask(
        ...     name="EtherCAT Position Read",
        ...     sample_rate=1000,
        ...     stream_rate=100,
        ...     data_saving=True,
        ...     channels=[
        ...         ethercat.AutomaticInputChan(
        ...             device="servo-drive-key",
        ...             pdo="Position actual value",
        ...             channel=position_ch.key,
        ...         ),
        ...     ],
        ... )
        >>>
        >>> # Configure and start
        >>> client.tasks.configure(read_task)
        >>> read_task.start()

    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from slaves (Hz).
    :param stream_rate: The rate at which data is streamed to Synnax (Hz).
    :param data_saving: Whether to save data permanently or just stream it.
    :param auto_start: Whether to start the task automatically when configured.
    :param channels: List of input channel configurations.
    """

    TYPE = "ethercat_read"
    config: ReadTaskConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        name: str = "",
        sample_rate: CrudeRate = 0,
        stream_rate: CrudeRate = 0,
        data_saving: bool = True,
        auto_start: bool = False,
        channels: list[InputChan] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadTaskConfig.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = ReadTaskConfig(
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )


class WriteTask(StarterStopperMixin, JSONConfigMixin, TaskProtocol):
    """A write task for sending commands to EtherCAT slave devices.

    This task configures the Synnax driver to receive commands from Synnax
    channels and write them to EtherCAT slave PDOs during cyclic exchange.

    The task supports state feedback - if state_channel is specified on a
    channel, the current PDO value will be written back to Synnax at the
    configured state_rate.

    Example:
        >>> import synnax as sy
        >>> from synnax import ethercat
        >>>
        >>> # Create command and state channels
        >>> cmd_time = client.channels.create(
        ...     name="cmd_time",
        ...     data_type=sy.DataType.TIMESTAMP,
        ...     is_index=True,
        ... )
        >>> velocity_cmd = client.channels.create(
        ...     name="velocity_cmd",
        ...     data_type=sy.DataType.INT32,
        ...     index=cmd_time.key,
        ... )
        >>> velocity_state = client.channels.create(
        ...     name="velocity_state",
        ...     data_type=sy.DataType.INT32,
        ...     index=cmd_time.key,
        ... )
        >>>
        >>> # Create the write task
        >>> write_task = ethercat.WriteTask(
        ...     name="EtherCAT Velocity Control",
        ...     state_rate=10,
        ...     execution_rate=1000,
        ...     channels=[
        ...         ethercat.AutomaticOutputChan(
        ...             device="servo-drive-key",
        ...             pdo="Target velocity",
        ...             cmd_channel=velocity_cmd.key,
        ...             state_channel=velocity_state.key,
        ...         ),
        ...     ],
        ... )
        >>>
        >>> # Configure and start
        >>> client.tasks.configure(write_task)
        >>> write_task.start()

    :param name: A human-readable name for the task.
    :param state_rate: Rate at which state feedback is written to Synnax (Hz).
    :param execution_rate: Rate at which commands are executed on the bus (Hz).
    :param data_saving: Whether to persist state data to disk.
    :param auto_start: Whether to start the task automatically when configured.
    :param channels: List of output channel configurations.
    """

    TYPE = "ethercat_write"
    config: WriteTaskConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        name: str = "",
        state_rate: float = 1.0,
        execution_rate: float = 1000.0,
        data_saving: bool = True,
        auto_start: bool = False,
        channels: list[OutputChan] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteTaskConfig.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = WriteTaskConfig(
            state_rate=state_rate,
            execution_rate=execution_rate,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )


class Device(device.Device):
    """EtherCAT slave device configuration.

    This class extends the base Device class to provide EtherCAT-specific
    properties including vendor/product identification, bus position, and
    PDO definitions discovered during scanning.

    Typically, EtherCAT devices are created automatically by the driver's
    scan task. However, this class can be used to create devices manually
    or to update device properties.

    Example:
        >>> from synnax import ethercat
        >>>
        >>> # Create an EtherCAT slave device manually
        >>> device = ethercat.Device(
        ...     name="Servo Drive",
        ...     network="eth0",
        ...     position=0,
        ...     vendor_id=0x00000002,
        ...     product_code=0x12345678,
        ...     revision=0x00010000,
        ...     serial=12345,
        ...     rack=rack.key,
        ...     input_pdos=[
        ...         ethercat.PDOEntry(
        ...             name="Position actual value",
        ...             index=0x6064,
        ...             sub_index=0,
        ...             bit_length=32,
        ...             data_type="int32",
        ...         ),
        ...     ],
        ...     output_pdos=[
        ...         ethercat.PDOEntry(
        ...             name="Target velocity",
        ...             index=0x60FF,
        ...             sub_index=0,
        ...             bit_length=32,
        ...             data_type="int32",
        ...         ),
        ...     ],
        ... )
        >>> client.devices.create(device)

    :param name: Human-readable name for the device.
    :param network: Network interface name (e.g., "eth0", "enp3s0").
    :param position: Position of the slave on the EtherCAT bus (0-indexed).
    :param vendor_id: EtherCAT vendor ID from device EEPROM.
    :param product_code: Product code identifying the device model.
    :param revision: Hardware/firmware revision number.
    :param serial: Unique serial number from device EEPROM.
    :param rack: Rack key this device belongs to.
    :param input_pdos: List of input PDO entry definitions (TxPDO, slave->master).
    :param output_pdos: List of output PDO entry definitions (RxPDO, master->slave).
    :param key: Unique key for the device (auto-generated if empty).
    :param configured: Whether the device has been configured.
    :param enabled: Whether the device is enabled for operation.
    """

    def __init__(
        self,
        *,
        name: str = "",
        network: str = "",
        position: int = 0,
        vendor_id: int = 0,
        product_code: int = 0,
        revision: int = 0,
        serial: int = 0,
        rack: int = 0,
        input_pdos: list[PDOEntry] | None = None,
        output_pdos: list[PDOEntry] | None = None,
        key: str = "",
        configured: bool = False,
        enabled: bool = True,
    ):
        """Initialize an EtherCAT slave device."""
        if not key:
            key = str(uuid4())

        props = {
            "network": network,
            "position": position,
            "vendor_id": vendor_id,
            "product_code": product_code,
            "revision": revision,
            "serial": serial,
            "name": name,
            "enabled": enabled,
            "pdos": {
                "inputs": [p.model_dump() for p in (input_pdos or [])],
                "outputs": [p.model_dump() for p in (output_pdos or [])],
            },
        }

        super().__init__(
            key=key,
            location=f"{network}:{position}",
            rack=rack,
            name=name,
            make=MAKE,
            model=MODEL,
            configured=configured,
            properties=json.dumps(props),
        )
