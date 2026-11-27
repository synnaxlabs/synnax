#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
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
        created = client.labjack.devices.create(new_device(rack))
        assert client.labjack.devices.retrieve(created.key) == created

    def test_create_many(self, client: sy.Synnax, rack: sy.Rack) -> None:
        devices = client.labjack.devices.create(
            [new_device(rack, "A"), new_device(rack, "B")]
        )
        assert [d.name for d in devices] == ["A", "B"]

    def test_retrieve_by_serial(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.labjack.devices.create(new_device(rack))
        devices = client.labjack.devices.retrieve(serial=[created.serial])
        assert [d.key for d in devices] == [created.key]

    def test_retrieve_by_rack(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.labjack.devices.create(new_device(rack))
        devices = client.labjack.devices.retrieve(rack=rack.key)
        assert [d.key for d in devices] == [created.key]

    def test_retrieve_status(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.labjack.devices.create(new_device(rack))
        retrieved = client.labjack.devices.retrieve(created.key, include_status=True)
        assert retrieved.status is not None
        assert retrieved.status.details is not None
        assert retrieved.status.details.device == created.key

    def test_retrieve_missing(self, client: sy.Synnax) -> None:
        with pytest.raises(sy.NotFoundError):
            client.labjack.devices.retrieve(uuid4())

    def test_update_identity(self, client: sy.Synnax, rack: sy.Rack) -> None:
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
        created = client.labjack.devices.create(new_device(rack))
        with pytest.raises(sy.ValidationError):
            client.labjack.devices.update(
                created.key,
                labjack.device.ConfigurationUpdate(
                    name="T4", ports=labjack.device.T4Ports()
                ),
            )

    def test_delete(self, client: sy.Synnax, rack: sy.Rack) -> None:
        created = client.labjack.devices.create(new_device(rack))
        client.labjack.devices.delete(created.key)
        with pytest.raises(sy.NotFoundError):
            client.labjack.devices.retrieve(created.key)
