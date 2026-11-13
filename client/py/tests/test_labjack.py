#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable, Iterator
from typing import ClassVar, cast
from uuid import UUID

import pytest
from pydantic import TypeAdapter, ValidationError

import synnax as sy
import synnax.labjack.device
from tests.json_value import obj
from tests.task_targets import TaskTargets
from x.strings import random_name


@pytest.mark.labjack
class TestLabJackReadTask:
    """Tests for LabJack Read Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_analog_input",
                "data": {
                    "device": "labjack-device-key",
                    "sample_rate": 100,
                    "stream_rate": 25,
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "analog",
                            "key": "ai-1",
                            "disabled": False,
                            "port": "AIN0",
                            "channel": 1234,
                            "range": 10.0,
                            "scale": {"type": "none"},
                        },
                    ],
                },
            },
            {
                "name": "thermocouple_k_type",
                "data": {
                    "device": "labjack-device-key",
                    "sample_rate": 10,
                    "stream_rate": 10,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "thermocouple",
                            "key": "tc-1",
                            "disabled": False,
                            "port": "AIN0",
                            "channel": 5678,
                            "thermocouple_type": "K",
                            "cjc_source": "TEMPERATURE_DEVICE_K",
                            "cjc_slope": 1.0,
                            "cjc_offset": 0.0,
                            "units": "C",
                            "scale": {"type": "none"},
                        },
                    ],
                },
            },
            {
                "name": "digital_input",
                "data": {
                    "device": "labjack-device-key",
                    "sample_rate": 50,
                    "stream_rate": 25,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "digital",
                            "key": "di-1",
                            "disabled": False,
                            "port": "FIO4",
                            "channel": 9012,
                        },
                    ],
                },
            },
            {
                "name": "mixed_channels",
                "data": {
                    "device": "labjack-device-key",
                    "sample_rate": 1000,
                    "stream_rate": 250,
                    "data_saving_disabled": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "analog",
                            "key": "ai-1",
                            "disabled": False,
                            "port": "AIN0",
                            "channel": 1000,
                            "range": 10.0,
                            "scale": {"type": "none"},
                        },
                        {
                            "type": "analog",
                            "key": "ai-2",
                            "disabled": False,
                            "port": "AIN1",
                            "channel": 2000,
                            "range": 1.0,
                            "scale": {"type": "none"},
                        },
                        {
                            "type": "digital",
                            "key": "di-1",
                            "disabled": False,
                            "port": "FIO5",
                            "channel": 3000,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_labjack_read_task(self, test_data: dict[str, object]) -> None:
        """Test that ReadConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.labjack.ReadConfig.model_validate(input_data)

    def test_read_config_defaults(self) -> None:
        """Test that ReadConfig applies the shared task config defaults."""
        config = sy.labjack.ReadConfig(device="test-device")
        assert config.sample_rate == sy.Rate(10)
        assert config.stream_rate == sy.Rate(5)
        assert config.data_saving_disabled is False
        assert config.auto_start is False
        assert config.channels == []

    def test_read_task_auto_key_generation(self) -> None:
        """Test that the ReadTask assigns keys to channels missing one."""
        task = sy.labjack.ReadTask(
            name="test",
            device="test-device",
            channels=[
                sy.labjack.AnalogReadChannel(
                    type="analog",
                    port="AIN0",
                    channel=1234,
                    range=10.0,
                    scale=sy.labjack.NoneScale(type="none"),
                )
            ],
        )
        channel = task.config.channels[0]
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_thermocouple_type_validation(self) -> None:
        """Test that thermocouple types are validated."""
        # Valid thermocouple types
        tc_types: list[sy.labjack.ThermocoupleType] = [
            "B",
            "E",
            "J",
            "K",
            "N",
            "R",
            "S",
            "T",
            "C",
        ]
        for tc_type in tc_types:
            sy.labjack.ThermocoupleReadChannel(
                type="thermocouple",
                port="AIN0",
                channel=1234,
                thermocouple_type=tc_type,
                cjc_source="TEMPERATURE_DEVICE_K",
                cjc_slope=1.0,
                cjc_offset=0.0,
                units="C",
                scale=sy.labjack.NoneScale(type="none"),
            )

        # Invalid thermocouple type
        channel = cast(Callable[..., object], sy.labjack.ThermocoupleReadChannel)
        with pytest.raises(ValidationError):
            channel(
                type="thermocouple",
                port="AIN0",
                channel=1234,
                thermocouple_type="InvalidType",
                cjc_source="TEMPERATURE_DEVICE_K",
                cjc_slope=1.0,
                cjc_offset=0.0,
                units="C",
                scale=sy.labjack.NoneScale(type="none"),
            )

    def test_create_and_retrieve_read_task(
        self, client: sy.Synnax, task_targets: TaskTargets
    ) -> None:
        """Test that ReadTask can be created and retrieved from the database."""
        task = sy.labjack.ReadTask(
            name="test-labjack-read-task",
            device="some-device-key",
            sample_rate=100,
            stream_rate=25,
            data_saving_disabled=True,
            auto_start=False,
            channels=[
                sy.labjack.AnalogReadChannel(
                    type="analog",
                    key="ai-1",
                    port="AIN0",
                    channel=1234,
                    range=10.0,
                    scale=sy.labjack.NoneScale(type="none"),
                ),
                sy.labjack.DigitalReadChannel(
                    type="digital",
                    key="di-1",
                    port="FIO4",
                    channel=5678,
                ),
            ],
        )
        task_targets.attach(task.config)
        created_task = client.tasks.create(
            name="test-labjack-read-task",
            type="labjack_read",
            config=task.config,
        )
        sy.labjack.ReadTask(created_task)


@pytest.mark.labjack
class TestLabJackWriteTask:
    """Tests for LabJack Write Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_analog_output",
                "data": {
                    "device": "labjack-device-key",
                    "state_rate": 20,
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "analog",
                            "key": "ao-1",
                            "disabled": False,
                            "port": "DAC0",
                            "cmd_channel": 1234,
                            "state_channel": 1235,
                        },
                    ],
                },
            },
            {
                "name": "basic_digital_output",
                "data": {
                    "device": "labjack-device-key",
                    "state_rate": 10,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "digital",
                            "key": "do-1",
                            "disabled": False,
                            "port": "FIO4",
                            "cmd_channel": 5678,
                            "state_channel": 5679,
                        },
                    ],
                },
            },
            {
                "name": "mixed_outputs",
                "data": {
                    "device": "labjack-device-key",
                    "state_rate": 50,
                    "data_saving_disabled": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "analog",
                            "key": "ao-1",
                            "disabled": False,
                            "port": "DAC0",
                            "cmd_channel": 1000,
                            "state_channel": 1001,
                        },
                        {
                            "type": "analog",
                            "key": "ao-2",
                            "disabled": True,
                            "port": "DAC1",
                            "cmd_channel": 2000,
                            "state_channel": 2001,
                        },
                        {
                            "type": "digital",
                            "key": "do-1",
                            "disabled": False,
                            "port": "FIO4",
                            "cmd_channel": 3000,
                            "state_channel": 3001,
                        },
                        {
                            "type": "digital",
                            "key": "do-2",
                            "disabled": False,
                            "port": "FIO5",
                            "cmd_channel": 4000,
                            "state_channel": 4001,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_labjack_write_task(self, test_data: dict[str, object]) -> None:
        """Test that WriteConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.labjack.WriteConfig.model_validate(input_data)

    def test_write_task_disabled_channels(self) -> None:
        """Test that disabled channels are handled correctly."""
        config = sy.labjack.WriteConfig(
            device="test-device",
            state_rate=sy.Rate(20),
            auto_start=False,
            channels=[
                sy.labjack.DigitalWriteChannel(
                    type="digital",
                    port="FIO4",
                    cmd_channel=1234,
                    state_channel=1235,
                    disabled=False,
                ),
                sy.labjack.DigitalWriteChannel(
                    type="digital",
                    port="FIO5",
                    cmd_channel=5678,
                    state_channel=5679,
                    disabled=True,
                ),
            ],
        )
        assert len(config.channels) == 2
        assert config.channels[0].disabled is False
        assert config.channels[1].disabled is True

    def test_write_channel_auto_key_generation(self) -> None:
        """Test that the WriteTask assigns keys to channels missing one."""
        task = sy.labjack.WriteTask(
            name="test",
            device="test-device",
            state_rate=20,
            channels=[
                sy.labjack.AnalogWriteChannel(
                    type="analog",
                    port="DAC0",
                    cmd_channel=1234,
                    state_channel=1235,
                )
            ],
        )
        channel = task.config.channels[0]
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_create_and_retrieve_write_task(
        self, client: sy.Synnax, task_targets: TaskTargets
    ) -> None:
        """Test that WriteTask can be created and retrieved from the database."""
        task = sy.labjack.WriteTask(
            name="test-labjack-write-task",
            device="some-device-key",
            state_rate=20,
            data_saving_disabled=False,
            auto_start=False,
            channels=[
                sy.labjack.AnalogWriteChannel(
                    key="ao-1",
                    type="analog",
                    port="DAC0",
                    cmd_channel=1234,
                    state_channel=1235,
                ),
                sy.labjack.DigitalWriteChannel(
                    key="do-1",
                    type="digital",
                    port="FIO4",
                    cmd_channel=5678,
                    state_channel=5679,
                ),
            ],
        )
        task_targets.attach(task.config)
        created_task = client.tasks.create(
            name="test-labjack-write-task",
            type="labjack_write",
            config=task.config,
        )
        sy.labjack.WriteTask(created_task)

    def test_write_task_serialization_round_trip(
        self, client: sy.Synnax, task_targets: TaskTargets
    ) -> None:
        """Test that task can be serialized and deserialized correctly."""
        original_task = sy.labjack.WriteTask(
            name="test-round-trip",
            device="some-device-key",
            state_rate=20,
            data_saving_disabled=False,
            auto_start=False,
            channels=[
                sy.labjack.AnalogWriteChannel(
                    key="ao-1",
                    type="analog",
                    port="DAC0",
                    cmd_channel=1234,
                    state_channel=1235,
                    disabled=False,
                ),
                sy.labjack.DigitalWriteChannel(
                    key="do-1",
                    type="digital",
                    port="FIO4",
                    cmd_channel=5678,
                    state_channel=5679,
                    disabled=True,
                ),
            ],
        )

        task_targets.attach(original_task.config)
        created_task = client.tasks.create(
            name="test-round-trip",
            type="labjack_write",
            config=original_task.config,
        )

        retrieved_task = sy.labjack.WriteTask(created_task)
        assert retrieved_task.config.device == original_task.config.device
        assert retrieved_task.config.state_rate == original_task.config.state_rate
        assert (
            retrieved_task.config.data_saving_disabled
            == original_task.config.data_saving_disabled
        )
        assert retrieved_task.config.auto_start == original_task.config.auto_start
        assert len(retrieved_task.config.channels) == len(original_task.config.channels)

        for orig_ch, retr_ch in zip(
            original_task.config.channels, retrieved_task.config.channels, strict=True
        ):
            assert retr_ch.key == orig_ch.key
            assert retr_ch.type == orig_ch.type
            assert retr_ch.port == orig_ch.port
            assert retr_ch.cmd_channel == orig_ch.cmd_channel
            assert retr_ch.state_channel == orig_ch.state_channel
            assert retr_ch.disabled == orig_ch.disabled


@pytest.mark.labjack
class TestLabJackDevice:
    """Tests for the LabJack device constructor."""

    def test_key_is_serial_number(self) -> None:
        """Should key the device by its serial number, as the Driver's scanner does."""
        device = sy.labjack.Device(model=sy.labjack.T8, serial_number=480010003)
        assert device.key == "480010003"
        assert device.make == "LabJack"
        assert device.model == "LJM_dtT8"

    def test_invalid_model(self) -> None:
        """Should reject a model that Synnax does not support."""
        with pytest.raises(ValueError, match="Invalid model 'LJM_dtDIGIT'"):
            sy.labjack.Device(
                model=cast(sy.labjack.SUPPORTED_MODELS, "LJM_dtDIGIT"),
                serial_number=1,
            )


@pytest.mark.labjack
class TestLabJackDevicePropertyUpdates:
    """Tests that device properties are correctly updated with channel mappings."""

    @pytest.fixture
    def device(self, client: sy.Synnax) -> Iterator[sy.labjack.Device]:
        """A T7 on the embedded rack, deleted after the test."""
        dev = sy.labjack.Device(
            model=sy.labjack.T7,
            serial_number=470000001,
            name="Test LabJack T7",
            location="LJM_ctUSB",
            rack=client.racks.retrieve_embedded_rack().key,
        )
        client.devices.create(dev)
        yield dev
        client.devices.delete(keys=[dev.key])

    def test_read_task_updates_device_properties(
        self, client: sy.Synnax, device: sy.labjack.Device
    ) -> None:
        """Test that configuring a ReadTask updates device properties with channel
        mappings.
        """
        # Create channels
        suffix = random_name()
        time_ch = client.channels.create(
            name=f"labjack_time_{suffix}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )

        ch1 = client.channels.create(
            name=f"ain0_{suffix}",
            data_type=sy.DataType.FLOAT32,
            index=time_ch.key,
        )

        ch2 = client.channels.create(
            name=f"fio4_{suffix}",
            data_type=sy.DataType.UINT8,
            index=time_ch.key,
        )

        ch3 = client.channels.create(
            name=f"mio1_{suffix}",
            data_type=sy.DataType.UINT8,
            index=time_ch.key,
        )

        # Create task with multiple channel types
        task = sy.labjack.ReadTask(
            name="Test Read Task",
            device=device.key,
            sample_rate=100,
            stream_rate=25,
            channels=[
                sy.labjack.AnalogReadChannel(
                    type="analog",
                    port="AIN0",
                    channel=ch1.key,
                    range=10.0,
                    scale=sy.labjack.NoneScale(type="none"),
                ),
                sy.labjack.DigitalReadChannel(
                    type="digital",
                    port="FIO4",
                    channel=ch2.key,
                ),
                sy.labjack.DigitalReadChannel(
                    type="digital",
                    port="MIO1",
                    channel=ch3.key,
                ),
            ],
        )

        # Trigger device property update
        task.update_device_properties(client.devices)

        # Retrieve device and check properties
        updated_device = client.devices.retrieve(key=device.key)
        props = updated_device.properties

        assert obj(props, "AI", "channels") == {"AIN0": ch1.key}
        assert obj(props, "DI", "channels") == {"DIO4": ch2.key, "DIO21": ch3.key}

    def test_write_task_updates_device_properties(
        self, client: sy.Synnax, device: sy.labjack.Device
    ) -> None:
        """Test that configuring a WriteTask updates device properties with channel
        mappings.
        """
        suffix = random_name()

        # Create command and state channels
        cmd_time = client.channels.create(
            name=f"cmd_time_{suffix}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )

        state_time = client.channels.create(
            name=f"state_time_{suffix}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )

        dac0_cmd = client.channels.create(
            name=f"dac0_command_{suffix}",
            data_type=sy.DataType.FLOAT32,
            index=cmd_time.key,
        )

        dac0_state = client.channels.create(
            name=f"dac0_state_{suffix}",
            data_type=sy.DataType.FLOAT32,
            index=state_time.key,
        )

        fio4_cmd = client.channels.create(
            name=f"fio4_command_{suffix}",
            data_type=sy.DataType.UINT8,
            index=cmd_time.key,
        )

        fio4_state = client.channels.create(
            name=f"fio4_state_{suffix}",
            data_type=sy.DataType.UINT8,
            index=state_time.key,
        )

        # Create write task
        task = sy.labjack.WriteTask(
            name="Test Write Task",
            device=device.key,
            state_rate=20,
            channels=[
                sy.labjack.AnalogWriteChannel(
                    type="analog",
                    port="DAC0",
                    cmd_channel=dac0_cmd.key,
                    state_channel=dac0_state.key,
                ),
                sy.labjack.DigitalWriteChannel(
                    type="digital",
                    port="FIO4",
                    cmd_channel=fio4_cmd.key,
                    state_channel=fio4_state.key,
                ),
            ],
        )

        # Trigger device property update
        task.update_device_properties(client.devices)

        # Retrieve device and check properties
        updated_device = client.devices.retrieve(key=device.key)
        props = updated_device.properties

        assert obj(props, "AO", "channels") == {
            "DAC0": {"command": dac0_cmd.key, "state": dac0_state.key}
        }
        assert obj(props, "DO", "channels") == {
            "DIO4": {"command": fio4_cmd.key, "state": fio4_state.key}
        }


@pytest.mark.labjack
class TestLabJackDeviceResource:
    """Tests for the typed LabJack device resource."""

    KEY = UUID("11111111-1111-4111-8111-111111111111")
    DEVICE: ClassVar[dict[str, object]] = {
        "rack": 1,
        "name": "T7-0001",
        "serial": 470010001,
        "connection": "LJM_ctUSB",
        "model": "t7",
    }

    def test_parses_the_model_variant(self) -> None:
        """Should parse the variant that the model names and leave it unconfigured."""
        device = TypeAdapter(sy.labjack.device.Device).validate_python(self.DEVICE)
        assert isinstance(device, sy.labjack.device.T7Device)
        assert device.identifier == ""
        assert device.read_index == 0

    @pytest.mark.parametrize(
        "change",
        [
            {"model": "t9"},
            {"connection": "LJM_ctWIFI"},
            {"serial": None},
        ],
    )
    def test_rejects_an_invalid_device(self, change: dict[str, object]) -> None:
        """Should reject an unknown model or connection and a missing serial."""
        data = {k: v for k, v in {**self.DEVICE, **change}.items() if v is not None}
        with pytest.raises(ValidationError):
            TypeAdapter(sy.labjack.device.Device).validate_python(data)

    def test_resource_id(self) -> None:
        """Should name the device by its key under the labjack_device type."""
        assert sy.labjack.device.resource_id(self.KEY).type == "labjack_device"
        assert sy.labjack.device.resource_id(self.KEY).key == str(self.KEY)
