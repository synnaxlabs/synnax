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
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "automatic",
                            "key": "auto-input-1",
                            "disabled": False,
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
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "manual",
                            "key": "manual-input-1",
                            "disabled": False,
                            "device": "slave-device-key",
                            "index": 0x6064,
                            "sub_index": 0,
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
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "automatic",
                            "key": "auto-1",
                            "disabled": False,
                            "device": "slave-1",
                            "pdo": "Position actual value",
                            "channel": 1000,
                        },
                        {
                            "type": "manual",
                            "key": "manual-1",
                            "disabled": False,
                            "device": "slave-2",
                            "index": 0x6077,
                            "sub_index": 0,
                            "bit_length": 16,
                            "data_type": "int16",
                            "channel": 2000,
                        },
                        {
                            "type": "automatic",
                            "key": "auto-2",
                            "disabled": True,
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
        """Test that ReadConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.ethercat.ReadConfig.model_validate(input_data)

    def test_read_config_defaults(self):
        """Test that ReadConfig applies the shared task config defaults."""
        config = sy.ethercat.ReadConfig()
        assert config.sample_rate == sy.Rate(10)
        assert config.stream_rate == sy.Rate(5)
        assert config.data_saving_disabled is False
        assert config.auto_start is False
        assert config.channels == []

    def test_read_task_auto_key_generation(self):
        """Test that the ReadTask assigns keys to channels missing one."""
        task = sy.ethercat.ReadTask(
            name="test",
            channels=[
                sy.ethercat.AutomaticReadChannel(
                    type="automatic",
                    device="slave-key",
                    pdo="Position actual value",
                    channel=1234,
                )
            ],
        )
        channel = task.config.channels[0]
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_manual_input_index_bounds(self):
        """Test that index validation works (0-65535)."""
        # Valid index
        sy.ethercat.ManualReadChannel(
            type="manual",
            device="slave-key",
            index=0,
            sub_index=0,
            bit_length=16,
            data_type="uint16",
            channel=1234,
        )
        sy.ethercat.ManualReadChannel(
            type="manual",
            device="slave-key",
            index=65535,
            sub_index=0,
            bit_length=16,
            data_type="uint16",
            channel=1234,
        )

        # Invalid index
        with pytest.raises(ValidationError):
            sy.ethercat.ManualReadChannel(
                type="manual",
                device="slave-key",
                index=-1,
                sub_index=0,
                bit_length=16,
                data_type="uint16",
                channel=1234,
            )
        with pytest.raises(ValidationError):
            sy.ethercat.ManualReadChannel(
                type="manual",
                device="slave-key",
                index=65536,
                sub_index=0,
                bit_length=16,
                data_type="uint16",
                channel=1234,
            )

    def test_manual_input_sub_index_bounds(self):
        """Test that sub_index validation works (0-255)."""
        # Valid sub_index
        sy.ethercat.ManualReadChannel(
            type="manual",
            device="slave-key",
            index=0x6064,
            sub_index=0,
            bit_length=32,
            data_type="int32",
            channel=1234,
        )
        sy.ethercat.ManualReadChannel(
            type="manual",
            device="slave-key",
            index=0x6064,
            sub_index=255,
            bit_length=32,
            data_type="int32",
            channel=1234,
        )

        # Invalid sub_index
        with pytest.raises(ValidationError):
            sy.ethercat.ManualReadChannel(
                type="manual",
                device="slave-key",
                index=0x6064,
                sub_index=-1,
                bit_length=32,
                data_type="int32",
                channel=1234,
            )
        with pytest.raises(ValidationError):
            sy.ethercat.ManualReadChannel(
                type="manual",
                device="slave-key",
                index=0x6064,
                sub_index=256,
                bit_length=32,
                data_type="int32",
                channel=1234,
            )

    def test_manual_input_bit_length_bounds(self):
        """Test that bit_length validation works (0-255)."""
        # Valid bit lengths
        sy.ethercat.ManualReadChannel(
            type="manual",
            device="slave-key",
            index=0x6064,
            sub_index=0,
            bit_length=1,
            data_type="uint8",
            channel=1234,
        )
        sy.ethercat.ManualReadChannel(
            type="manual",
            device="slave-key",
            index=0x6064,
            sub_index=0,
            bit_length=64,
            data_type="float64",
            channel=1234,
        )

        # Invalid bit lengths
        with pytest.raises(ValidationError):
            sy.ethercat.ManualReadChannel(
                type="manual",
                device="slave-key",
                index=0x6064,
                sub_index=0,
                bit_length=-1,
                data_type="uint8",
                channel=1234,
            )
        with pytest.raises(ValidationError):
            sy.ethercat.ManualReadChannel(
                type="manual",
                device="slave-key",
                index=0x6064,
                sub_index=0,
                bit_length=256,
                data_type="uint8",
                channel=1234,
            )

    def test_create_and_retrieve_read_task(self, client: sy.Synnax):
        """Test that ReadTask can be created and retrieved from the database."""
        task = sy.ethercat.ReadTask(
            name="test-ethercat-read-task",
            sample_rate=1000,
            stream_rate=100,
            data_saving_disabled=False,
            auto_start=False,
            channels=[
                sy.ethercat.AutomaticReadChannel(
                    type="automatic",
                    key="auto-input-1",
                    device="slave-device-key",
                    pdo="Position actual value",
                    channel=1234,
                ),
                sy.ethercat.ManualReadChannel(
                    type="manual",
                    key="manual-input-1",
                    device="slave-device-key",
                    index=0x6077,
                    sub_index=0,
                    bit_length=16,
                    data_type="int16",
                    channel=5678,
                ),
            ],
        )
        created_task = client.tasks.create(
            name="test-ethercat-read-task",
            type="ethercat_read",
            config=task.config.model_dump(),
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
                    "state_rate": 10.0,
                    "execution_rate": 1000.0,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "automatic",
                            "key": "auto-output-1",
                            "disabled": False,
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
                    "state_rate": 5.0,
                    "execution_rate": 500.0,
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "manual",
                            "key": "manual-output-1",
                            "disabled": False,
                            "device": "slave-device-key",
                            "index": 0x60FF,
                            "sub_index": 0,
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
                    "state_rate": 1.0,
                    "execution_rate": 2000.0,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "automatic",
                            "key": "auto-1",
                            "disabled": False,
                            "device": "slave-1",
                            "pdo": "Target velocity",
                            "cmd_channel": 1000,
                            "state_channel": 1001,
                        },
                        {
                            "type": "manual",
                            "key": "manual-1",
                            "disabled": False,
                            "device": "slave-2",
                            "index": 0x6040,
                            "sub_index": 0,
                            "bit_length": 16,
                            "data_type": "uint16",
                            "cmd_channel": 2000,
                            "state_channel": 2001,
                        },
                        {
                            "type": "automatic",
                            "key": "auto-2",
                            "disabled": True,
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
        """Test that WriteConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.ethercat.WriteConfig.model_validate(input_data)

    def test_write_config_defaults(self):
        """Test that WriteConfig applies the shared task config defaults."""
        config = sy.ethercat.WriteConfig()
        assert config.state_rate == sy.Rate(25)
        assert config.execution_rate == sy.Rate(1000)
        assert config.data_saving_disabled is False
        assert config.auto_start is False
        assert config.channels == []

    def test_write_task_disabled_channels(self):
        """Test that disabled channels are handled correctly."""
        config = sy.ethercat.WriteConfig(
            state_rate=sy.Rate(1),
            execution_rate=sy.Rate(1000),
            auto_start=False,
            channels=[
                sy.ethercat.AutomaticWriteChannel(
                    type="automatic",
                    key="auto-1",
                    device="slave-key",
                    pdo="Target velocity",
                    cmd_channel=1234,
                    disabled=False,
                ),
                sy.ethercat.AutomaticWriteChannel(
                    type="automatic",
                    key="auto-2",
                    device="slave-key",
                    pdo="Target position",
                    cmd_channel=5678,
                    disabled=True,
                ),
            ],
        )
        assert len(config.channels) == 2
        assert config.channels[0].disabled is False
        assert config.channels[1].disabled is True

    def test_write_channel_auto_key_generation(self):
        """Test that the WriteTask assigns keys to channels missing one."""
        task = sy.ethercat.WriteTask(
            name="test",
            channels=[
                sy.ethercat.AutomaticWriteChannel(
                    type="automatic",
                    device="slave-key",
                    pdo="Target velocity",
                    cmd_channel=1234,
                )
            ],
        )
        channel = task.config.channels[0]
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_write_task_state_channel_optional(self):
        """Test that state_channel is optional (defaults to 0)."""
        channel = sy.ethercat.AutomaticWriteChannel(
            type="automatic",
            device="slave-key",
            pdo="Target velocity",
            cmd_channel=1234,
        )
        assert channel.state_channel == 0

        channel_with_state = sy.ethercat.AutomaticWriteChannel(
            type="automatic",
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
            data_saving_disabled=False,
            auto_start=False,
            channels=[
                sy.ethercat.AutomaticWriteChannel(
                    type="automatic",
                    key="auto-output-1",
                    device="slave-device-key",
                    pdo="Target velocity",
                    cmd_channel=1234,
                    state_channel=5678,
                ),
                sy.ethercat.ManualWriteChannel(
                    type="manual",
                    key="manual-output-1",
                    device="slave-device-key",
                    index=0x6040,
                    sub_index=0,
                    bit_length=16,
                    data_type="uint16",
                    cmd_channel=9012,
                ),
            ],
        )
        created_task = client.tasks.create(
            name="test-ethercat-write-task",
            type="ethercat_write",
            config=task.config.model_dump(),
        )
        sy.ethercat.WriteTask(created_task)

    def test_write_task_serialization_round_trip(self, client: sy.Synnax):
        """Test that task can be serialized and deserialized correctly."""
        original_task = sy.ethercat.WriteTask(
            name="test-round-trip",
            state_rate=5.0,
            execution_rate=500.0,
            data_saving_disabled=False,
            auto_start=False,
            channels=[
                sy.ethercat.AutomaticWriteChannel(
                    type="automatic",
                    key="auto-1",
                    device="slave-1",
                    pdo="Target velocity",
                    cmd_channel=1234,
                    state_channel=5678,
                    disabled=False,
                ),
                sy.ethercat.ManualWriteChannel(
                    type="manual",
                    key="manual-1",
                    device="slave-2",
                    index=0x6040,
                    sub_index=0,
                    bit_length=16,
                    data_type="uint16",
                    cmd_channel=9012,
                    state_channel=0,
                    disabled=True,
                ),
            ],
        )

        # Create task in database
        created_task = client.tasks.create(
            name="test-round-trip",
            type="ethercat_write",
            config=original_task.config,
        )

        # Deserialize from database
        retrieved_task = sy.ethercat.WriteTask(created_task)

        # Verify all fields match
        assert retrieved_task.config.state_rate == original_task.config.state_rate
        assert (
            retrieved_task.config.execution_rate == original_task.config.execution_rate
        )
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
            assert retr_ch.device == orig_ch.device
            assert retr_ch.disabled == orig_ch.disabled


@pytest.mark.ethercat
class TestEtherCATDevice:
    """Tests for EtherCAT Device configuration."""

    def test_create_device_with_pdos(self, client: sy.Synnax):
        """Test that Device can be created with PDO definitions."""
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
                sy.ethercat.PDOEntry(
                    name="Position actual value",
                    index=0x6064,
                    sub_index=0,
                    bit_length=32,
                    data_type="int32",
                ),
                sy.ethercat.PDOEntry(
                    name="Velocity actual value",
                    index=0x606C,
                    sub_index=0,
                    bit_length=32,
                    data_type="int32",
                ),
            ],
            output_pdos=[
                sy.ethercat.PDOEntry(
                    name="Target velocity",
                    index=0x60FF,
                    sub_index=0,
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
        props = created_device.properties
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
class TestPDOEntry:
    """Tests for PDOEntry validation."""

    def test_pdo_entry_valid(self):
        """Test valid PDOEntry creation."""
        pdo = sy.ethercat.PDOEntry(
            name="Position actual value",
            index=0x6064,
            sub_index=0,
            bit_length=32,
            data_type="int32",
        )
        assert pdo.name == "Position actual value"
        assert pdo.index == 0x6064
        assert pdo.sub_index == 0
        assert pdo.bit_length == 32
        assert pdo.data_type == "int32"

    def test_pdo_entry_index_bounds(self):
        """Test PDOEntry index validation."""
        # Valid
        sy.ethercat.PDOEntry(
            name="Test",
            index=0,
            sub_index=0,
            bit_length=8,
            data_type="uint8",
        )
        sy.ethercat.PDOEntry(
            name="Test",
            index=65535,
            sub_index=0,
            bit_length=8,
            data_type="uint8",
        )

        # Invalid
        with pytest.raises(ValidationError):
            sy.ethercat.PDOEntry(
                name="Test",
                index=-1,
                sub_index=0,
                bit_length=8,
                data_type="uint8",
            )
        with pytest.raises(ValidationError):
            sy.ethercat.PDOEntry(
                name="Test",
                index=65536,
                sub_index=0,
                bit_length=8,
                data_type="uint8",
            )

    def test_pdo_entry_bit_length_bounds(self):
        """Test PDOEntry bit_length validation."""
        # Valid
        sy.ethercat.PDOEntry(
            name="Test",
            index=0x6000,
            sub_index=0,
            bit_length=1,
            data_type="uint8",
        )
        sy.ethercat.PDOEntry(
            name="Test",
            index=0x6000,
            sub_index=0,
            bit_length=64,
            data_type="float64",
        )

        # Invalid
        with pytest.raises(ValidationError):
            sy.ethercat.PDOEntry(
                name="Test",
                index=0x6000,
                sub_index=0,
                bit_length=0,
                data_type="uint8",
            )
        with pytest.raises(ValidationError):
            sy.ethercat.PDOEntry(
                name="Test",
                index=0x6000,
                sub_index=0,
                bit_length=65,
                data_type="uint8",
            )
