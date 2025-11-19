#  Copyright 2025 Synnax Labs, Inc.
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

    def test_delete(self, client: sy.Synnax, new_devices: BasicDevices):
        d1, _ = new_devices
        client.hardware.devices.create(d1)
        client.hardware.devices.delete(keys=[d1.key])
        with pytest.raises(sy.NotFoundError):
            client.hardware.devices.retrieve(key=d1.key)

    def test_retrieve_ignore_not_found(self, client: sy.Synnax):
        # Test multiple device retrieval
        devices = client.hardware.devices.retrieve(
            keys=["nonexistent_key1", "nonexistent_key2"], ignore_not_found=True
        )
        assert len(devices) == 0

    def test_retrieve_not_found_error(self, client: sy.Synnax):
        # Test multiple device retrieval
        with pytest.raises(sy.NotFoundError):
            client.hardware.devices.retrieve(
                keys=["nonexistent_key1", "nonexistent_key2"], ignore_not_found=False
            )

    def test_create_with_configured_true(self, client: sy.Synnax):
        """Test creating a device with configured=True."""
        r = client.hardware.racks.create(name="Test Rack")
        device = client.hardware.devices.create(
            key=str(uuid4()),
            name="Configured Device",
            rack=r.key,
            location="test-location",
            configured=True,
        )
        assert device.configured is True
        # Verify it persists when retrieved
        retrieved = client.hardware.devices.retrieve(key=device.key)
        assert retrieved.configured is True

    def test_create_with_configured_false(self, client: sy.Synnax):
        """Test creating a device with configured=False (default)."""
        r = client.hardware.racks.create(name="Test Rack")
        device = client.hardware.devices.create(
            key=str(uuid4()),
            name="Unconfigured Device",
            rack=r.key,
            location="test-location",
            configured=False,
        )
        assert device.configured is False
        # Verify it persists when retrieved
        retrieved = client.hardware.devices.retrieve(key=device.key)
        assert retrieved.configured is False

    def test_create_multiple_with_configured(self, client: sy.Synnax):
        """Test creating multiple devices with different configured states."""
        r = client.hardware.racks.create(name="Test Rack")
        d1 = sy.Device(
            key=str(uuid4()),
            name="Device 1",
            rack=r.key,
            location="loc1",
            configured=True,
        )
        d2 = sy.Device(
            key=str(uuid4()),
            name="Device 2",
            rack=r.key,
            location="loc2",
            configured=False,
        )
        devices = client.hardware.devices.create(devices=[d1, d2])
        assert len(devices) == 2
        assert devices[0].configured is True
        assert devices[1].configured is False
