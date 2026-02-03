#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from random import randint

import pytest
from pydantic import ValidationError

import synnax as sy


@pytest.mark.ethercat
class TestEtherCATReadTask:
    """Tests for EtherCAT Read Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "automatic_input_channel",
                "data": {
                    "sample_rate": 1000,
                    "stream_rate": 100,
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "automatic",
                            "key": "auto-input-1",
                            "enabled": True,
                            "device": "slave-device-key",
                            "pdo": "Position actual value",
                            "channel": 1234,
                        },
                    ],
                },
            },
            {
                "name": "manual_input_channel",
                "data": {
                    "sample_rate": 500,
                    "stream_rate": 50,
                    "data_saving": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "manual",
                            "key": "manual-input-1",
                            "enabled": True,
                            "device": "slave-device-key",
                            "index": 0x6064,
                            "subindex": 0,
                            "bit_length": 32,
                            "data_type": "int32",
                            "channel": 5678,
                        },
                    ],
                },
            },
            {
                "name": "mixed_channels",
                "data": {
                    "sample_rate": 2000,
                    "stream_rate": 200,
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "automatic",
                            "key": "auto-1",
                            "enabled": True,
                            "device": "slave-1",
                            "pdo": "Position actual value",
                            "channel": 1000,
                        },
                        {
                            "type": "manual",
                            "key": "manual-1",
                            "enabled": True,
                            "device": "slave-2",
                            "index": 0x6077,
                            "subindex": 0,
                            "bit_length": 16,
                            "data_type": "int16",
                            "channel": 2000,
                        },
                        {
                            "type": "automatic",
                            "key": "auto-2",
                            "enabled": False,
                            "device": "slave-1",
                            "pdo": "Velocity actual value",
                            "channel": 3000,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_ethercat_read_task(self, test_data):
        """Test that ReadTaskConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.ethercat.ReadTaskConfig.model_validate(input_data)

    def test_read_task_stream_rate_validation(self):
        """Test that stream_rate cannot exceed sample_rate."""
        with pytest.raises(ValidationError) as exc_info:
            sy.ethercat.ReadTaskConfig(
                sample_rate=100,
                stream_rate=200,  # Invalid: greater than sample_rate
                data_saving=False,
                auto_start=False,
                channels=[
                    sy.ethercat.AutomaticInputChan(
                        device="slave-key",
                        pdo="Position actual value",
                        channel=1234,
                    )
                ],
            )
        assert "stream rate" in str(exc_info.value).lower()

    def test_read_task_empty_channels(self):
        """Test that empty channel list raises validation error."""
        with pytest.raises(ValidationError) as exc_info:
            sy.ethercat.ReadTaskConfig(
                sample_rate=1000,
                stream_rate=100,
                data_saving=False,
                auto_start=False,
                channels=[],
            )
        assert "at least one channel" in str(exc_info.value).lower()

    def test_read_task_auto_key_generation(self):
        """Test that channels auto-generate keys if not provided."""
        channel = sy.ethercat.AutomaticInputChan(
            device="slave-key",
            pdo="Position actual value",
            channel=1234,
        )
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_manual_input_index_bounds(self):
        """Test that index validation works (0-65535)."""
        # Valid index
        sy.ethercat.ManualInputChan(
            device="slave-key",
            index=0,
            subindex=0,
            bit_length=16,
            data_type="uint16",
            channel=1234,
        )
        sy.ethercat.ManualInputChan(
            device="slave-key",
            index=65535,
            subindex=0,
            bit_length=16,
            data_type="uint16",
            channel=1234,
        )

        # Invalid index
        with pytest.raises(ValidationError):
            sy.ethercat.ManualInputChan(
                device="slave-key",
                index=-1,
                subindex=0,
                bit_length=16,
                data_type="uint16",
                channel=1234,
            )
        with pytest.raises(ValidationError):
            sy.ethercat.ManualInputChan(
                device="slave-key",
                index=65536,
                subindex=0,
                bit_length=16,
                data_type="uint16",
                channel=1234,
            )

    def test_manual_input_subindex_bounds(self):
        """Test that subindex validation works (0-255)."""
        # Valid subindex
        sy.ethercat.ManualInputChan(
            device="slave-key",
            index=0x6064,
            subindex=0,
            bit_length=32,
            data_type="int32",
            channel=1234,
        )
        sy.ethercat.ManualInputChan(
            device="slave-key",
            index=0x6064,
            subindex=255,
            bit_length=32,
            data_type="int32",
            channel=1234,
        )

        # Invalid subindex
        with pytest.raises(ValidationError):
            sy.ethercat.ManualInputChan(
                device="slave-key",
                index=0x6064,
                subindex=-1,
                bit_length=32,
                data_type="int32",
                channel=1234,
            )
        with pytest.raises(ValidationError):
            sy.ethercat.ManualInputChan(
                device="slave-key",
                index=0x6064,
                subindex=256,
                bit_length=32,
                data_type="int32",
                channel=1234,
            )

    def test_manual_input_bit_length_bounds(self):
        """Test that bit_length validation works (1-64)."""
        # Valid bit lengths
        sy.ethercat.ManualInputChan(
            device="slave-key",
            index=0x6064,
            subindex=0,
            bit_length=1,
            data_type="bool",
            channel=1234,
        )
        sy.ethercat.ManualInputChan(
            device="slave-key",
            index=0x6064,
            subindex=0,
            bit_length=64,
            data_type="float64",
            channel=1234,
        )

        # Invalid bit lengths
        with pytest.raises(ValidationError):
            sy.ethercat.ManualInputChan(
                device="slave-key",
                index=0x6064,
                subindex=0,
                bit_length=0,
                data_type="uint8",
                channel=1234,
            )
        with pytest.raises(ValidationError):
            sy.ethercat.ManualInputChan(
                device="slave-key",
                index=0x6064,
                subindex=0,
                bit_length=65,
                data_type="uint8",
                channel=1234,
            )

    def test_create_and_retrieve_read_task(self, client: sy.Synnax):
        """Test that ReadTask can be created and retrieved from the database."""
        task = sy.ethercat.ReadTask(
            name="test-ethercat-read-task",
            sample_rate=1000,
            stream_rate=100,
            data_saving=True,
            auto_start=False,
            channels=[
                sy.ethercat.AutomaticInputChan(
                    key="auto-input-1",
                    device="slave-device-key",
                    pdo="Position actual value",
                    channel=1234,
                ),
                sy.ethercat.ManualInputChan(
                    key="manual-input-1",
                    device="slave-device-key",
                    index=0x6077,
                    subindex=0,
                    bit_length=16,
                    data_type="int16",
                    channel=5678,
                ),
            ],
        )
        created_task = client.tasks.create(
            name="test-ethercat-read-task",
            type="ethercat_read",
            config=task.config.model_dump_json(),
        )
        sy.ethercat.ReadTask(created_task)


@pytest.mark.ethercat
class TestEtherCATWriteTask:
    """Tests for EtherCAT Write Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "automatic_output_channel",
                "data": {
                    "device": "",
                    "state_rate": 10.0,
                    "execution_rate": 1000.0,
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "automatic",
                            "key": "auto-output-1",
                            "enabled": True,
                            "device": "slave-device-key",
                            "pdo": "Target velocity",
                            "cmd_channel": 1234,
                            "state_channel": 5678,
                        },
                    ],
                },
            },
            {
                "name": "manual_output_channel",
                "data": {
                    "device": "",
                    "state_rate": 5.0,
                    "execution_rate": 500.0,
                    "data_saving": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "manual",
                            "key": "manual-output-1",
                            "enabled": True,
                            "device": "slave-device-key",
                            "index": 0x60FF,
                            "subindex": 0,
                            "bit_length": 32,
                            "data_type": "int32",
                            "cmd_channel": 1234,
                            "state_channel": 0,
                        },
                    ],
                },
            },
            {
                "name": "mixed_outputs",
                "data": {
                    "device": "",
                    "state_rate": 1.0,
                    "execution_rate": 2000.0,
                    "data_saving": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "automatic",
                            "key": "auto-1",
                            "enabled": True,
                            "device": "slave-1",
                            "pdo": "Target velocity",
                            "cmd_channel": 1000,
                            "state_channel": 1001,
                        },
                        {
                            "type": "manual",
                            "key": "manual-1",
                            "enabled": True,
                            "device": "slave-2",
                            "index": 0x6040,
                            "subindex": 0,
                            "bit_length": 16,
                            "data_type": "uint16",
                            "cmd_channel": 2000,
                            "state_channel": 2001,
                        },
                        {
                            "type": "automatic",
                            "key": "auto-2",
                            "enabled": False,
                            "device": "slave-1",
                            "pdo": "Target position",
                            "cmd_channel": 3000,
                            "state_channel": 0,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_ethercat_write_task(self, test_data):
        """Test that WriteTaskConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.ethercat.WriteTaskConfig.model_validate(input_data)

    def test_write_task_empty_channels(self):
        """Test that empty channel list raises validation error."""
        with pytest.raises(ValidationError) as exc_info:
            sy.ethercat.WriteTaskConfig(
                state_rate=1.0,
                execution_rate=1000.0,
                data_saving=False,
                auto_start=False,
                channels=[],
            )
        assert "at least one channel" in str(exc_info.value).lower()

    def test_write_task_disabled_channels(self):
        """Test that disabled channels are handled correctly."""
        config = sy.ethercat.WriteTaskConfig(
            state_rate=1.0,
            execution_rate=1000.0,
            data_saving=False,
            auto_start=False,
            channels=[
                sy.ethercat.AutomaticOutputChan(
                    key="auto-1",
                    device="slave-key",
                    pdo="Target velocity",
                    cmd_channel=1234,
                    enabled=True,
                ),
                sy.ethercat.AutomaticOutputChan(
                    key="auto-2",
                    device="slave-key",
                    pdo="Target position",
                    cmd_channel=5678,
                    enabled=False,
                ),
            ],
        )
        assert len(config.channels) == 2
        assert config.channels[0].enabled is True
        assert config.channels[1].enabled is False

    def test_write_channel_auto_key_generation(self):
        """Test that output channels auto-generate a key if not provided."""
        channel = sy.ethercat.AutomaticOutputChan(
            device="slave-key",
            pdo="Target velocity",
            cmd_channel=1234,
        )
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_write_task_state_channel_optional(self):
        """Test that state_channel is optional (defaults to 0)."""
        channel = sy.ethercat.AutomaticOutputChan(
            device="slave-key",
            pdo="Target velocity",
            cmd_channel=1234,
        )
        assert channel.state_channel == 0

        channel_with_state = sy.ethercat.AutomaticOutputChan(
            device="slave-key",
            pdo="Target velocity",
            cmd_channel=1234,
            state_channel=5678,
        )
        assert channel_with_state.state_channel == 5678

    def test_create_and_retrieve_write_task(self, client: sy.Synnax):
        """Test that WriteTask can be created and retrieved from the database."""
        task = sy.ethercat.WriteTask(
            name="test-ethercat-write-task",
            state_rate=10.0,
            execution_rate=1000.0,
            data_saving=True,
            auto_start=False,
            channels=[
                sy.ethercat.AutomaticOutputChan(
                    key="auto-output-1",
                    device="slave-device-key",
                    pdo="Target velocity",
                    cmd_channel=1234,
                    state_channel=5678,
                ),
                sy.ethercat.ManualOutputChan(
                    key="manual-output-1",
                    device="slave-device-key",
                    index=0x6040,
                    subindex=0,
                    bit_length=16,
                    data_type="uint16",
                    cmd_channel=9012,
                ),
            ],
        )
        created_task = client.tasks.create(
            name="test-ethercat-write-task",
            type="ethercat_write",
            config=task.config.model_dump_json(),
        )
        sy.ethercat.WriteTask(created_task)

    def test_write_task_serialization_round_trip(self, client: sy.Synnax):
        """Test that task can be serialized and deserialized correctly."""
        original_task = sy.ethercat.WriteTask(
            name="test-round-trip",
            state_rate=5.0,
            execution_rate=500.0,
            data_saving=True,
            auto_start=False,
            channels=[
                sy.ethercat.AutomaticOutputChan(
                    key="auto-1",
                    device="slave-1",
                    pdo="Target velocity",
                    cmd_channel=1234,
                    state_channel=5678,
                    enabled=True,
                ),
                sy.ethercat.ManualOutputChan(
                    key="manual-1",
                    device="slave-2",
                    index=0x6040,
                    subindex=0,
                    bit_length=16,
                    data_type="uint16",
                    cmd_channel=9012,
                    state_channel=0,
                    enabled=False,
                ),
            ],
        )

        # Serialize to JSON
        config_json = original_task.config.model_dump_json()

        # Create task in database
        created_task = client.tasks.create(
            name="test-round-trip",
            type="ethercat_write",
            config=config_json,
        )

        # Deserialize from database
        retrieved_task = sy.ethercat.WriteTask(created_task)

        # Verify all fields match
        assert retrieved_task.config.state_rate == original_task.config.state_rate
        assert (
            retrieved_task.config.execution_rate == original_task.config.execution_rate
        )
        assert retrieved_task.config.data_saving == original_task.config.data_saving
        assert retrieved_task.config.auto_start == original_task.config.auto_start
        assert len(retrieved_task.config.channels) == len(original_task.config.channels)

        for orig_ch, retr_ch in zip(
            original_task.config.channels, retrieved_task.config.channels
        ):
            assert retr_ch.key == orig_ch.key
            assert retr_ch.device == orig_ch.device
            assert retr_ch.enabled == orig_ch.enabled


@pytest.mark.ethercat
class TestEtherCATDevice:
    """Tests for EtherCAT Device configuration."""

    def test_create_device_with_pdos(self, client: sy.Synnax):
        """Test that Device can be created with PDO definitions."""
        import json

        rack = client.racks.retrieve_embedded_rack()

        device = sy.ethercat.Device(
            name="Test Servo Drive",
            network="eth0",
            position=0,
            vendor_id=0x00000002,
            product_code=0x12345678,
            revision=0x00010000,
            serial=12345,
            rack=rack.key,
            input_pdos=[
                sy.ethercat.PDOInfo(
                    name="Position actual value",
                    index=0x6064,
                    subindex=0,
                    bit_length=32,
                    data_type="int32",
                ),
                sy.ethercat.PDOInfo(
                    name="Velocity actual value",
                    index=0x606C,
                    subindex=0,
                    bit_length=32,
                    data_type="int32",
                ),
            ],
            output_pdos=[
                sy.ethercat.PDOInfo(
                    name="Target velocity",
                    index=0x60FF,
                    subindex=0,
                    bit_length=32,
                    data_type="int32",
                ),
            ],
        )

        created_device = client.devices.create(device)
        assert created_device.key != ""
        assert created_device.name == "Test Servo Drive"
        assert created_device.make == "EtherCAT"
        assert created_device.model == "Slave"

        # Verify properties
        props = json.loads(created_device.properties)
        assert props["network"] == "eth0"
        assert props["position"] == 0
        assert props["vendor_id"] == 0x00000002
        assert props["product_code"] == 0x12345678
        assert len(props["pdos"]["inputs"]) == 2
        assert len(props["pdos"]["outputs"]) == 1
        assert props["pdos"]["inputs"][0]["name"] == "Position actual value"
        assert props["pdos"]["outputs"][0]["name"] == "Target velocity"

    def test_device_auto_key_generation(self):
        """Test that Device auto-generates key if not provided."""
        device = sy.ethercat.Device(
            name="Test Device",
            network="eth0",
            position=0,
        )
        assert device.key != ""
        assert len(device.key) > 0

    def test_device_location_format(self):
        """Test that device location is formatted as network:position."""
        device = sy.ethercat.Device(
            name="Test Device",
            network="enp3s0",
            position=5,
        )
        assert device.location == "enp3s0:5"


@pytest.mark.ethercat
class TestPDOInfo:
    """Tests for PDOInfo validation."""

    def test_pdo_info_valid(self):
        """Test valid PDOInfo creation."""
        pdo = sy.ethercat.PDOInfo(
            name="Position actual value",
            index=0x6064,
            subindex=0,
            bit_length=32,
            data_type="int32",
        )
        assert pdo.name == "Position actual value"
        assert pdo.index == 0x6064
        assert pdo.subindex == 0
        assert pdo.bit_length == 32
        assert pdo.data_type == "int32"

    def test_pdo_info_index_bounds(self):
        """Test PDOInfo index validation."""
        # Valid
        sy.ethercat.PDOInfo(
            name="Test",
            index=0,
            subindex=0,
            bit_length=8,
            data_type="uint8",
        )
        sy.ethercat.PDOInfo(
            name="Test",
            index=65535,
            subindex=0,
            bit_length=8,
            data_type="uint8",
        )

        # Invalid
        with pytest.raises(ValidationError):
            sy.ethercat.PDOInfo(
                name="Test",
                index=-1,
                subindex=0,
                bit_length=8,
                data_type="uint8",
            )
        with pytest.raises(ValidationError):
            sy.ethercat.PDOInfo(
                name="Test",
                index=65536,
                subindex=0,
                bit_length=8,
                data_type="uint8",
            )

    def test_pdo_info_bit_length_bounds(self):
        """Test PDOInfo bit_length validation."""
        # Valid
        sy.ethercat.PDOInfo(
            name="Test",
            index=0x6000,
            subindex=0,
            bit_length=1,
            data_type="bool",
        )
        sy.ethercat.PDOInfo(
            name="Test",
            index=0x6000,
            subindex=0,
            bit_length=64,
            data_type="float64",
        )

        # Invalid
        with pytest.raises(ValidationError):
            sy.ethercat.PDOInfo(
                name="Test",
                index=0x6000,
                subindex=0,
                bit_length=0,
                data_type="uint8",
            )
        with pytest.raises(ValidationError):
            sy.ethercat.PDOInfo(
                name="Test",
                index=0x6000,
                subindex=0,
                bit_length=65,
                data_type="uint8",
            )
