#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import ValidationError

import synnax as sy
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
                            "neg_chan": 199,
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
                            "neg_chan": 199,
                            "pos_chan": 0,
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
                            "neg_chan": 199,
                            "scale": {"type": "none"},
                        },
                        {
                            "type": "analog",
                            "key": "ai-2",
                            "disabled": False,
                            "port": "AIN1",
                            "channel": 2000,
                            "range": 1.0,
                            "neg_chan": 199,
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
    def test_parse_labjack_read_task(self, test_data):
        """Test that ReadConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.labjack.ReadConfig.model_validate(input_data)

    def test_read_config_defaults(self):
        """Test that ReadConfig applies the shared task config defaults."""
        config = sy.labjack.ReadConfig(device="test-device")
        assert config.sample_rate == sy.Rate(10)
        assert config.stream_rate == sy.Rate(5)
        assert config.data_saving_disabled is False
        assert config.auto_start is False
        assert config.channels == []

    def test_read_task_auto_key_generation(self):
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

    def test_thermocouple_type_validation(self):
        """Test that thermocouple types are validated."""
        # Valid thermocouple types
        for tc_type in ["B", "E", "J", "K", "N", "R", "S", "T", "C"]:
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
        with pytest.raises(ValidationError):
            sy.labjack.ThermocoupleReadChannel(
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

    def test_create_and_retrieve_read_task(self, client: sy.Synnax):
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
    def test_parse_labjack_write_task(self, test_data):
        """Test that WriteConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.labjack.WriteConfig.model_validate(input_data)

    def test_write_task_disabled_channels(self):
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

    def test_write_channel_auto_key_generation(self):
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

    def test_create_and_retrieve_write_task(self, client: sy.Synnax):
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
        created_task = client.tasks.create(
            name="test-labjack-write-task",
            type="labjack_write",
            config=task.config,
        )
        sy.labjack.WriteTask(created_task)

    def test_write_task_serialization_round_trip(self, client: sy.Synnax):
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
            original_task.config.channels, retrieved_task.config.channels
        ):
            assert retr_ch.key == orig_ch.key
            assert retr_ch.type == orig_ch.type
            assert retr_ch.port == orig_ch.port
            assert retr_ch.cmd_channel == orig_ch.cmd_channel
            assert retr_ch.state_channel == orig_ch.state_channel
            assert retr_ch.disabled == orig_ch.disabled


@pytest.mark.labjack
class TestLabJackDevicePropertyUpdates:
    """Tests that device properties are correctly updated with channel mappings."""

    def test_read_task_updates_device_properties(self, client: sy.Synnax):
        """Test that configuring a ReadTask updates device properties with channel mappings."""
        # Create a rack
        rack = client.racks.retrieve_embedded_rack()

        # Create a device
        device = sy.labjack.Device(
            model=sy.labjack.T7,
            identifier="ANY",
            name="Test LabJack T7",
            location="USB",
            rack=rack.key,
            connection_type="ANY",
        )

        client.devices.create(device)

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
            ],
        )

        # Trigger device property update
        task.update_device_properties(client.devices)

        # Retrieve device and check properties
        updated_device = client.devices.retrieve(key=device.key)
        props = updated_device.properties

        # Verify read.channels mapping exists
        assert "read" in props
        assert "channels" in props["read"]

        # Verify port -> channel key mapping
        channels = props["read"]["channels"]
        assert "AIN0" in channels
        assert channels["AIN0"] == ch1.key
        assert "FIO4" in channels
        assert channels["FIO4"] == ch2.key

    def test_write_task_updates_device_properties(self, client: sy.Synnax):
        """Test that configuring a WriteTask updates device properties with channel mappings."""
        # Create a rack
        rack = client.racks.retrieve_embedded_rack()

        # Create a device
        device = sy.labjack.Device(
            model=sy.labjack.T7,
            identifier="ANY",
            name="Test LabJack Write T7",
            location="USB",
            rack=rack.key,
            connection_type="ANY",
        )

        client.devices.create(device)

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

        # Verify write.channels mapping exists
        assert "write" in props
        assert "channels" in props["write"]

        # Verify port -> state_channel key mapping
        channels = props["write"]["channels"]
        assert "DAC0" in channels
        assert channels["DAC0"] == dac0_state.key
        assert "FIO4" in channels
        assert channels["FIO4"] == fio4_state.key
