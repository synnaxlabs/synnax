#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest

import synnax as sy
from synnax import ni


def _create_device(client: sy.Synnax, rack: sy.Rack) -> ni.device.Device:
    return client.ni.devices.create(
        ni.device.Device(
            rack=rack.key,
            name="NI 9205",
            resource_name=str(uuid4()),
            model="9205",
        )
    )


@pytest.mark.ni
class TestNIDeviceClient:
    def test_create_and_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should create a device and retrieve it by key with its status."""
        rack = client.racks.create(name="ni_device_rack")
        dev = _create_device(client, rack)
        got = client.ni.devices.retrieve(dev.key, include_status=True)
        assert got.name == "NI 9205"
        assert got.rack == rack.key
        assert got.status is not None

    def test_retrieve_by_filter(self, client: sy.Synnax) -> None:
        """Should retrieve the devices that a filter field selects."""
        rack = client.racks.create(name="ni_device_rack")
        first = _create_device(client, rack)
        second = _create_device(client, rack)
        got = client.ni.devices.retrieve(rack=rack.key)
        assert {d.key for d in got} == {first.key, second.key}
        one = client.ni.devices.retrieve(resource_name=first.resource_name, one=True)
        assert one.key == first.key

    def test_update_configuration(self, client: sy.Synnax) -> None:
        """Should write the configuration group and keep the identity group."""
        rack = client.racks.create(name="ni_device_rack")
        dev = _create_device(client, rack)
        got = client.ni.devices.update(
            dev.key,
            ni.device.ConfigurationUpdate(name="Thermocouples", identifier="tc"),
        )
        assert got.name == "Thermocouples"
        assert got.identifier == "tc"
        assert got.resource_name == dev.resource_name

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a device."""
        rack = client.racks.create(name="ni_device_rack")
        dev = _create_device(client, rack)
        client.ni.devices.delete(dev.key)
        with pytest.raises(sy.NotFoundError):
            client.ni.devices.retrieve(dev.key)
