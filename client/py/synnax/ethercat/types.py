#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

from pydantic import BaseModel, Field

from synnax import device, task
from synnax.ethercat.types_gen import (
    ReadChannel,
    ReadConfig,
    WriteChannel,
    WriteConfig,
)
from synnax.telem import CrudeRate, Rate

# Device identifiers - must match driver expectations
MAKE = "EtherCAT"
MODEL = "Slave"


class PDOEntry(BaseModel):
    """A single PDO entry stored in slave device properties after scanning.

    :param name: Human-readable name of the PDO entry (e.g. "Position actual value").
    :param pdo_index: Parent PDO index (e.g. 0x1A00 for TxPDO, 0x1600 for RxPDO).
    :param index: CoE object dictionary index (e.g. 0x6064 = 24676).
    :param sub_index: CoE object dictionary subindex.
    :param bit_length: Size of the data in bits.
    :param data_type: Data type string (e.g. "uint16", "int32", "float32").
    """

    name: str
    pdo_index: int = Field(default=0, ge=0, le=65535)
    index: int = Field(ge=0, le=65535)
    sub_index: int = Field(ge=0, le=255)
    bit_length: int = Field(ge=1, le=64)
    data_type: str


class ReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A read task for sampling data from EtherCAT slave devices.

    The task supports both automatic channels (PDO resolved by name) and manual
    channels (PDO address specified inline).

    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from slaves (Hz).
    :param stream_rate: The rate at which data is streamed to Synnax (Hz).
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when configured.
    :param channels: The input channels to acquire data from (ReadChannel variants:
        AutomaticReadChannel, ManualReadChannel).
    """

    TYPE = "ethercat_read"
    config: ReadConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
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
            sample_rate=Rate(sample_rate),
            stream_rate=Rate(stream_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)


class WriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A write task for sending commands to EtherCAT slave devices.

    If a channel specifies a state_channel, the current PDO value is written back to
    Synnax at the configured state_rate.

    :param name: A human-readable name for the task.
    :param state_rate: Rate at which state feedback is written to Synnax (Hz).
    :param execution_rate: Rate at which commands are executed on the bus (Hz).
    :param data_saving_disabled: Whether to only stream state data for real-time
        consumption instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when configured.
    :param channels: The output channels to write to (WriteChannel variants:
        AutomaticWriteChannel, ManualWriteChannel).
    """

    TYPE = "ethercat_write"
    config: WriteConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        name: str = "",
        state_rate: CrudeRate = 25,
        execution_rate: CrudeRate = 1000,
        data_saving_disabled: bool = False,
        auto_start: bool = False,
        channels: list[WriteChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            state_rate=Rate(state_rate),
            execution_rate=Rate(execution_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)


class Device(device.Device):
    """An EtherCAT slave device.

    Typically, EtherCAT devices are created automatically by the driver's scan task.
    This class can be used to create devices manually or to update device properties.

    :param name: Human-readable name for the device.
    :param network: Network interface name (e.g. "eth0", "enp3s0").
    :param position: Position of the slave on the EtherCAT bus, 0-indexed.
    :param vendor_id: EtherCAT vendor ID from device EEPROM.
    :param product_code: Product code identifying the device model.
    :param revision: Hardware/firmware revision number.
    :param serial: Unique serial number from device EEPROM.
    :param rack: Rack key this device belongs to.
    :param input_pdos: Input PDO entry definitions (TxPDO, slave to master).
    :param output_pdos: Output PDO entry definitions (RxPDO, master to slave).
    :param key: Unique key for the device. Auto-generated if empty.
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
            properties=props,
        )
