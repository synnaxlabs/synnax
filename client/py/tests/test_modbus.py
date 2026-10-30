#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

import pytest
from pydantic import ValidationError

import synnax as sy
from synnax.exceptions import PathError
from synnax.modbus.types import _READ_NAME_TYPES, _WRITE_NAME_TYPES
from tests.json_value import arr, obj
from x.strings import random_name


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
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "holding_register",
                            "key": "holding-reg-1",
                            "disabled": False,
                            "address": 0,
                            "channel": 1234,
                            "data_type": "float32",
                            "bytes_swapped": False,
                            "words_swapped": False,
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
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "input_register",
                            "key": "input-reg-1",
                            "disabled": False,
                            "address": 100,
                            "channel": 5678,
                            "data_type": "uint32",
                            "bytes_swapped": True,
                            "words_swapped": False,
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
                    "data_saving_disabled": False,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "coil",
                            "key": "coil-1",
                            "disabled": False,
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
                    "data_saving_disabled": True,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "discrete_input",
                            "key": "discrete-1",
                            "disabled": False,
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
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "holding_register",
                            "key": "holding-1",
                            "disabled": False,
                            "address": 0,
                            "channel": 1000,
                            "data_type": "float32",
                            "bytes_swapped": False,
                            "words_swapped": False,
                            "string_length": 0,
                        },
                        {
                            "type": "input_register",
                            "key": "input-1",
                            "disabled": False,
                            "address": 50,
                            "channel": 2000,
                            "data_type": "int16",
                            "bytes_swapped": False,
                            "words_swapped": False,
                            "string_length": 0,
                        },
                        {
                            "type": "coil",
                            "key": "coil-1",
                            "disabled": False,
                            "address": 0,
                            "channel": 3000,
                        },
                        {
                            "type": "discrete_input",
                            "key": "discrete-1",
                            "disabled": False,
                            "address": 0,
                            "channel": 4000,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_modbus_read_task(self, test_data: dict[str, object]) -> None:
        """Test that ReadConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.modbus.ReadConfig.model_validate(input_data)

    def test_read_config_defaults(self) -> None:
        """Test that ReadConfig applies the shared task config defaults."""
        config = sy.modbus.ReadConfig(device="test-device")
        assert config.sample_rate == sy.Rate(10)
        assert config.stream_rate == sy.Rate(5)
        assert config.data_saving_disabled is False
        assert config.auto_start is False
        assert config.channels == []

    def test_read_task_auto_key_generation(self) -> None:
        """Test that the ReadTask assigns keys to channels missing one."""
        task = sy.modbus.ReadTask(
            name="test",
            device="test-device",
            channels=[
                sy.modbus.HoldingRegisterReadChannel(
                    type="holding_register",
                    address=0,
                    channel=1234,
                    data_type=sy.DataType.FLOAT32,
                )
            ],
        )
        channel = task.config.channels[0]
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_read_task_address_bounds(self) -> None:
        """Test that address validation works (0-65535)."""
        # Valid address
        sy.modbus.HoldingRegisterReadChannel(
            type="holding_register",
            address=0,
            channel=1234,
            data_type=sy.DataType.FLOAT32,
        )
        sy.modbus.HoldingRegisterReadChannel(
            type="holding_register",
            address=65535,
            channel=1234,
            data_type=sy.DataType.FLOAT32,
        )

        # Invalid addresses
        with pytest.raises(ValidationError):
            sy.modbus.HoldingRegisterReadChannel(
                type="holding_register",
                address=-1,
                channel=1234,
                data_type=sy.DataType.FLOAT32,
            )
        with pytest.raises(ValidationError):
            sy.modbus.HoldingRegisterReadChannel(
                type="holding_register",
                address=65536,
                channel=1234,
                data_type=sy.DataType.FLOAT32,
            )

    def test_register_bit_bounds(self) -> None:
        """Test that a register bit channel takes bits 0 to 15."""
        ch = sy.modbus.InputRegisterBitReadChannel(
            type="input_register_bit", address=3, channel=1234, bit=15
        )
        assert ch.bit == 15
        with pytest.raises(ValidationError):
            sy.modbus.InputRegisterBitReadChannel(
                type="input_register_bit", address=3, channel=1234, bit=16
            )

    def test_register_channels_default_to_uint16(self) -> None:
        """Test that a register channel without a data type reads uint16."""
        ch = sy.modbus.HoldingRegisterReadChannel(
            type="holding_register", address=0, channel=1234
        )
        assert ch.data_type == "uint16"

    def test_to_payload_serializes_config(self) -> None:
        """Test that to_payload() correctly serializes the config into the payload.

        This is a regression test for the JSONConfigMixin.to_payload() method which
        must serialize self.config into the payload, not just return the internal task.
        """
        task = sy.modbus.ReadTask(
            name="test-payload-serialization",
            device="some-device-key",
            sample_rate=50,
            stream_rate=10,
            data_saving_disabled=False,
            auto_start=False,
            channels=[
                sy.modbus.HoldingRegisterReadChannel(
                    type="holding_register",
                    key="holding-reg-1",
                    address=0,
                    channel=1234,
                    data_type=sy.DataType.FLOAT32,
                ),
            ],
        )

        payload = task.to_payload()

        # Verify the config is properly serialized in the payload
        assert payload.config is not None
        assert isinstance(payload.config, dict)
        assert payload.config["sample_rate"] == 50
        assert payload.config["stream_rate"] == 10
        assert payload.config["device"] == "some-device-key"
        assert payload.config["data_saving_disabled"] is False
        assert payload.config["auto_start"] is False
        channels = arr(payload.config, "channels")
        assert len(channels) == 1
        assert obj(channels[0])["address"] == 0
        assert obj(channels[0])["data_type"] == "float32"

    def test_create_and_retrieve_read_task(self, client: sy.Synnax) -> None:
        """Test that ReadTask can be created and retrieved from the database."""
        task = sy.modbus.ReadTask(
            name="test-modbus-read-task",
            device="some-device-key",
            sample_rate=10,
            stream_rate=5,
            data_saving_disabled=True,
            auto_start=False,
            channels=[
                sy.modbus.HoldingRegisterReadChannel(
                    type="holding_register",
                    key="holding-reg-1",
                    address=0,
                    channel=1234,
                    data_type=sy.DataType.FLOAT32,
                ),
                sy.modbus.CoilReadChannel(
                    type="coil",
                    key="coil-1",
                    address=0,
                    channel=5678,
                ),
            ],
        )
        created_task = client.tasks.create(
            name="test-modbus-read-task",
            type="modbus_read",
            config=task.config,
        )
        sy.modbus.ReadTask(created_task)


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
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "coil",
                            "key": "coil-cmd-1",
                            "disabled": False,
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
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "holding_register",
                            "key": "hold-cmd-1",
                            "disabled": False,
                            "address": 100,
                            "channel": 5678,
                            "data_type": "float32",
                            "bytes_swapped": False,
                            "words_swapped": True,
                        },
                    ],
                },
            },
            {
                "name": "mixed_outputs",
                "data": {
                    "device": "modbus-device-key",
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "coil",
                            "key": "coil-cmd-1",
                            "disabled": False,
                            "address": 0,
                            "channel": 1000,
                        },
                        {
                            "type": "coil",
                            "key": "coil-cmd-2",
                            "disabled": True,
                            "address": 1,
                            "channel": 2000,
                        },
                        {
                            "type": "holding_register",
                            "key": "hold-cmd-1",
                            "disabled": False,
                            "address": 0,
                            "channel": 3000,
                            "data_type": "int16",
                            "bytes_swapped": False,
                            "words_swapped": False,
                        },
                        {
                            "type": "holding_register",
                            "key": "hold-cmd-2",
                            "disabled": False,
                            "address": 10,
                            "channel": 4000,
                            "data_type": "uint32",
                            "bytes_swapped": True,
                            "words_swapped": False,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_modbus_write_task(self, test_data: dict[str, object]) -> None:
        """Test that WriteConfig can parse various channel configurations."""
        input_data = test_data["data"]
        sy.modbus.WriteConfig.model_validate(input_data)

    def test_write_task_disabled_channels(self) -> None:
        """Test that disabled channels are handled correctly."""
        config = sy.modbus.WriteConfig(
            device="test-device",
            auto_start=False,
            channels=[
                sy.modbus.CoilWriteChannel(
                    type="coil",
                    key="coil-1",
                    address=0,
                    channel=1234,
                    disabled=False,
                ),
                sy.modbus.CoilWriteChannel(
                    type="coil",
                    key="coil-2",
                    address=1,
                    channel=5678,
                    disabled=True,
                ),
            ],
        )
        assert len(config.channels) == 2
        assert config.channels[0].disabled is False
        assert config.channels[1].disabled is True

    def test_write_channel_auto_key_generation(self) -> None:
        """Test that the WriteTask assigns keys to channels missing one."""
        task = sy.modbus.WriteTask(
            name="test",
            device="test-device",
            channels=[
                sy.modbus.CoilWriteChannel(
                    type="coil",
                    address=0,
                    channel=1234,
                )
            ],
        )
        channel = task.config.channels[0]
        assert channel.key != ""
        assert len(channel.key) > 0

    def test_to_payload_serializes_config(self) -> None:
        """Test that to_payload() correctly serializes the config into the payload.

        This is a regression test for the JSONConfigMixin.to_payload() method which
        must serialize self.config into the payload, not just return the internal task.
        """
        task = sy.modbus.WriteTask(
            name="test-payload-serialization",
            device="some-device-key",
            auto_start=False,
            channels=[
                sy.modbus.CoilWriteChannel(
                    type="coil",
                    key="coil-1",
                    address=5,
                    channel=1234,
                ),
            ],
        )

        payload = task.to_payload()

        # Verify the config is properly serialized in the payload
        assert payload.config is not None
        assert isinstance(payload.config, dict)
        assert payload.config["device"] == "some-device-key"
        assert payload.config["auto_start"] is False
        channels = arr(payload.config, "channels")
        assert len(channels) == 1
        assert obj(channels[0])["address"] == 5

    def test_create_and_retrieve_write_task(self, client: sy.Synnax) -> None:
        """Test that WriteTask can be created and retrieved from the database."""
        task = sy.modbus.WriteTask(
            name="test-modbus-write-task",
            device="some-device-key",
            auto_start=False,
            channels=[
                sy.modbus.CoilWriteChannel(
                    type="coil",
                    key="coil-cmd-1",
                    address=0,
                    channel=1234,
                ),
                sy.modbus.HoldingRegisterWriteChannel(
                    type="holding_register",
                    key="hold-cmd-1",
                    address=0,
                    channel=5678,
                    data_type=sy.DataType.FLOAT32,
                ),
            ],
        )
        created_task = client.tasks.create(
            name="test-modbus-write-task",
            type="modbus_write",
            config=task.config,
        )
        sy.modbus.WriteTask(created_task)

    def test_write_task_serialization_round_trip(self, client: sy.Synnax) -> None:
        """Test that task can be serialized and deserialized correctly."""
        original_task = sy.modbus.WriteTask(
            name="test-round-trip",
            device="some-device-key",
            auto_start=False,
            channels=[
                sy.modbus.CoilWriteChannel(
                    type="coil",
                    key="coil-cmd-1",
                    address=0,
                    channel=1234,
                    disabled=False,
                ),
                sy.modbus.HoldingRegisterWriteChannel(
                    type="holding_register",
                    key="hold-cmd-1",
                    address=10,
                    channel=5678,
                    data_type=sy.DataType.INT16,
                    bytes_swapped=True,
                    words_swapped=False,
                    disabled=True,
                ),
            ],
        )

        # Create task in database
        created_task = client.tasks.create(
            name="test-round-trip",
            type="modbus_write",
            config=original_task.config,
        )

        # Deserialize from database
        retrieved_task = sy.modbus.WriteTask(created_task)

        # Verify all fields match
        assert retrieved_task.config.device == original_task.config.device
        assert retrieved_task.config.auto_start == original_task.config.auto_start
        assert len(retrieved_task.config.channels) == len(original_task.config.channels)

        for orig_ch, retr_ch in zip(
            original_task.config.channels, retrieved_task.config.channels, strict=True
        ):
            assert retr_ch.key == orig_ch.key
            assert retr_ch.address == orig_ch.address
            assert retr_ch.channel == orig_ch.channel
            assert retr_ch.disabled == orig_ch.disabled
            if isinstance(orig_ch, sy.modbus.HoldingRegisterWriteChannel):
                assert isinstance(retr_ch, sy.modbus.HoldingRegisterWriteChannel)
                assert retr_ch.data_type == orig_ch.data_type
                assert retr_ch.bytes_swapped == orig_ch.bytes_swapped
                assert retr_ch.words_swapped == orig_ch.words_swapped

    def test_register_channel_without_override_follows_device(
        self, client: sy.Synnax
    ) -> None:
        """A register channel left unset keeps no swap fields, so the device's order
        applies, while an explicit False survives as an override."""
        task = sy.modbus.WriteTask(
            name="test-swap-inherit",
            device="some-device-key",
            channels=[
                sy.modbus.HoldingRegisterWriteChannel(
                    key="inherit",
                    address=0,
                    channel=1234,
                    data_type=sy.DataType.FLOAT32,
                ),
                sy.modbus.HoldingRegisterWriteChannel(
                    key="override",
                    address=2,
                    channel=5678,
                    data_type=sy.DataType.FLOAT32,
                    bytes_swapped=False,
                ),
            ],
        )
        created = client.tasks.create(
            name="test-swap-inherit", type="modbus_write", config=task.config
        )
        inherit, override = sy.modbus.WriteTask(created).config.channels
        assert isinstance(inherit, sy.modbus.HoldingRegisterWriteChannel)
        assert isinstance(override, sy.modbus.HoldingRegisterWriteChannel)
        assert inherit.bytes_swapped is None
        assert inherit.words_swapped is None
        assert override.bytes_swapped is False
        assert override.words_swapped is None


@pytest.mark.modbus
class TestModbusDevicePropertyUpdates:
    """Tests that device properties are correctly updated with channel mappings."""

    def test_read_task_updates_device_properties(self, client: sy.Synnax) -> None:
        """Test that configuring a ReadTask updates device properties with channel
        mappings.
        """
        # Create a rack
        rack = client.racks.retrieve_embedded_rack()

        # Create a device
        device = sy.modbus.Device(
            host="127.0.0.1",
            port=502,
            name="Test Modbus Device",
            location="127.0.0.1:502",
            rack=rack.key,
            swap_bytes=False,
            swap_words=False,
        )

        client.devices.create(device)

        # Create channels
        suffix = random_name()
        time_ch = client.channels.create(
            name=f"modbus_time_{suffix}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )

        ch1 = client.channels.create(
            name=f"register_0_{suffix}",
            data_type=sy.DataType.UINT8,
            index=time_ch.key,
        )

        ch2 = client.channels.create(
            name=f"register_1_{suffix}",
            data_type=sy.DataType.UINT16,
            index=time_ch.key,
        )

        # Create task with multiple channel types
        task = sy.modbus.ReadTask(
            name="Test Read Task",
            device=device.key,
            sample_rate=10,
            stream_rate=10,
            channels=[
                sy.modbus.InputRegisterReadChannel(
                    type="input_register",
                    channel=ch1.key,
                    address=0,
                    data_type=sy.DataType.UINT8,
                ),
                sy.modbus.HoldingRegisterReadChannel(
                    type="holding_register",
                    channel=ch2.key,
                    address=5,
                    data_type=sy.DataType.UINT16,
                ),
            ],
        )

        # Trigger device property update
        task.update_device_properties(client.devices)

        # Retrieve device and check properties
        updated_device = client.devices.retrieve(key=device.key)

        # Verify read.channels mapping exists
        assert "read" in updated_device.properties
        assert "channels" in obj(updated_device.properties, "read")

        # Verify channel keys match Console format:
        # InputRegisterReadChannel: "register-input-{address}-{dataType}"
        # HoldingRegisterReadChannel: "holding-register-input-{address}-{dataType}"
        channels = obj(updated_device.properties, "read", "channels")

        # Check InputRegisterReadChannel mapping (type-address-dataType,
        # underscores replaced with hyphens)
        assert "register-input-0-uint8" in channels
        assert channels["register-input-0-uint8"] == ch1.key

        # Check HoldingRegisterReadChannel mapping
        assert "holding-register-input-5-uint16" in channels
        assert channels["holding-register-input-5-uint16"] == ch2.key

    def test_write_task_updates_device_properties(self, client: sy.Synnax) -> None:
        """Test that configuring a WriteTask updates device properties with channel
        mappings.
        """
        # Create a rack
        rack = client.racks.retrieve_embedded_rack()

        # Create a device
        device = sy.modbus.Device(
            host="127.0.0.1",
            port=502,
            name="Test Modbus Write Device",
            location="127.0.0.1:502",
            rack=rack.key,
            swap_bytes=False,
            swap_words=False,
        )

        client.devices.create(device)

        # Create command channels
        random_id = random_name()
        cmd_time = client.channels.create(
            name=f"cmd_time_{random_id}",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )

        coil_cmd = client.channels.create(
            name=f"coil_command_{random_id}",
            data_type=sy.DataType.UINT8,
            index=cmd_time.key,
        )

        holding_cmd = client.channels.create(
            name=f"holding_command_{random_id}",
            data_type=sy.DataType.FLOAT32,
            index=cmd_time.key,
        )

        # Create write task
        task = sy.modbus.WriteTask(
            name="Test Write Task",
            device=device.key,
            channels=[
                sy.modbus.CoilWriteChannel(
                    type="coil",
                    channel=coil_cmd.key,
                    address=10,
                ),
                sy.modbus.HoldingRegisterWriteChannel(
                    type="holding_register",
                    channel=holding_cmd.key,
                    address=20,
                    data_type=sy.DataType.FLOAT32,
                ),
            ],
        )

        # Trigger device property update
        task.update_device_properties(client.devices)

        # Retrieve device and check properties
        updated_device = client.devices.retrieve(key=device.key)

        # Verify write.channels mapping exists
        assert "write" in updated_device.properties
        assert "channels" in obj(updated_device.properties, "write")

        # Verify channel keys match Console format
        channels = obj(updated_device.properties, "write", "channels")

        # Check coil output mapping (type-address, underscores replaced with hyphens)
        assert "coil-output-10" in channels
        assert channels["coil-output-10"] == coil_cmd.key

        # Check holding register output mapping
        assert "holding-register-output-20-float32" in channels
        assert channels["holding-register-output-20-float32"] == holding_cmd.key

    def test_device_connection_defaults(self) -> None:
        """Should store the defaults the Driver applies to absent fields."""
        dev = sy.modbus.Device(host="10.0.0.7", port=502)
        assert dev.properties["connection"] == {
            "transport": "tcp",
            "host": "10.0.0.7",
            "port": 502,
            "swap_bytes": False,
            "swap_words": False,
            "unit_id": 255,
            "timeout_ms": 1000,
            "retries": 1,
        }
        assert dev.properties["requests"] == {
            "max_registers": 125,
            "max_gap": 125,
            "write_codes": "multiple",
            "mask_write_supported": False,
            "delay_ms": 0,
        }

    def test_device_unit_and_requests(self) -> None:
        """Should store the unit, timing, and request settings it is given."""
        dev = sy.modbus.Device(
            host="gateway",
            port=502,
            unit_id=7,
            timeout_ms=250,
            retries=3,
            max_registers=60,
            max_gap=0,
            write_codes="single",
            mask_write_supported=True,
            delay_ms=20,
        )
        connection = obj(dev.properties["connection"])
        assert connection["unit_id"] == 7
        assert connection["timeout_ms"] == 250
        assert connection["retries"] == 3
        assert dev.properties["requests"] == {
            "max_registers": 60,
            "max_gap": 0,
            "write_codes": "single",
            "mask_write_supported": True,
            "delay_ms": 20,
        }

    def test_device_rtu_defaults(self) -> None:
        """Should store an RTU connection at 19200 8E1 that addresses unit 1."""
        dev = sy.modbus.Device(transport="rtu", serial_port="/dev/ttyUSB0")
        assert dev.properties["connection"] == {
            "transport": "rtu",
            "serial_port": "/dev/ttyUSB0",
            "baud_rate": 19200,
            "parity": "even",
            "stop_bits": 1,
            "swap_bytes": False,
            "swap_words": False,
            "unit_id": 1,
            "timeout_ms": 1000,
            "retries": 1,
        }

    def test_device_rtu_line_settings(self) -> None:
        """Should store the line settings and unit it is given."""
        dev = sy.modbus.Device(
            transport="rtu",
            serial_port="COM3",
            baud_rate=230400,
            parity="none",
            stop_bits=2,
            unit_id=17,
        )
        connection = obj(dev.properties["connection"])
        assert connection["serial_port"] == "COM3"
        assert connection["baud_rate"] == 230400
        assert connection["parity"] == "none"
        assert connection["stop_bits"] == 2
        assert connection["unit_id"] == 17

    @pytest.mark.parametrize(
        ("kwargs", "message"),
        [
            ({}, "a TCP device needs a host and a port"),
            ({"host": "10.0.0.7"}, "a TCP device needs a host and a port"),
            (
                {"host": "10.0.0.7", "port": 502, "serial_port": "COM3"},
                "a TCP device has no serial port",
            ),
            ({"transport": "rtu"}, "an RTU device needs a serial port"),
            (
                {"transport": "rtu", "serial_port": "COM3", "host": "10.0.0.7"},
                "an RTU device has no host or port",
            ),
        ],
    )
    def test_device_rejects_fields_of_other_transport(
        self, kwargs: dict[str, Any], message: str
    ) -> None:
        """Should reject a connection that lacks or mixes the fields of a transport."""
        with pytest.raises(ValueError, match=f"^{message}$"):
            sy.modbus.Device(**kwargs)

    @pytest.mark.parametrize("baud_rate", [600, 14400, 460800])
    def test_device_rejects_unsupported_baud_rate(self, baud_rate: int) -> None:
        """Should reject a baud rate that libmodbus cannot set on every platform."""
        with pytest.raises(
            PathError,
            match=r"^baud_rate: must be one of 1200, 2400, 4800, 9600, 19200, 38400, "
            r"57600, 115200, 230400$",
        ):
            sy.modbus.Device(
                transport="rtu",
                serial_port="COM3",
                baud_rate=baud_rate,  # type: ignore[arg-type]
            )

    def test_device_property_key_format(self) -> None:
        """Test that map keys keep the released type spellings after the rename."""
        # Test InputRegisterReadChannel key format
        ch = sy.modbus.InputRegisterReadChannel(
            type="input_register",
            channel=123,
            address=5,
            data_type=sy.DataType.UINT8,
        )
        expected_key = "register-input-5-uint8"
        key = f"{_READ_NAME_TYPES[ch.type]}-{ch.address}"
        if hasattr(ch, "data_type"):
            key += f"-{ch.data_type}"
        key = key.replace("_", "-")
        assert key == expected_key

        # Test HoldingRegisterReadChannel key format
        ch2 = sy.modbus.HoldingRegisterReadChannel(
            type="holding_register",
            channel=456,
            address=10,
            data_type=sy.DataType.FLOAT32,
        )
        expected_key2 = "holding-register-input-10-float32"
        key2 = f"{_READ_NAME_TYPES[ch2.type]}-{ch2.address}"
        if hasattr(ch2, "data_type"):
            key2 += f"-{ch2.data_type}"
        key2 = key2.replace("_", "-")
        assert key2 == expected_key2

        # Test coil output key format (no dataType)
        ch3 = sy.modbus.CoilWriteChannel(
            type="coil",
            channel=789,
            address=15,
        )
        expected_key3 = "coil-output-15"
        key3 = f"{_WRITE_NAME_TYPES[ch3.type]}-{ch3.address}".replace("_", "-")
        assert key3 == expected_key3

    def test_bit_channels_update_device_properties(self, client: sy.Synnax) -> None:
        """Test that register bit channels map by register and bit, so two bits of one
        register bind to two channels."""
        rack = client.racks.retrieve_embedded_rack()
        device = sy.modbus.Device(
            host="127.0.0.1",
            port=502,
            name="Test Modbus Bit Device",
            location="127.0.0.1:502",
            rack=rack.key,
        )
        client.devices.create(device)
        suffix = random_name()
        index = client.channels.create(
            name=f"bit_time_{suffix}", data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        chs = [
            client.channels.create(
                name=f"bit_{i}_{suffix}",
                data_type=sy.DataType.UINT8,
                index=index.key,
            )
            for i in range(3)
        ]
        read = sy.modbus.ReadTask(
            name="Bit read task",
            device=device.key,
            channels=[
                sy.modbus.HoldingRegisterBitReadChannel(
                    type="holding_register_bit", channel=chs[0].key, address=4, bit=0
                ),
                sy.modbus.InputRegisterBitReadChannel(
                    type="input_register_bit", channel=chs[1].key, address=4, bit=9
                ),
            ],
        )
        read.update_device_properties(client.devices)
        write = sy.modbus.WriteTask(
            name="Bit write task",
            device=device.key,
            channels=[
                sy.modbus.HoldingRegisterBitWriteChannel(
                    type="holding_register_bit", channel=chs[2].key, address=4, bit=3
                ),
            ],
        )
        write.update_device_properties(client.devices)

        updated = client.devices.retrieve(key=device.key)
        read_channels = obj(updated.properties, "read", "channels")
        assert read_channels["holding-register-bit-input-4-0"] == chs[0].key
        assert read_channels["register-bit-input-4-9"] == chs[1].key
        write_channels = obj(updated.properties, "write", "channels")
        assert write_channels["holding-register-bit-output-4-3"] == chs[2].key
