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
from tests.json_value import arr, obj


@pytest.mark.ethercat
class TestEtherCATReadTask:
    """Tests for EtherCAT Read Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "entry_input_channel",
                "data": {
                    "sample_rate": 1000,
                    "stream_rate": 100,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "entry",
                            "key": "entry-input-1",
                            "disabled": False,
                            "device": "slave-device-key",
                            "index": 0x6064,
                            "sub_index": 0,
                            "data_type": "int32",
                            "channel": 1234,
                        },
                    ],
                },
            },
            {
                "name": "unresolved_input_channel",
                "data": {
                    "sample_rate": 500,
                    "stream_rate": 50,
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "unresolved",
                            "key": "unresolved-input-1",
                            "disabled": False,
                            "device": "slave-device-key",
                            "pdo": "Input",
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
                            "type": "entry",
                            "key": "entry-1",
                            "disabled": False,
                            "device": "slave-1",
                            "index": 0x6064,
                            "sub_index": 0,
                            "data_type": "int32",
                            "channel": 1000,
                        },
                        {
                            "type": "unresolved",
                            "key": "unresolved-1",
                            "disabled": True,
                            "device": "slave-2",
                            "pdo": "Input",
                            "channel": 2000,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_ethercat_read_task(self, test_data: dict[str, object]) -> None:
        """Test that ReadConfig can parse various channel configurations."""
        sy.ethercat.ReadConfig.model_validate(test_data["data"])

    def test_parse_discriminates_by_type(self) -> None:
        """Should parse each channel into the class named by its type."""
        config = sy.ethercat.ReadConfig.model_validate(
            {
                "channels": [
                    {
                        "type": "entry",
                        "index": 0x6000,
                        "sub_index": 1,
                        "data_type": "uint8",
                    },
                    {"type": "unresolved", "pdo": "Input"},
                ]
            }
        )
        assert isinstance(config.channels[0], sy.ethercat.EntryReadChannel)
        assert isinstance(config.channels[1], sy.ethercat.UnresolvedReadChannel)

    def test_read_config_defaults(self) -> None:
        """Test that ReadConfig applies the shared task config defaults."""
        config = sy.ethercat.ReadConfig()
        assert config.sample_rate == sy.Rate(10)
        assert config.stream_rate == sy.Rate(5)
        assert config.data_saving_disabled is False
        assert config.auto_start is False
        assert config.channels == []

    def test_read_task_auto_key_generation(self) -> None:
        """Test that the ReadTask assigns keys to channels missing one."""
        task = sy.ethercat.ReadTask(
            name="test",
            channels=[
                sy.ethercat.EntryReadChannel(
                    device="slave-key",
                    index=0x6064,
                    sub_index=0,
                    data_type=sy.DataType.INT32,
                    channel=1234,
                )
            ],
        )
        assert task.config.channels[0].key != ""

    def test_entry_channel_scale_defaults_to_none(self) -> None:
        """Should leave an entry channel unscaled unless a scale is given."""
        ch = sy.ethercat.EntryReadChannel(
            index=0x6000, sub_index=1, data_type=sy.DataType.INT16
        )
        assert isinstance(ch.scale, sy.ethercat.NoneScale)

    @pytest.mark.parametrize(
        ("scale", "cls"),
        [
            ({"type": "linear", "slope": 2.5, "offset": -10}, sy.ethercat.LinearScale),
            (
                {
                    "type": "map",
                    "pre_scaled_min": 4,
                    "pre_scaled_max": 20,
                    "scaled_min": 0,
                    "scaled_max": 100,
                },
                sy.ethercat.MapScale,
            ),
        ],
    )
    def test_entry_channel_scale_round_trip(
        self, scale: dict[str, object], cls: type
    ) -> None:
        """Should parse each scale into the class named by its type and dump it back."""
        ch = sy.ethercat.EntryReadChannel.model_validate(
            {"index": 0x6000, "sub_index": 1, "data_type": "int16", "scale": scale}
        )
        assert isinstance(ch.scale, cls)
        assert ch.model_dump()["scale"] == scale

    @pytest.mark.parametrize("field", ["index", "sub_index", "data_type"])
    def test_entry_channel_requires_address(self, field: str) -> None:
        """Should reject an entry channel without its full address and type."""
        fields = {"index": 0x6000, "sub_index": 1, "data_type": "uint8"}
        del fields[field]
        with pytest.raises(ValidationError):
            sy.ethercat.EntryReadChannel.model_validate(
                {"device": "slave-key", **fields}
            )

    @pytest.mark.parametrize(
        ("index", "sub_index", "valid"),
        [
            (0, 0, True),
            (65535, 255, True),
            (-1, 0, False),
            (65536, 0, False),
            (0x6064, -1, False),
            (0x6064, 256, False),
        ],
    )
    def test_entry_channel_address_bounds(
        self, index: int, sub_index: int, valid: bool
    ) -> None:
        """Should bound index to uint16 and sub_index to uint8."""

        def build() -> sy.ethercat.EntryReadChannel:
            return sy.ethercat.EntryReadChannel(
                device="slave-key",
                index=index,
                sub_index=sub_index,
                data_type=sy.DataType.INT32,
                channel=1234,
            )

        if valid:
            build()
        else:
            with pytest.raises(ValidationError):
                build()

    def test_create_and_retrieve_read_task(self, client: sy.Synnax) -> None:
        """Test that ReadTask can be created and retrieved from the database."""
        task = sy.ethercat.ReadTask(
            name="test-ethercat-read-task",
            sample_rate=1000,
            stream_rate=100,
            data_saving_disabled=False,
            auto_start=False,
            channels=[
                sy.ethercat.EntryReadChannel(
                    key="entry-input-1",
                    device="slave-device-key",
                    index=0x6064,
                    sub_index=0,
                    data_type=sy.DataType.INT32,
                    channel=1234,
                ),
                sy.ethercat.EntryReadChannel(
                    key="entry-input-2",
                    device="slave-device-key",
                    index=0x6077,
                    sub_index=0,
                    data_type=sy.DataType.INT16,
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
                "name": "entry_output_channel",
                "data": {
                    "state_rate": 10.0,
                    "execution_rate": 1000.0,
                    "data_saving_disabled": False,
                    "auto_start": False,
                    "channels": [
                        {
                            "type": "entry",
                            "key": "entry-output-1",
                            "disabled": False,
                            "device": "slave-device-key",
                            "index": 0x60FF,
                            "sub_index": 0,
                            "data_type": "int32",
                            "cmd_channel": 1234,
                            "state_channel": 5678,
                        },
                    ],
                },
            },
            {
                "name": "unresolved_output_channel",
                "data": {
                    "state_rate": 5.0,
                    "execution_rate": 500.0,
                    "data_saving_disabled": True,
                    "auto_start": True,
                    "channels": [
                        {
                            "type": "unresolved",
                            "key": "unresolved-output-1",
                            "disabled": False,
                            "device": "slave-device-key",
                            "pdo": "Output",
                            "cmd_channel": 1234,
                            "state_channel": 0,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_ethercat_write_task(self, test_data: dict[str, object]) -> None:
        """Test that WriteConfig can parse various channel configurations."""
        sy.ethercat.WriteConfig.model_validate(test_data["data"])

    def test_write_config_defaults(self) -> None:
        """Test that WriteConfig applies the shared task config defaults."""
        config = sy.ethercat.WriteConfig()
        assert config.state_rate == sy.Rate(25)
        assert config.execution_rate == sy.Rate(1000)
        assert config.data_saving_disabled is False
        assert config.auto_start is False
        assert config.channels == []

    def test_write_task_disabled_channels(self) -> None:
        """Test that disabled channels are handled correctly."""
        config = sy.ethercat.WriteConfig(
            state_rate=sy.Rate(1),
            execution_rate=sy.Rate(1000),
            auto_start=False,
            channels=[
                sy.ethercat.EntryWriteChannel(
                    key="entry-1",
                    device="slave-key",
                    index=0x60FF,
                    sub_index=0,
                    data_type=sy.DataType.INT32,
                    cmd_channel=1234,
                    disabled=False,
                ),
                sy.ethercat.EntryWriteChannel(
                    key="entry-2",
                    device="slave-key",
                    index=0x607A,
                    sub_index=0,
                    data_type=sy.DataType.INT32,
                    cmd_channel=5678,
                    disabled=True,
                ),
            ],
        )
        assert len(config.channels) == 2
        assert config.channels[0].disabled is False
        assert config.channels[1].disabled is True

    def test_write_channel_auto_key_generation(self) -> None:
        """Test that the WriteTask assigns keys to channels missing one."""
        task = sy.ethercat.WriteTask(
            name="test",
            channels=[
                sy.ethercat.EntryWriteChannel(
                    device="slave-key",
                    index=0x60FF,
                    sub_index=0,
                    data_type=sy.DataType.INT32,
                    cmd_channel=1234,
                )
            ],
        )
        assert task.config.channels[0].key != ""

    def test_write_task_state_channel_optional(self) -> None:
        """Test that state_channel is optional (defaults to 0)."""
        channel = sy.ethercat.EntryWriteChannel(
            device="slave-key",
            index=0x60FF,
            sub_index=0,
            data_type=sy.DataType.INT32,
            cmd_channel=1234,
        )
        assert channel.state_channel == 0

    def test_create_and_retrieve_write_task(self, client: sy.Synnax) -> None:
        """Test that WriteTask can be created and retrieved from the database."""
        task = sy.ethercat.WriteTask(
            name="test-ethercat-write-task",
            state_rate=10.0,
            execution_rate=1000.0,
            data_saving_disabled=False,
            auto_start=False,
            channels=[
                sy.ethercat.EntryWriteChannel(
                    key="entry-output-1",
                    device="slave-device-key",
                    index=0x60FF,
                    sub_index=0,
                    data_type=sy.DataType.INT32,
                    cmd_channel=1234,
                    state_channel=5678,
                ),
            ],
        )
        created_task = client.tasks.create(
            name="test-ethercat-write-task",
            type="ethercat_write",
            config=task.config.model_dump(),
        )
        sy.ethercat.WriteTask(created_task)

    def test_write_task_serialization_round_trip(self, client: sy.Synnax) -> None:
        """Test that task can be serialized and deserialized correctly."""
        original_task = sy.ethercat.WriteTask(
            name="test-round-trip",
            state_rate=5.0,
            execution_rate=500.0,
            data_saving_disabled=False,
            auto_start=False,
            channels=[
                sy.ethercat.EntryWriteChannel(
                    key="entry-1",
                    device="slave-1",
                    index=0x60FF,
                    sub_index=0,
                    data_type=sy.DataType.INT32,
                    cmd_channel=1234,
                    state_channel=5678,
                    disabled=False,
                ),
                sy.ethercat.EntryWriteChannel(
                    key="entry-2",
                    device="slave-2",
                    index=0x6040,
                    sub_index=0,
                    data_type=sy.DataType.UINT16,
                    cmd_channel=9012,
                    state_channel=0,
                    disabled=True,
                ),
            ],
        )
        created_task = client.tasks.create(
            name="test-round-trip",
            type="ethercat_write",
            config=original_task.config,
        )
        retrieved_task = sy.ethercat.WriteTask(created_task)
        assert retrieved_task.config.state_rate == original_task.config.state_rate
        assert (
            retrieved_task.config.execution_rate == original_task.config.execution_rate
        )
        assert len(retrieved_task.config.channels) == len(original_task.config.channels)
        for orig_ch, retr_ch in zip(
            original_task.config.channels, retrieved_task.config.channels, strict=True
        ):
            assert retr_ch == orig_ch


@pytest.mark.ethercat
class TestEtherCATDevice:
    """Tests for EtherCAT Device configuration."""

    def test_create_device_with_pdos(self, client: sy.Synnax) -> None:
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
        assert created_device.make == "ethercat"
        assert created_device.model == "slave"

        # Verify properties
        props = created_device.properties
        assert props["network"] == "eth0"
        assert props["position"] == 0
        assert props["vendor_id"] == 0x00000002
        assert props["product_code"] == 0x12345678
        inputs = arr(props, "pdos", "inputs")
        outputs = arr(props, "pdos", "outputs")
        assert len(inputs) == 2
        assert len(outputs) == 1
        assert obj(inputs[0])["name"] == "Position actual value"
        assert obj(outputs[0])["name"] == "Target velocity"

    def test_device_auto_key_generation(self) -> None:
        """Test that Device auto-generates key if not provided."""
        device = sy.ethercat.Device(
            name="Test Device",
            network="eth0",
            position=0,
        )
        assert device.key != ""
        assert len(device.key) > 0

    def test_device_location_format(self) -> None:
        """Test that device location is formatted as network:position."""
        device = sy.ethercat.Device(
            name="Test Device",
            network="enp3s0",
            position=5,
        )
        assert device.location == "enp3s0:5"

    def test_device_omits_unset_pdo_names(self) -> None:
        """Test that Device leaves out the PDO name of an entry that has none."""
        device = sy.ethercat.Device(
            name="Test Device",
            network="eth0",
            position=0,
            input_pdos=[
                sy.ethercat.PDOEntry(
                    name="Input",
                    index=0x6000,
                    sub_index=1,
                    bit_length=1,
                    data_type="uint8",
                ),
                sy.ethercat.PDOEntry(
                    name="Input",
                    pdo_name="Channel 2",
                    index=0x6010,
                    sub_index=1,
                    bit_length=1,
                    data_type="uint8",
                ),
            ],
        )
        inputs = arr(device.properties, "pdos", "inputs")
        assert "pdo_name" not in obj(inputs[0])
        assert obj(inputs[1])["pdo_name"] == "Channel 2"


@pytest.mark.ethercat
class TestPDOEntry:
    """Tests for PDOEntry validation."""

    def test_pdo_entry_valid(self) -> None:
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

    def test_pdo_entry_index_bounds(self) -> None:
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

    def test_pdo_entry_bit_length_bounds(self) -> None:
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


def _create_el1008() -> sy.Device:
    """An EL1008-like slave: eight "Input" entries, one per PDO group."""
    return sy.ethercat.Device(
        name="EL1008",
        network="eth0",
        position=1,
        input_pdos=[
            sy.ethercat.PDOEntry(
                name="Input",
                pdo_name=f"Channel {i + 1}",
                pdo_index=0x1A00 + i,
                index=0x6000 + 0x10 * i,
                sub_index=1,
                bit_length=1,
                data_type="uint8",
            )
            for i in range(8)
        ],
        output_pdos=[
            sy.ethercat.PDOEntry(
                name="Output",
                pdo_name="Channel 1",
                pdo_index=0x1600,
                index=0x7000,
                sub_index=1,
                bit_length=1,
                data_type="uint8",
            )
        ],
    )


@pytest.mark.ethercat
class TestFindEntry:
    """Tests for resolving a PDO entry name to its address."""

    def test_unique_name(self) -> None:
        """Should resolve a name that only one entry has."""
        entry = sy.ethercat.find_entry(_create_el1008(), "Output", direction="output")
        assert (entry.index, entry.sub_index, entry.data_type) == (
            0x7000,
            1,
            "uint8",
        )

    def test_repeated_name_with_pdo_name(self) -> None:
        """Should pick the entry inside the named PDO group."""
        entry = sy.ethercat.find_entry(_create_el1008(), "Input", pdo_name="Channel 3")
        assert (entry.index, entry.sub_index) == (0x6020, 1)

    def test_repeated_name_alone(self) -> None:
        """Should raise and list every address when the name repeats."""
        found = ", ".join(
            f"0x{0x6000 + 0x10 * i:04X}:01 (Channel {i + 1})" for i in range(8)
        )
        with pytest.raises(sy.MultipleFoundError) as exc:
            sy.ethercat.find_entry(_create_el1008(), "Input")
        assert str(exc.value) == (
            f"input entry 'Input' on slave 'EL1008' matches {found}. "
            "Pass pdo_name to pick one"
        )

    def test_unknown_name(self) -> None:
        """Should raise when no entry has the name."""
        with pytest.raises(
            sy.NotFoundError, match=r"^no input entry 'Velocity' on slave 'EL1008'$"
        ):
            sy.ethercat.find_entry(_create_el1008(), "Velocity")

    def test_unknown_pdo_name(self) -> None:
        """Should name the PDO when no entry in it has the name."""
        with pytest.raises(
            sy.NotFoundError,
            match=r"^no input entry 'Input' in PDO 'Channel 9' on slave 'EL1008'$",
        ):
            sy.ethercat.find_entry(_create_el1008(), "Input", pdo_name="Channel 9")

    def test_wrong_direction(self) -> None:
        """Should only search entries in the given direction."""
        with pytest.raises(
            sy.NotFoundError, match=r"^no input entry 'Output' on slave 'EL1008'$"
        ):
            sy.ethercat.find_entry(_create_el1008(), "Output")

    def test_output_with_pdo_name(self) -> None:
        """Should search output entries inside the named PDO group."""
        entry = sy.ethercat.find_entry(
            _create_el1008(), "Output", pdo_name="Channel 1", direction="output"
        )
        assert (entry.index, entry.sub_index) == (0x7000, 1)

    def test_unsupported_entry_elsewhere(self) -> None:
        """Should resolve a name when another entry on the slave fails validation."""
        slave = _create_el1008()
        arr(slave.properties, "pdos", "inputs").append(
            {
                "name": "Raw",
                "index": 0x6100,
                "sub_index": 1,
                "bit_length": 128,
                "data_type": "uint8",
            }
        )
        entry = sy.ethercat.find_entry(slave, "Input", pdo_name="Channel 2")
        assert (entry.index, entry.sub_index) == (0x6010, 1)

    def test_repeated_address(self) -> None:
        """Should resolve once when groups repeat the same object."""
        slave = sy.ethercat.Device(
            name="EJ7342",
            input_pdos=[
                sy.ethercat.PDOEntry(
                    name="SubIndex 032",
                    pdo_name=f"DCM TxPDO-Map {i}",
                    index=0x1C32,
                    sub_index=0x20,
                    bit_length=1,
                    data_type="uint8",
                )
                for i in range(4)
            ],
        )
        entry = sy.ethercat.find_entry(slave, "SubIndex 032")
        assert (entry.index, entry.sub_index) == (0x1C32, 0x20)
