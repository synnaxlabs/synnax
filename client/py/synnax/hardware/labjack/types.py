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
MAKE = "LabJack"
# Supported models
T4 = "LJM_dtT4"
T7 = "LJM_dtT7"
T8 = "LJM_dtT8"


class BaseChan(BaseModel):
    """Base class for all LabJack channels."""

    key: str = ""
    "A unique key to identify this channel."
    enabled: bool = True
    "Whether the channel is enabled."
    port: str = Field(min_length=1)
    "The port location of the channel (e.g., 'AIN0', 'DIO4')."

    def __init__(self, **data):
        if "key" not in data or not data["key"]:
            data["key"] = str(uuid4())
        super().__init__(**data)


class AIChan(BaseChan):
    """
    Analog Input Channel configuration for LabJack devices.

    Reads analog voltage from a specified input terminal. Supports single-ended
    and differential configurations via neg_chan parameter.

    For detailed information, see the LabJack documentation:
    <https://labjack.com/pages/support?doc=/datasheets/t-series-datasheet/>
    """

    type: Literal["AI"] = "AI"
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."
    range: confloat(gt=0) = 10.0
    "The voltage range for the channel (±range volts)."
    neg_chan: int = 199
    "The negative channel for differential measurements. 199 = single-ended (GND)."
    pos_chan: int = 0
    "The positive channel number (e.g., 0 for AIN0)."


class ThermocoupleChan(BaseChan):
    """
    Thermocouple Input Channel configuration for LabJack devices.

    Reads temperature from a thermocouple connected to an analog input with
    cold junction compensation (CJC). Supports multiple thermocouple types
    and CJC sources.

    Cold Junction Compensation (CJC) Explained:
    Thermocouples measure temperature difference between the measurement junction
    (hot end) and the reference junction (cold end, typically at the device terminals).
    To get absolute temperature, we need to know the temperature at the cold junction.

    CJC Source Options:
    1. Internal device sensor (default): Most common, easiest to use
    2. External temperature sensor: More accurate for high-precision measurements
    3. Another analog input: Use when you have a dedicated temperature sensor

    For detailed information, see the LabJack documentation:
    <https://labjack.com/pages/support?doc=/datasheets/t-series-datasheet/33-appendix-a-1-thermocouples-t-series-datasheet/>

    Example 1: Basic K-type thermocouple with internal CJC
        >>> # Most common configuration - simple and reliable
        >>> tc_chan = ThermocoupleChan(
        ...     port="AIN0",
        ...     channel=100,
        ...     thermocouple_type="K",  # K-type thermocouple
        ...     cjc_source="TEMPERATURE_DEVICE_K",  # Use device sensor
        ...     cjc_slope=1.0,    # Default for internal sensor
        ...     cjc_offset=0.0,   # Default for internal sensor
        ...     units="C",
        ...     neg_chan=199,  # Single-ended (referenced to GND)
        ...     pos_chan=0     # AIN0
        ... )

    Example 2: J-type thermocouple with external LM34 CJC sensor
        >>> # Higher accuracy using dedicated external temperature sensor
        >>> # LM34 outputs 10mV/°F, sensor connected to AIN1
        >>> tc_chan = ThermocoupleChan(
        ...     port="AIN0",
        ...     channel=100,
        ...     thermocouple_type="J",
        ...     cjc_source="AIN1",  # External sensor on AIN1
        ...     cjc_slope=55.56,    # LM34 conversion: 1°F = 0.01V, converted to K/V
        ...     cjc_offset=255.37,  # LM34 offset conversion to Kelvin
        ...     units="F",
        ...     neg_chan=199,
        ...     pos_chan=0
        ... )

    Example 3: T-type thermocouple in differential mode
        >>> # Differential measurement for better noise rejection
        >>> tc_chan = ThermocoupleChan(
        ...     port="AIN0",
        ...     channel=100,
        ...     thermocouple_type="T",
        ...     cjc_source="TEMPERATURE_DEVICE_K",
        ...     cjc_slope=1.0,
        ...     cjc_offset=0.0,
        ...     units="C",
        ...     neg_chan=1,  # AIN1 as negative (differential)
        ...     pos_chan=0   # AIN0 as positive
        ... )

    :param port: The port location of the channel (e.g., 'AIN0')
    :param channel: Synnax channel key that will receive temperature data
    :param thermocouple_type: Type of thermocouple (K is most common, J for lower temps)
    :param cjc_source: CJC temperature source - 'TEMPERATURE_DEVICE_K' (internal),
                       'TEMPERATURE_AIR_K' (air), or 'AIN#' (external sensor)
    :param cjc_slope: Slope for CJC voltage-to-temperature conversion in Kelvin/Volts
                      - Internal device sensor: 1.0 (default)
                      - LM34 sensor: 55.56 (converts 10mV/°F to K/V)
                      - Custom sensor: calculate based on sensor datasheet
    :param cjc_offset: Offset for CJC temperature in Kelvin
                       - Internal device sensor: 0.0 (default)
                       - LM34 sensor: 255.37 (converts Fahrenheit offset to Kelvin)
                       - Custom sensor: calculate based on sensor datasheet
    :param units: Temperature units for output (K, C, or F)
    :param neg_chan: Negative channel for differential mode (199 = single-ended GND reference)
    :param pos_chan: Positive channel number (0 for AIN0, 1 for AIN1, etc.)
    """

    type: Literal["TC"] = "TC"
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."
    thermocouple_type: Literal["B", "E", "J", "K", "N", "R", "S", "T", "C"]
    "The type of thermocouple being used."
    cjc_source: str = "TEMPERATURE_DEVICE_K"
    """
    The cold junction compensation (CJC) source. Options:
    - 'TEMPERATURE_DEVICE_K': Use device internal temperature sensor
    - 'TEMPERATURE_AIR_K': Use air temperature sensor
    - 'AIN#': Use another analog input (e.g., 'AIN1')
    """
    cjc_slope: float = 1.0
    """
    Slope for CJC voltage to temperature conversion (Kelvin/Volts).
    - Device temp: 1.0
    - LM34 sensor: 55.56
    """
    cjc_offset: float = 0.0
    """
    Offset for CJC temperature (Kelvin).
    - Device temp: 0.0
    - LM34 sensor: 255.37
    """
    units: Literal["K", "C", "F"] = "C"
    "Temperature units for the reading (Kelvin, Celsius, or Fahrenheit)."
    neg_chan: int = 199
    "The negative channel for differential measurements. 199 = single-ended (GND)."
    pos_chan: int = 0
    "The positive channel number (e.g., 0 for AIN0)."


class DIChan(BaseChan):
    """
    Digital Input Channel configuration for LabJack devices.

    Reads digital state (0 or 1) from a specified digital I/O line.

    For detailed information, see the LabJack documentation:
    <https://labjack.com/pages/support?doc=/datasheets/t-series-datasheet/3220-digital-io-t-series-datasheet/>
    """

    type: Literal["DI"] = "DI"
    channel: ChannelKey
    "The Synnax channel key that will be written to during acquisition."


# Union type for all input channels
InputChan = AIChan | ThermocoupleChan | DIChan


class OutputChan(BaseChan):
    """
    Output Channel configuration for LabJack devices.

    Writes analog voltage or digital state to a specified output terminal.
    Supports both analog outputs (DAC) and digital I/O lines.

    For detailed information, see the LabJack documentation:
    <https://labjack.com/pages/support?doc=/datasheets/t-series-datasheet/>
    """

    type: Literal["AO", "DO"] = "DO"
    "The type of output channel ('AO' for analog, 'DO' for digital)."
    cmd_channel: ChannelKey
    "The Synnax channel key to read command values from."
    state_channel: ChannelKey
    "The Synnax channel key to write state values to."


class ReadTaskConfig(BaseReadTaskConfig):
    """
    Configuration for a LabJack read task.

    Inherits common read task fields (sample_rate, stream_rate, data_saving,
    auto_start) from BaseReadTaskConfig and adds LabJack-specific channel configuration
    with LabJack hardware sample rate limits (100kHz max).
    """

    device: str = Field(min_length=1)
    "The key of the Synnax LabJack device to read from."
    sample_rate: conint(ge=0, le=100000)
    stream_rate: conint(ge=0, le=100000)
    channels: list[InputChan]
    "A list of input channel configurations to acquire data from."

    @field_validator("channels")
    def validate_channels_not_empty(cls, v):
        """Validate that at least one channel is provided."""
        if len(v) == 0:
            raise ValueError("Task must have at least one channel")
        return v


class WriteTaskConfig(BaseWriteTaskConfig):
    """
    Configuration for a LabJack write task.

    Inherits common write task fields (device, data_saving, auto_start) from
    BaseWriteTaskConfig and adds LabJack-specific state rate and channel configuration
    with LabJack hardware state rate limits (10kHz max).
    """

    state_rate: conint(ge=0, le=10000)
    "The rate at which to write task channel states to the Synnax cluster (Hz)."
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
    A read task for sampling data from LabJack devices and writing the data to a
    Synnax cluster. This task is a programmatic representation of the LabJack read
    task configurable within the Synnax console.

    For detailed information on configuring/operating a LabJack read task, see
    https://docs.synnaxlabs.com/reference/driver/labjack/read-task


    :param device: The key of the Synnax LabJack device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the LabJack device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster. For example, a sample rate of 100Hz and a stream rate of 25Hz will
        result in groups of 4 samples being streamed to the cluster every 40ms.
    :param data_saving: Whether to save data permanently within Synnax, or just stream
        it for real-time consumption.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: A list of input channel configurations (InputChan subtypes:
        AIChan, ThermocoupleChan, DIChan).
    """

    TYPE = "labjack_read"
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
        """Internal: Update device properties before task configuration."""
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
            # Map port location -> channel key for Console
            props["read"]["channels"][ch.port] = ch.channel

        dev.properties = json.dumps(props)
        device_client.create(dev)


class WriteTask(StarterStopperMixin, JSONConfigMixin, MetaTask):
    """
    A write task for sending commands to LabJack devices. This task is a programmatic
    representation of the LabJack write task configurable within the Synnax console.

    For detailed information on configuring/operating a LabJack write task, see
    https://docs.synnaxlabs.com/reference/driver/labjack/write-task


    :param device: The key of the Synnax LabJack device to write to.
    :param name: A human-readable name for the task.
    :param state_rate: The rate at which to write task channel states to the Synnax
        cluster.
    :param data_saving: Whether to save data permanently within Synnax, or just stream
        it for real-time consumption.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: A list of output channel configurations (OutputChan).
    """

    TYPE = "labjack_write"
    config: WriteTaskConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        device: str = "",
        name: str = "",
        state_rate: CrudeRate = 0,
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
            state_rate=state_rate,
            data_saving=data_saving,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )

    def _update_device_properties(self, device_client):
        """Internal: Update device properties before task configuration."""
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
            # Map port location -> state_channel key for Console
            props["write"]["channels"][ch.port] = ch.state_channel

        dev.properties = json.dumps(props)
        device_client.create(dev)


def device_props(
    identifier: str,
    connection_type: str = "ANY",
) -> dict:
    """
    Create device properties for a LabJack connection.

    Args:
        identifier: Device identifier (serial number, IP address, or device name)
        connection_type: Connection method - "ANY", "USB", "TCP", "ETHERNET", or "WIFI"

    Returns:
        Dictionary of device properties with the correct structure for Console
    """
    return {
        "connection": {
            "identifier": identifier,
            "connection_type": connection_type,
        },
        "read": {"index": 0, "channels": {}},
        "write": {"channels": {}},
    }


def create_device(client, model: str, **kwargs):
    """
    Create a LabJack device with make, model, and key automatically set.

    This is a thin wrapper around client.hardware.devices.create() that
    automatically fills in:
    - make: "LabJack"
    - model: Specified model (T4, T7, or T8)
    - key: auto-generated UUID if not provided

    All other parameters are passed through unchanged.

    Example:
        >>> import json
        >>> from synnax.hardware import labjack
        >>> device = labjack.create_device(
        ...     client=client,
        ...     model=labjack.T7,
        ...     name="My LabJack T7",
        ...     location="USB",
        ...     rack=rack.key,
        ...     properties=json.dumps(labjack.device_props(identifier="ANY"))
        ... )

    Args:
        client: Synnax client instance
        model: LabJack model (use module constants: T4, T7, T8)
        **kwargs: Additional arguments passed to client.hardware.devices.create()
    """
    from uuid import uuid4

    # Validate model
    valid_models = [T4, T7, T8]
    if model not in valid_models:
        raise ValueError(f"Invalid model '{model}'. Must be one of: {valid_models}")

    # Auto-generate key if not provided
    if "key" not in kwargs:
        kwargs["key"] = str(uuid4())

    kwargs["make"] = MAKE
    kwargs["model"] = model
    return client.hardware.devices.create(**kwargs)
