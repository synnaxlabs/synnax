#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import synnax as sy
from uuid import uuid4


def basic_device(rack: int, n: int) -> sy.Device:
    key = str(uuid4())
    return sy.Device(
        key=key, name=f"My Device {n} {key}", rack=rack, location=f"dev{n}"
    )


BasicDevices = tuple[sy.Device, sy.Device]


@pytest.mark.device
class TestDevice:
    @pytest.fixture
    def new_devices(self, client: sy.Synnax) -> BasicDevices:
        r = client.hardware.racks.create(name="dog")
        d1 = basic_device(r.key, 1)
        d2 = basic_device(r.key, 2)
        return d1, d2

    def test_create(self, client: sy.Synnax):
        r = client.hardware.racks.create(name="First Rack")
        dev = basic_device(r.key, 1)
        created = client.hardware.devices.create(dev)
        assert created.name.startswith("My Device")
        assert created.rack == r.key
        assert created.location == "dev1"

    def test_create_multiple(self, client: sy.Synnax, new_devices: BasicDevices):
        d1, d2 = new_devices
        devices = client.hardware.devices.create(devices=[d1, d2])
        assert len(devices) == 2
        assert devices[0].name.startswith("My Device 1")
        assert devices[1].name.startswith("My Device 2")

    def test_retrieve_by_keys(self, client: sy.Synnax, new_devices: BasicDevices):
        d1, d2 = new_devices
        client.hardware.devices.create(devices=[d1, d2])
        devices = client.hardware.devices.retrieve(keys=[d1.key, d2.key])
        assert len(devices) == 2
        assert devices[0].name.startswith("My Device 1")
        assert devices[1].name.startswith("My Device 2")

    def test_retrieve_by_names(self, client: sy.Synnax, new_devices: BasicDevices):
        d1, d2 = new_devices
        client.hardware.devices.create(devices=[d1, d2])
        devices = client.hardware.devices.retrieve(names=[d1.name, d2.name])
        assert len(devices) == 2

    def test_retrieve_by_key(self, client: sy.Synnax, new_devices: BasicDevices):
        d1, _ = new_devices
        client.hardware.devices.create(d1)
        device = client.hardware.devices.retrieve(key=d1.key)
        assert device.key == d1.key
        assert device.name.startswith("My Device 1")

    def test_retrieve_by_name(self, client: sy.Synnax, new_devices: BasicDevices):
        d1, _ = new_devices
        client.hardware.devices.create(d1)
        device = client.hardware.devices.retrieve(name=d1.name)
        assert device.key == d1.key
        assert device.name.startswith("My Device 1")

    def test_retrieve_by_model(self, client: sy.Synnax, new_devices: BasicDevices):
        d1, _ = new_devices
        d1.model = str(uuid4())
        client.hardware.devices.create(d1)
        device = client.hardware.devices.retrieve(model=d1.model)
        assert device.key == d1.key
        assert device.name.startswith("My Device 1")
        assert device.model == d1.model
