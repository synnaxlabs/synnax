#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid
from types import ModuleType

import pytest

import synnax as sy

INTEGRATIONS = [sy.can, sy.serial, sy.tcp, sy.udp, sy.arinc429, sy.mil1553]
FRAMED = [sy.serial, sy.tcp]
POLLED = [sy.serial, sy.tcp, sy.udp]


def create_library(client: sy.Synnax) -> sy.Library:
    return client.libraries.create(
        name="Bus",
        entries=[
            sy.library.MessageEntry(
                name="Status",
                identifier=sy.library.CanIdentifier(id=0x100),
                length=8,
                fields=[sy.library.BinaryField(name="rpm", bit_length=16)],
            )
        ],
    )


@pytest.mark.bus
class TestBusTasks:
    """Tests for the read and write task types of every bus integration."""

    @pytest.mark.parametrize("integration", INTEGRATIONS)
    def test_read_task_type(self, integration: ModuleType) -> None:
        """Should name the read task type after the integration."""
        t = integration.ReadTask(library=uuid.uuid4())
        assert t.TYPE == f"{integration.__name__.split('.')[-1]}_read"

    @pytest.mark.parametrize("integration", INTEGRATIONS)
    def test_read_config_round_trip(self, integration: ModuleType) -> None:
        """Should rebuild a read task from its stored task."""
        message, field = uuid.uuid4(), uuid.uuid4()
        t = integration.ReadTask(
            device="dev",
            name="Reader",
            library=uuid.uuid4(),
            raw=9,
            messages=[
                sy.bus.ReadMessage(
                    message=message,
                    index=1,
                    fields=[sy.bus.ReadField(field=field, channel=2)],
                )
            ],
        )
        stored = sy.Task(
            name="Reader",
            type=t.TYPE,
            config=t.config.model_dump(mode="json"),
        )
        rebuilt = integration.ReadTask(stored)
        assert rebuilt.config == t.config
        assert rebuilt.config.messages[0].fields[0].field == field

    @pytest.mark.parametrize("integration", FRAMED)
    def test_framing_defaults_to_newline(self, integration: ModuleType) -> None:
        """Should default a byte stream task to newline framing."""
        t = integration.ReadTask(library=uuid.uuid4())
        assert t.config.framing == sy.bus.DelimiterFraming(delimiter="\n")
        w = integration.WriteTask(library=uuid.uuid4())
        assert w.config.framing == sy.bus.DelimiterFraming(delimiter="\n")

    @pytest.mark.parametrize("integration", FRAMED)
    def test_sync_framing(self, integration: ModuleType) -> None:
        """Should carry a sync and length framing with a checksum."""
        framing = sy.bus.SyncFraming(
            sync="AA55", length_offset=2, length_size=2, checksum="crc16_modbus"
        )
        t = integration.ReadTask(library=uuid.uuid4(), framing=framing)
        dumped = t.config.model_dump(mode="json")["framing"]
        assert dumped["type"] == "sync"
        assert dumped["checksum"] == "crc16_modbus"

    @pytest.mark.parametrize("integration", POLLED)
    def test_poll_settings(self, integration: ModuleType) -> None:
        """Should store the poll rate and reply timeout."""
        t = integration.ReadTask(
            library=uuid.uuid4(), rate=5, timeout=250 * sy.TimeSpan.MILLISECOND
        )
        assert t.config.rate == 5
        assert t.config.timeout == 250 * sy.TimeSpan.MILLISECOND

    def test_can_device(self) -> None:
        """Should store CAN properties and use the backend as the model."""
        d = sy.can.Device(backend="pcan", channel="PCAN_USBBUS1", fd=True)
        assert d.make == "CAN"
        assert d.model == "pcan"
        assert d.properties["channel"] == "PCAN_USBBUS1"
        assert d.properties["fd"] is True
        assert d.properties["bitrate"] == 500000

    def test_serial_device(self) -> None:
        """Should store serial port settings with their defaults."""
        d = sy.serial.Device(port="/dev/ttyUSB0", rs485=True)
        assert d.properties == {
            "port": "/dev/ttyUSB0",
            "baud_rate": 9600,
            "data_bits": 8,
            "parity": "none",
            "stop_bits": "1",
            "flow_control": "none",
            "rs485": True,
        }

    def test_invalid_data_bits(self) -> None:
        """Should reject data bits outside a byte."""
        with pytest.raises(ValueError, match="less than or equal to 255"):
            sy.serial.Device(port="/dev/ttyUSB0", data_bits=300)

    @pytest.mark.parametrize("integration", INTEGRATIONS)
    def test_create_stamps_library_hash(
        self, client: sy.Synnax, integration: ModuleType
    ) -> None:
        """Should stamp the library hash into a task config on create."""
        lib = create_library(client)
        message = lib.entries[0]
        assert isinstance(message, sy.library.MessageEntry)
        t = integration.ReadTask(
            name="Reader",
            library=lib.key,
            messages=[
                sy.bus.ReadMessage(
                    message=message.key,
                    fields=[sy.bus.ReadField(field=message.fields[0].key)],
                )
            ],
        )
        created = client.tasks.create(name="Reader", type=t.TYPE, config=t.config)
        assert integration.ReadTask(created).config.library_hash != ""

    def test_create_rejects_unknown_message(self, client: sy.Synnax) -> None:
        """Should reject a read task whose message is not in its library."""
        lib = create_library(client)
        t = sy.can.ReadTask(
            name="Reader",
            library=lib.key,
            messages=[sy.bus.ReadMessage(message=uuid.uuid4())],
        )
        with pytest.raises(sy.ValidationError, match="is not in the library"):
            client.tasks.create(name="Reader", type=t.TYPE, config=t.config)
