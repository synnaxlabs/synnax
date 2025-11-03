#  Copyright 2025 Synnax Labs, Inc.
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
from synnax.hardware.modbus import (
    CoilInputChan,
    CoilOutputChan,
    DiscreteInputChan,
    HoldingRegisterInputChan,
    HoldingRegisterOutputChan,
    InputRegisterChan,
    ReadTask,
    ReadTaskConfig,
    WriteTask,
    WriteTaskConfig,
)


@pytest.mark.modbus
class TestModbusReadTask:
    """Tests for Modbus TCP Read Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_holding_register",
                "data": {
                    "device": "modbus-device-key",
                    "sample_rate": 10,
                    "stream_rate": 5,
                    "data_saving": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "holding_register_input",
                            "key": "holding-reg-1",
                            "enabled": True,
                            "address": 0,
                            "channel": 1234,
                            "data_type": "float32",
                            "swap_bytes": False,
                            "swap_words": False,
                            "string_length": 0,
                        },
                    ],
                },
            },
            {
                "name": "input_register",
                "data": {
                    "device": "modbus-device-key",
                    "sample_rate": 100,
                    "stream_rate": 50,
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "register_input",
                            "key": "input-reg-1",
                            "enabled": True,
                            "address": 100,
                            "channel": 5678,
                            "data_type": "uint32",
                            "swap_bytes": True,
                            "swap_words": False,
                            "string_length": 0,
                        },
                    ],
                },
            },
            {
                "name": "coil_input",
                "data": {
                    "device": "modbus-device-key",
                    "sample_rate": 20,
                    "stream_rate": 10,
                    "data_saving": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "coil_input",
                            "key": "coil-1",
                            "enabled": True,
                            "address": 0,
                            "channel": 9012,
                        },
                    ],
                },
            },
            {
                "name": "discrete_input",
                "data": {
                    "device": "modbus-device-key",
                    "sample_rate": 50,
                    "stream_rate": 25,
                    "data_saving": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "discrete_input",
                            "key": "discrete-1",
                            "enabled": True,
                            "address": 10,
                            "channel": 3456,
                        },
                    ],
                },
            },
            {
                "name": "mixed_channels",
                "data": {
                    "device": "modbus-device-key",
                    "sample_rate": 100,
                    "stream_rate": 50,
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "holding_register_input",
                            "key": "holding-1",
                            "enabled": True,
                            "address": 0,
                            "channel": 1000,
                            "data_type": "float32",
                            "swap_bytes": False,
                            "swap_words": False,
                            "string_length": 0,
                        },
                        {
                            "type": "register_input",
                            "key": "input-1",
                            "enabled": True,
                            "address": 50,
                            "channel": 2000,
                            "data_type": "int16",
                            "swap_bytes": False,
                            "swap_words": False,
                            "string_length": 0,
                        },
                        {
                            "type": "coil_input",
                            "key": "coil-1",
                            "enabled": True,
                            "address": 0,
                            "channel": 3000,
                        },
                        {
                            "type": "discrete_input",
                            "key": "discrete-1",
                            "enabled": True,
                            "address": 0,
                            "channel": 4000,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_modbus_read_task(self, test_data):
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
                    HoldingRegisterInputChan(
                        address=0,
                        channel=1234,
                        data_type="float32",
                    )
                ],
            )
        assert "stream rate" in str(exc_info.value).lower()

    def test_read_task_empty_channels(self):
        """Test that empty channel list raises validation error."""
        with pytest.raises(ValidationError) as exc_info:
            ReadTaskConfig(
                device="test-device",
                sample_rate=10,
                stream_rate=5,
                data_saving=False,
                auto_start=False,
                channels=[],  # Empty list
            )
        assert "at least one channel" in str(exc_info.value).lower()

    def test_read_task_auto_key_generation(self):
        """Test that channels auto-generate keys if not provided."""
        channel = HoldingRegisterInputChan(
            address=0,
            channel=1234,
            data_type="float32",
        )
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_read_task_address_bounds(self):
        """Test that address validation works (0-65535)."""
        # Valid address
        HoldingRegisterInputChan(address=0, channel=1234, data_type="float32")
        HoldingRegisterInputChan(address=65535, channel=1234, data_type="float32")

        # Invalid addresses
        with pytest.raises(ValidationError):
            HoldingRegisterInputChan(address=-1, channel=1234, data_type="float32")
        with pytest.raises(ValidationError):
            HoldingRegisterInputChan(address=65536, channel=1234, data_type="float32")

    def test_create_and_retrieve_read_task(self, client: sy.Synnax):
        """Test that ReadTask can be created and retrieved from the database."""
        task = ReadTask(
            name="test-modbus-read-task",
            device="some-device-key",
            sample_rate=10,
            stream_rate=5,
            data_saving=False,
            auto_start=False,
            channels=[
                HoldingRegisterInputChan(
                    key="holding-reg-1",
                    address=0,
                    channel=1234,
                    data_type="float32",
                ),
                CoilInputChan(
                    key="coil-1",
                    address=0,
                    channel=5678,
                ),
            ],
        )
        created_task = client.hardware.tasks.create(
            name="test-modbus-read-task",
            type="modbus_read",
            config=task.config.model_dump_json(),
        )
        ReadTask(created_task)


@pytest.mark.modbus
class TestModbusWriteTask:
    """Tests for Modbus TCP Write Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_coil_output",
                "data": {
                    "device": "modbus-device-key",
                    "data_saving": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "coil_output",
                            "key": "coil-cmd-1",
                            "enabled": True,
                            "address": 0,
                            "channel": 1234,
                        },
                    ],
                },
            },
            {
                "name": "holding_register_output",
                "data": {
                    "device": "modbus-device-key",
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "holding_register_output",
                            "key": "hold-cmd-1",
                            "enabled": True,
                            "address": 100,
                            "channel": 5678,
                            "data_type": "float32",
                            "swap_bytes": False,
                            "swap_words": True,
                        },
                    ],
                },
            },
            {
                "name": "mixed_outputs",
                "data": {
                    "device": "modbus-device-key",
                    "data_saving": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "coil_output",
                            "key": "coil-cmd-1",
                            "enabled": True,
                            "address": 0,
                            "channel": 1000,
                        },
                        {
                            "type": "coil_output",
                            "key": "coil-cmd-2",
                            "enabled": False,
                            "address": 1,
                            "channel": 2000,
                        },
                        {
                            "type": "holding_register_output",
                            "key": "hold-cmd-1",
                            "enabled": True,
                            "address": 0,
                            "channel": 3000,
                            "data_type": "int16",
                            "swap_bytes": False,
                            "swap_words": False,
                        },
                        {
                            "type": "holding_register_output",
                            "key": "hold-cmd-2",
                            "enabled": True,
                            "address": 10,
                            "channel": 4000,
                            "data_type": "uint32",
                            "swap_bytes": True,
                            "swap_words": False,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_modbus_write_task(self, test_data):
        """Test that WriteTaskConfig can parse various channel configurations."""
        input_data = test_data["data"]
        WriteTaskConfig.model_validate(input_data)

    def test_write_task_empty_channels(self):
        """Test that empty channel list raises validation error."""
        with pytest.raises(ValidationError) as exc_info:
            WriteTaskConfig(
                device="test-device",
                data_saving=False,
                auto_start=False,
                channels=[],
            )
        assert "at least one channel" in str(exc_info.value).lower()

    def test_write_task_disabled_channels(self):
        """Test that disabled channels are handled correctly."""
        config = WriteTaskConfig(
            device="test-device",
            data_saving=False,
            auto_start=False,
            channels=[
                CoilOutputChan(
                    key="coil-1",
                    address=0,
                    channel=1234,
                    enabled=True,
                ),
                CoilOutputChan(
                    key="coil-2",
                    address=1,
                    channel=5678,
                    enabled=False,
                ),
            ],
        )
        assert len(config.channels) == 2
        assert config.channels[0].enabled is True
        assert config.channels[1].enabled is False

    def test_write_channel_auto_key_generation(self):
        """Test that WriteChannel auto-generates a key if not provided."""
        channel = CoilOutputChan(
            address=0,
            channel=1234,
        )
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_create_and_retrieve_write_task(self, client: sy.Synnax):
        """Test that WriteTask can be created and retrieved from the database."""
        task = WriteTask(
            name="test-modbus-write-task",
            device="some-device-key",
            data_saving=True,
            auto_start=False,
            channels=[
                CoilOutputChan(
                    key="coil-cmd-1",
                    address=0,
                    channel=1234,
                ),
                HoldingRegisterOutputChan(
                    key="hold-cmd-1",
                    address=0,
                    channel=5678,
                    data_type="float32",
                ),
            ],
        )
        created_task = client.hardware.tasks.create(
            name="test-modbus-write-task",
            type="modbus_write",
            config=task.config.model_dump_json(),
        )
        WriteTask(created_task)

    def test_write_task_serialization_round_trip(self, client: sy.Synnax):
        """Test that task can be serialized and deserialized correctly."""
        original_task = WriteTask(
            name="test-round-trip",
            device="some-device-key",
            data_saving=True,
            auto_start=False,
            channels=[
                CoilOutputChan(
                    key="coil-cmd-1",
                    address=0,
                    channel=1234,
                    enabled=True,
                ),
                HoldingRegisterOutputChan(
                    key="hold-cmd-1",
                    address=10,
                    channel=5678,
                    data_type="int16",
                    swap_bytes=True,
                    swap_words=False,
                    enabled=False,
                ),
            ],
        )

        # Serialize to JSON
        config_json = original_task.config.model_dump_json()

        # Create task in database
        created_task = client.hardware.tasks.create(
            name="test-round-trip",
            type="modbus_write",
            config=config_json,
        )

        # Deserialize from database
        retrieved_task = WriteTask(created_task)

        # Verify all fields match
        assert retrieved_task.config.device == original_task.config.device
        assert retrieved_task.config.data_saving == original_task.config.data_saving
        assert retrieved_task.config.auto_start == original_task.config.auto_start
        assert len(retrieved_task.config.channels) == len(original_task.config.channels)

        for orig_ch, retr_ch in zip(
            original_task.config.channels, retrieved_task.config.channels
        ):
            assert retr_ch.key == orig_ch.key
            assert retr_ch.address == orig_ch.address
            assert retr_ch.channel == orig_ch.channel
            assert retr_ch.enabled == orig_ch.enabled
            if isinstance(orig_ch, HoldingRegisterOutputChan):
                assert isinstance(retr_ch, HoldingRegisterOutputChan)
                assert retr_ch.data_type == orig_ch.data_type
                assert retr_ch.swap_bytes == orig_ch.swap_bytes
                assert retr_ch.swap_words == orig_ch.swap_words
