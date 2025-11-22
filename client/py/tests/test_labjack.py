#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import pytest
from pydantic import ValidationError

import synnax as sy
from synnax.hardware.labjack import (
    T7,
    AIChan,
    DIChan,
    OutputChan,
    ReadTask,
    ReadTaskConfig,
    ThermocoupleChan,
    WriteTask,
    WriteTaskConfig,
)


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
                    "data_saving": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "AI",
                            "key": "ai-1",
                            "enabled": True,
                            "port": "AIN0",
                            "channel": 1234,
                            "range": 10.0,
                            "neg_chan": 199,
                            "pos_chan": 0,
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
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "TC",
                            "key": "tc-1",
                            "enabled": True,
                            "port": "AIN0",
                            "channel": 5678,
                            "thermocouple_type": "K",
                            "cjc_source": "TEMPERATURE_DEVICE_K",
                            "cjc_slope": 1.0,
                            "cjc_offset": 0.0,
                            "units": "C",
                            "neg_chan": 199,
                            "pos_chan": 0,
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
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "DI",
                            "key": "di-1",
                            "enabled": True,
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
                    "data_saving": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "AI",
                            "key": "ai-1",
                            "enabled": True,
                            "port": "AIN0",
                            "channel": 1000,
                            "range": 10.0,
                            "neg_chan": 199,
                            "pos_chan": 0,
                        },
                        {
                            "type": "AI",
                            "key": "ai-2",
                            "enabled": True,
                            "port": "AIN1",
                            "channel": 2000,
                            "range": 1.0,
                            "neg_chan": 199,
                            "pos_chan": 1,
                        },
                        {
                            "type": "DI",
                            "key": "di-1",
                            "enabled": True,
                            "port": "FIO5",
                            "channel": 3000,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_labjack_read_task(self, test_data):
        """Test that ReadTaskConfig can parse various channel configurations."""
        input_data = test_data["data"]
        ReadTaskConfig.model_validate(input_data)

    def test_read_task_stream_rate_validation(self):
        """Test that stream_rate cannot exceed sample_rate."""
        with pytest.raises(ValidationError) as exc_info:
            ReadTaskConfig(
                device="test-device",
                sample_rate=10,
                stream_rate=20,  # Invalid: greater than sample_rate
                data_saving=False,
                auto_start=False,
                channels=[
                    AIChan(
                        port="AIN0",
                        channel=1234,
                        range=10.0,
                    )
                ],
            )
        assert "stream rate" in str(exc_info.value).lower()

    def test_read_task_empty_channels(self):
        """Test that empty channel list raises validation error."""
        with pytest.raises(ValidationError) as exc_info:
            ReadTaskConfig(
                device="test-device",
                sample_rate=100,
                stream_rate=25,
                data_saving=False,
                auto_start=False,
                channels=[],  # Empty list
            )
        assert "at least one channel" in str(exc_info.value).lower()

    def test_read_task_auto_key_generation(self):
        """Test that channels auto-generate keys if not provided."""
        channel = AIChan(
            port="AIN0",
            channel=1234,
            range=10.0,
        )
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_read_task_sample_rate_bounds(self):
        """Test that sample rate validation works (0-100000 Hz)."""
        # Valid sample rates
        ReadTaskConfig(
            device="test-device",
            sample_rate=1,
            stream_rate=1,
            data_saving=False,
            channels=[AIChan(port="AIN0", channel=1234, range=10.0)],
        )
        ReadTaskConfig(
            device="test-device",
            sample_rate=100000,
            stream_rate=100000,
            data_saving=False,
            channels=[AIChan(port="AIN0", channel=1234, range=10.0)],
        )

        # Invalid sample rates
        with pytest.raises(ValidationError):
            ReadTaskConfig(
                device="test-device",
                sample_rate=-1,
                stream_rate=1,
                data_saving=False,
                channels=[AIChan(port="AIN0", channel=1234, range=10.0)],
            )
        with pytest.raises(ValidationError):
            ReadTaskConfig(
                device="test-device",
                sample_rate=100001,
                stream_rate=100001,
                data_saving=False,
                channels=[AIChan(port="AIN0", channel=1234, range=10.0)],
            )

    def test_thermocouple_type_validation(self):
        """Test that thermocouple types are validated."""
        # Valid thermocouple types
        for tc_type in ["B", "E", "J", "K", "N", "R", "S", "T", "C"]:
            ThermocoupleChan(
                port="AIN0",
                channel=1234,
                thermocouple_type=tc_type,
                cjc_source="TEMPERATURE_DEVICE_K",
                cjc_slope=1.0,
                cjc_offset=0.0,
                units="C",
            )

        # Invalid thermocouple type
        with pytest.raises(ValidationError):
            ThermocoupleChan(
                port="AIN0",
                channel=1234,
                thermocouple_type="InvalidType",
                cjc_source="TEMPERATURE_DEVICE_K",
                cjc_slope=1.0,
                cjc_offset=0.0,
                units="C",
            )

    def test_create_and_retrieve_read_task(self, client: sy.Synnax):
        """Test that ReadTask can be created and retrieved from the database."""
        task = ReadTask(
            name="test-labjack-read-task",
            device="some-device-key",
            sample_rate=100,
            stream_rate=25,
            data_saving=False,
            auto_start=False,
            channels=[
                AIChan(
                    key="ai-1",
                    port="AIN0",
                    channel=1234,
                    range=10.0,
                ),
                DIChan(
                    key="di-1",
                    port="FIO4",
                    channel=5678,
                ),
            ],
        )
        created_task = client.hardware.tasks.create(
            name="test-labjack-read-task",
            type="labjack_read",
            config=task.config.model_dump_json(),
        )
        ReadTask(created_task)


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
                    "data_saving": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "AO",
                            "key": "ao-1",
                            "enabled": True,
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
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "DO",
                            "key": "do-1",
                            "enabled": True,
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
                    "data_saving": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "AO",
                            "key": "ao-1",
                            "enabled": True,
                            "port": "DAC0",
                            "cmd_channel": 1000,
                            "state_channel": 1001,
                        },
                        {
                            "type": "AO",
                            "key": "ao-2",
                            "enabled": False,
                            "port": "DAC1",
                            "cmd_channel": 2000,
                            "state_channel": 2001,
                        },
                        {
                            "type": "DO",
                            "key": "do-1",
                            "enabled": True,
                            "port": "FIO4",
                            "cmd_channel": 3000,
                            "state_channel": 3001,
                        },
                        {
                            "type": "DO",
                            "key": "do-2",
                            "enabled": True,
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
        """Test that WriteTaskConfig can parse various channel configurations."""
        input_data = test_data["data"]
        WriteTaskConfig.model_validate(input_data)

    def test_write_task_empty_channels(self):
        """Test that empty channel list raises validation error."""
        with pytest.raises(ValidationError) as exc_info:
            WriteTaskConfig(
                device="test-device",
                state_rate=20,
                data_saving=False,
                auto_start=False,
                channels=[],
            )
        assert "at least one channel" in str(exc_info.value).lower()

    def test_write_task_disabled_channels(self):
        """Test that disabled channels are handled correctly."""
        config = WriteTaskConfig(
            device="test-device",
            state_rate=20,
            data_saving=False,
            auto_start=False,
            channels=[
                OutputChan(
                    type="DO",
                    port="FIO4",
                    cmd_channel=1234,
                    state_channel=1235,
                    enabled=True,
                ),
                OutputChan(
                    type="DO",
                    port="FIO5",
                    cmd_channel=5678,
                    state_channel=5679,
                    enabled=False,
                ),
            ],
        )
        assert len(config.channels) == 2
        assert config.channels[0].enabled is True
        assert config.channels[1].enabled is False

    def test_write_channel_auto_key_generation(self):
        """Test that OutputChan auto-generates a key if not provided."""
        channel = OutputChan(
            port="DAC0",
            cmd_channel=1234,
            state_channel=1235,
        )
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_create_and_retrieve_write_task(self, client: sy.Synnax):
        """Test that WriteTask can be created and retrieved from the database."""
        task = WriteTask(
            name="test-labjack-write-task",
            device="some-device-key",
            state_rate=20,
            data_saving=True,
            auto_start=False,
            channels=[
                OutputChan(
                    key="ao-1",
                    type="AO",
                    port="DAC0",
                    cmd_channel=1234,
                    state_channel=1235,
                ),
                OutputChan(
                    key="do-1",
                    type="DO",
                    port="FIO4",
                    cmd_channel=5678,
                    state_channel=5679,
                ),
            ],
        )
        created_task = client.hardware.tasks.create(
            name="test-labjack-write-task",
            type="labjack_write",
            config=task.config.model_dump_json(),
        )
        WriteTask(created_task)

    def test_write_task_serialization_round_trip(self, client: sy.Synnax):
        """Test that task can be serialized and deserialized correctly."""
        original_task = WriteTask(
            name="test-round-trip",
            device="some-device-key",
            state_rate=20,
            data_saving=True,
            auto_start=False,
            channels=[
                OutputChan(
                    key="ao-1",
                    type="AO",
                    port="DAC0",
                    cmd_channel=1234,
                    state_channel=1235,
                    enabled=True,
                ),
                OutputChan(
                    key="do-1",
                    type="DO",
                    port="FIO4",
                    cmd_channel=5678,
                    state_channel=5679,
                    enabled=False,
                ),
            ],
        )

        # Serialize to JSON
        config_json = original_task.config.model_dump_json()

        # Create task in database
        created_task = client.hardware.tasks.create(
            name="test-round-trip",
            type="labjack_write",
            config=config_json,
        )

        # Deserialize from database
        retrieved_task = WriteTask(created_task)

        # Verify all fields match
        assert retrieved_task.config.device == original_task.config.device
        assert retrieved_task.config.state_rate == original_task.config.state_rate
        assert retrieved_task.config.data_saving == original_task.config.data_saving
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
            assert retr_ch.enabled == orig_ch.enabled


@pytest.mark.labjack
class TestLabJackDevicePropertyUpdates:
    """Tests that device properties are correctly updated with channel mappings."""

    def test_read_task_updates_device_properties(self, client: sy.Synnax):
        """Test that configuring a ReadTask updates device properties with channel mappings."""
        import json

        from synnax.hardware import labjack

        # Create a rack
        rack = client.hardware.racks.retrieve_embedded_rack()

        # Create a device
        device = labjack.Device(
            model=T7,
            identifier="ANY",
            name="Test LabJack T7",
            location="USB",
            rack=rack.key,
            connection_type="ANY",
        )

        device = client.hardware.devices.create(device)

        # Create channels
        rand_int = random.randint(0, 100000)
        time_ch = client.channels.create(
            name=f"labjack_time_{rand_int}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )

        ch1 = client.channels.create(
            name=f"ain0_{rand_int}",
            data_type=sy.DataType.FLOAT32,
            index=time_ch.key,
        )

        ch2 = client.channels.create(
            name=f"fio4_{rand_int}",
            data_type=sy.DataType.UINT8,
            index=time_ch.key,
        )

        # Create task with multiple channel types
        task = labjack.ReadTask(
            name="Test Read Task",
            device=device.key,
            sample_rate=100,
            stream_rate=25,
            data_saving=True,
            channels=[
                labjack.AIChan(
                    port="AIN0",
                    channel=ch1.key,
                    range=10.0,
                ),
                labjack.DIChan(
                    port="FIO4",
                    channel=ch2.key,
                ),
            ],
        )

        # Trigger device property update
        task.update_device_properties(client.hardware.devices)

        # Retrieve device and check properties
        updated_device = client.hardware.devices.retrieve(key=device.key)
        props = json.loads(updated_device.properties)

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
        import json

        from synnax.hardware import labjack

        # Create a rack
        rack = client.hardware.racks.retrieve_embedded_rack()

        # Create a device
        device = labjack.Device(
            model=T7,
            identifier="ANY",
            name="Test LabJack Write T7",
            location="USB",
            rack=rack.key,
            connection_type="ANY",
        )

        device = client.hardware.devices.create(device)

        rand_int = random.randint(0, 100000)

        # Create command and state channels
        cmd_time = client.channels.create(
            name=f"cmd_time_{rand_int}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )

        state_time = client.channels.create(
            name=f"state_time_{rand_int}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )

        dac0_cmd = client.channels.create(
            name=f"dac0_command_{rand_int}",
            data_type=sy.DataType.FLOAT32,
            index=cmd_time.key,
        )

        dac0_state = client.channels.create(
            name=f"dac0_state_{rand_int}",
            data_type=sy.DataType.FLOAT32,
            index=state_time.key,
        )

        fio4_cmd = client.channels.create(
            name=f"fio4_command_{rand_int}",
            data_type=sy.DataType.UINT8,
            index=cmd_time.key,
        )

        fio4_state = client.channels.create(
            name=f"fio4_state_{rand_int}",
            data_type=sy.DataType.UINT8,
            index=state_time.key,
        )

        # Create write task
        task = labjack.WriteTask(
            name="Test Write Task",
            device=device.key,
            state_rate=20,
            data_saving=True,
            channels=[
                labjack.OutputChan(
                    type="AO",
                    port="DAC0",
                    cmd_channel=dac0_cmd.key,
                    state_channel=dac0_state.key,
                ),
                labjack.OutputChan(
                    type="DO",
                    port="FIO4",
                    cmd_channel=fio4_cmd.key,
                    state_channel=fio4_state.key,
                ),
            ],
        )

        # Trigger device property update
        task.update_device_properties(client.hardware.devices)

        # Retrieve device and check properties
        updated_device = client.hardware.devices.retrieve(key=device.key)
        props = json.loads(updated_device.properties)

        # Verify write.channels mapping exists
        assert "write" in props
        assert "channels" in props["write"]

        # Verify port -> state_channel key mapping
        channels = props["write"]["channels"]
        assert "DAC0" in channels
        assert channels["DAC0"] == dac0_state.key
        assert "FIO4" in channels
        assert channels["FIO4"] == fio4_state.key
