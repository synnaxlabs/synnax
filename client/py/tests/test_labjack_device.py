#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
from typing import Any
from uuid import uuid4

import pytest

import synnax as sy
from synnax import labjack


def new_device(rack: sy.Rack, name: str = "T7") -> labjack.device.Device:
    return labjack.device.Device(
        rack=rack.key,
        name=name,
        serial=random.randint(1, 2**32 - 1),
        connection=labjack.device.CONNECTION_USB,
        model=labjack.device.MODEL_T7,
        ports=labjack.device.T7Ports(),
    )


@pytest.mark.labjack
class TestDeviceClient:
    def test_create_and_retrieve(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should store a device and retrieve it by key."""
        created = client.labjack.devices.create(new_device(rack))
        assert client.labjack.devices.retrieve(created.key) == created

    def test_create_many(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should create several devices in one call."""
        devices = client.labjack.devices.create(
            [new_device(rack, "A"), new_device(rack, "B")]
        )
        assert [d.name for d in devices] == ["A", "B"]

    def test_retrieve_by_serial(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should keep the devices with a serial number."""
        created = client.labjack.devices.create(new_device(rack))
        devices = client.labjack.devices.retrieve(serial=created.serial)
        assert [d.key for d in devices] == [created.key]

    def test_retrieve_by_rack(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should keep the devices on a rack."""
        created = client.labjack.devices.create(new_device(rack))
        devices = client.labjack.devices.retrieve(rack=rack.key)
        assert [d.key for d in devices] == [created.key]

    def test_retrieve_status(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should fill the status of a device on request."""
        created = client.labjack.devices.create(new_device(rack))
        retrieved = client.labjack.devices.retrieve(created.key, include_status=True)
        assert retrieved.status is not None
        assert retrieved.status.details is not None
        assert retrieved.status.details.device == created.key

    def test_retrieve_missing(self, client: sy.Synnax) -> None:
        """Should raise a NotFoundError for a key no device has."""
        with pytest.raises(sy.NotFoundError):
            client.labjack.devices.retrieve(uuid4())

    def test_update_identity(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should write the identity group and keep the configuration."""
        created = client.labjack.devices.create(new_device(rack))
        stored = client.labjack.devices.update(
            created.key,
            labjack.device.IdentityUpdate(
                rack=rack.key,
                serial=created.serial,
                connection=labjack.device.CONNECTION_ETHERNET,
                model=labjack.device.MODEL_T7,
            ),
        )
        assert stored.connection == labjack.device.CONNECTION_ETHERNET
        assert stored.name == created.name
        assert client.labjack.devices.retrieve(created.key) == stored

    def test_update_configuration(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should write the configuration group and keep the identity."""
        created = client.labjack.devices.create(new_device(rack))
        stored = client.labjack.devices.update(
            created.key,
            labjack.device.ConfigurationUpdate(
                name="Renamed",
                identifier="lj",
                ports=labjack.device.T7Ports(mux80_attached=True),
            ),
        )
        assert stored.name == "Renamed"
        assert stored.identifier == "lj"
        assert stored.serial == created.serial

    def test_update_other_model(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should refuse ports for another model."""
        created = client.labjack.devices.create(new_device(rack))
        with pytest.raises(sy.ValidationError):
            client.labjack.devices.update(
                created.key,
                labjack.device.ConfigurationUpdate(
                    name="T4", ports=labjack.device.T4Ports()
                ),
            )

    def test_delete(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should delete a device."""
        created = client.labjack.devices.create(new_device(rack))
        client.labjack.devices.delete(created.key)
        with pytest.raises(sy.NotFoundError):
            client.labjack.devices.retrieve(created.key)

    def test_retrieve_by_name(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should return one device for a plain name."""
        name = f"T7 {uuid4()}"
        created = client.labjack.devices.create(new_device(rack, name))
        assert client.labjack.devices.retrieve(name=name).key == created.key

    def test_retrieve_by_model(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should keep the devices of a model."""
        created = client.labjack.devices.create(new_device(rack))
        devices = client.labjack.devices.retrieve(
            rack=rack.key, model=labjack.device.MODEL_T7
        )
        assert [d.key for d in devices] == [created.key]
        assert (
            client.labjack.devices.retrieve(
                rack=rack.key, model=labjack.device.MODEL_T4
            )
            == []
        )

    def test_retrieve_by_keys(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should keep the devices with one of the keys."""
        a, b = client.labjack.devices.create(
            [new_device(rack, "A"), new_device(rack, "B")]
        )
        devices = client.labjack.devices.retrieve(keys=[a.key, b.key])
        assert {d.key for d in devices} == {a.key, b.key}

    def test_retrieve_one_of_many(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should raise a MultipleFoundError when one device is wanted and two match."""
        client.labjack.devices.create([new_device(rack, "A"), new_device(rack, "B")])
        with pytest.raises(sy.MultipleFoundError):
            client.labjack.devices.retrieve(rack=rack.key, one=True)

    def test_retrieve_pages(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should page through the devices with a limit and a cursor."""
        client.labjack.devices.create([new_device(rack, "A"), new_device(rack, "B")])
        first = client.labjack.devices.retrieve(rack=rack.key, limit=1)
        assert len(first) == 1
        rest = client.labjack.devices.retrieve(
            rack=rack.key,
            limit=1,
            after=labjack.device.Cursor(key=first[0].key),
        )
        assert len(rest) == 1
        assert rest[0].key != first[0].key

    def test_retrieve_none_key(self, client: sy.Synnax) -> None:
        """Should raise a ValidationError for a None key."""
        missing: Any = None
        with pytest.raises(sy.ValidationError):
            client.labjack.devices.retrieve(key=missing)

    def test_create_replaces(self, client: sy.Synnax, rack: sy.Rack) -> None:
        """Should replace a device whose key exists."""
        created = client.labjack.devices.create(new_device(rack))
        created.name = "Replaced"
        client.labjack.devices.create(created)
        assert client.labjack.devices.retrieve(created.key).name == "Replaced"
