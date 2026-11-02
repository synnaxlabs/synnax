#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable
from typing import assert_type, cast
from uuid import uuid4

import pytest
from pydantic import JsonValue

import synnax as sy
from synnax import device
from x.testutil import assert_eventually


def basic_device(rack: int, n: int) -> sy.Device:
    key = str(uuid4())
    return sy.Device(
        key=key,
        name=f"My Device {n} {key}",
        rack=rack,
        location=f"dev{n}",
        make="test_make",
        model="test_model",
        properties={"foo": "bar"},
    )


BasicDevices = tuple[sy.Device, sy.Device]


@pytest.mark.device
class TestDevice:
    @pytest.fixture
    def new_devices(self, client: sy.Synnax) -> BasicDevices:
        r = client.racks.create(name="dog")
        d1 = basic_device(r.key, 1)
        d2 = basic_device(r.key, 2)
        return d1, d2

    def test_create(self, client: sy.Synnax) -> None:
        r = client.racks.create(name="First Rack")
        dev = basic_device(r.key, 1)
        created = client.devices.create(dev)
        assert created.name.startswith("My Device")
        assert created.rack == r.key
        assert created.location == "dev1"

    def test_create_multiple(
        self, client: sy.Synnax, new_devices: BasicDevices
    ) -> None:
        d1, d2 = new_devices
        devices = client.devices.create(devices=[d1, d2])
        assert len(devices) == 2
        assert devices[0].name.startswith("My Device 1")
        assert devices[1].name.startswith("My Device 2")

    def test_retrieve_by_keys(
        self, client: sy.Synnax, new_devices: BasicDevices
    ) -> None:
        d1, d2 = new_devices
        client.devices.create(devices=[d1, d2])
        devices = client.devices.retrieve(keys=[d1.key, d2.key])
        assert len(devices) == 2
        assert devices[0].name.startswith("My Device 1")
        assert devices[1].name.startswith("My Device 2")

    def test_retrieve_by_names(
        self, client: sy.Synnax, new_devices: BasicDevices
    ) -> None:
        d1, d2 = new_devices
        client.devices.create(devices=[d1, d2])
        devices = client.devices.retrieve(names=[d1.name, d2.name])
        assert len(devices) == 2

    def test_retrieve_by_key(
        self, client: sy.Synnax, new_devices: BasicDevices
    ) -> None:
        d1, _ = new_devices
        client.devices.create(d1)
        device = client.devices.retrieve(key=d1.key)
        assert device.key == d1.key
        assert device.name.startswith("My Device 1")

    def test_retrieve_by_name(
        self, client: sy.Synnax, new_devices: BasicDevices
    ) -> None:
        d1, _ = new_devices
        client.devices.create(d1)
        device = client.devices.retrieve(name=d1.name)
        assert device.key == d1.key
        assert device.name.startswith("My Device 1")

    def test_retrieve_by_model(
        self, client: sy.Synnax, new_devices: BasicDevices
    ) -> None:
        d1, _ = new_devices
        d1.model = str(uuid4())
        client.devices.create(d1)
        device = client.devices.retrieve(model=d1.model)
        assert device.key == d1.key
        assert device.name.startswith("My Device 1")
        assert device.model == d1.model

    def test_delete(self, client: sy.Synnax, new_devices: BasicDevices) -> None:
        d1, _ = new_devices
        client.devices.create(d1)
        client.devices.delete(keys=[d1.key])
        with pytest.raises(sy.NotFoundError):
            client.devices.retrieve(key=d1.key)

    def test_rename(self, client: sy.Synnax, new_devices: BasicDevices) -> None:
        """Should rename a device and keep its properties."""
        dev = client.devices.create(new_devices[0])
        client.devices.rename(dev.key, "After Rename")
        renamed = client.devices.retrieve(key=dev.key)
        assert renamed.name == "After Rename"
        assert renamed.properties == {"foo": "bar"}

    def test_retrieve_missing_key(self, client: sy.Synnax) -> None:
        """Should treat a missing key as a non-match in a list retrieve."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))
        res = client.devices.retrieve(keys=[dev.key, str(uuid4())])
        assert [d.key for d in res] == [dev.key]

    def test_retrieve_not_found_error(self, client: sy.Synnax) -> None:
        """Should raise when a single device is requested and none matches."""
        with pytest.raises(sy.NotFoundError):
            client.devices.retrieve(key=str(uuid4()))

    def test_retrieve_conditions(self, client: sy.Synnax) -> None:
        """Should keep devices that match every condition and the other filters."""
        rack = client.racks.create(name=str(uuid4()))
        devs = [client.devices.create(basic_device(rack.key, i)) for i in range(3)]
        res = client.devices.retrieve(
            device.name.eq(devs[0].name)
            | (device.key.eq(devs[1].key) & device.model.regex("^test")),
            racks=[rack.key],
        )
        assert {d.key for d in res} == {devs[0].key, devs[1].key}

    def test_retrieve_by_bare_key(self, client: sy.Synnax) -> None:
        """Should retrieve a device by a key passed as the first argument."""
        rack = client.racks.create(name=str(uuid4()))
        devs = [client.devices.create(basic_device(rack.key, i)) for i in range(2)]
        assert client.devices.retrieve(devs[0].key).name == devs[0].name
        keys = [d.key for d in devs]
        assert {d.key for d in client.devices.retrieve(keys)} == set(keys)
        assert client.devices.retrieve(str(uuid4()), ignore_not_found=True) is None
        with pytest.raises(sy.MultipleFoundError):
            client.devices.retrieve(device.rack.eq(rack.key), one=True)

    def test_retrieve_one_ignore_not_found(self, client: sy.Synnax) -> None:
        """Should return None for one device when none matches and not-found errors
        are ignored."""
        res = client.devices.retrieve(
            device.make.eq(str(uuid4())), one=True, ignore_not_found=True
        )
        assert_type(res, sy.Device | None)
        assert res is None

    def test_create_converts_properties_to_json(self, client: sy.Synnax) -> None:
        """Should store a UUID in the properties as a string."""
        r = client.racks.create(name="Test Rack")
        ref = uuid4()
        device = client.devices.create(
            key=str(uuid4()),
            name="JSON Device",
            rack=r.key,
            location="test-location",
            make="test_make",
            model="test_model",
            properties={"ref": ref},
        )
        retrieved = client.devices.retrieve(key=device.key)
        assert retrieved.properties == {"ref": str(ref)}

    def test_create_with_configured_true(self, client: sy.Synnax) -> None:
        """Test creating a device with configured=True."""
        r = client.racks.create(name="Test Rack")
        device = client.devices.create(
            key=str(uuid4()),
            name="Configured Device",
            rack=r.key,
            location="test-location",
            make="test_make",
            model="test_model",
            properties={},
            configured=True,
        )
        assert device.configured is True
        # Verify it persists when retrieved
        retrieved = client.devices.retrieve(key=device.key)
        assert retrieved.configured is True

    def test_create_with_configured_false(self, client: sy.Synnax) -> None:
        """Test creating a device with configured=False (default)."""
        r = client.racks.create(name="Test Rack")
        device = client.devices.create(
            key=str(uuid4()),
            name="Unconfigured Device",
            rack=r.key,
            location="test-location",
            make="test_make",
            model="test_model",
            properties={},
            configured=False,
        )
        assert device.configured is False
        # Verify it persists when retrieved
        retrieved = client.devices.retrieve(key=device.key)
        assert retrieved.configured is False

    def test_create_multiple_with_configured(self, client: sy.Synnax) -> None:
        """Test creating multiple devices with different configured states."""
        r = client.racks.create(name="Test Rack")
        d1 = sy.Device(
            key=str(uuid4()),
            name="Device 1",
            rack=r.key,
            location="loc1",
            make="test_make",
            model="test_model",
            properties={},
            configured=True,
        )
        d2 = sy.Device(
            key=str(uuid4()),
            name="Device 2",
            rack=r.key,
            location="loc2",
            make="test_make",
            model="test_model",
            properties={},
            configured=False,
        )
        devices = client.devices.create(devices=[d1, d2])
        assert len(devices) == 2
        assert devices[0].configured is True
        assert devices[1].configured is False

    def test_retrieve_by_racks(self, client: sy.Synnax) -> None:
        """Should return a list of only the devices on the given racks."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))
        res = client.devices.retrieve(racks=[rack.key])
        assert [d.key for d in res] == [dev.key]

    def test_retrieve_by_models_and_racks(self, client: sy.Synnax) -> None:
        """Should return a list of the devices matching both the models and racks."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))
        res = client.devices.retrieve(models=["test_model"], racks=[rack.key])
        assert_type(res, list[sy.Device])
        assert [d.key for d in res] == [dev.key]

    def test_retrieve_by_names_with_limit(self, client: sy.Synnax) -> None:
        """Should return a list when names are combined with a limit."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))
        res = client.devices.retrieve(names=[dev.name], limit=1)
        assert_type(res, list[sy.Device])
        assert [d.key for d in res] == [dev.key]

    def test_retrieve_single_by_each_filter(self, client: sy.Synnax) -> None:
        """Should return one device for each singular filter."""
        rack = client.racks.create(name=str(uuid4()))
        dev = basic_device(rack.key, 1)
        dev.model = str(uuid4())
        dev = client.devices.create(dev)
        by_name = client.devices.retrieve(name=dev.name, include_status=True)
        assert_type(by_name, sy.Device)
        assert by_name.key == dev.key
        by_model = client.devices.retrieve(model=dev.model)
        assert_type(by_model, sy.Device)
        assert by_model.model == dev.model
        missing = client.devices.retrieve(location=str(uuid4()), ignore_not_found=True)
        assert_type(missing, sy.Device | None)
        assert missing is None

    def test_retrieve_single_multiple_found(self, client: sy.Synnax) -> None:
        """Should raise when more than one device matches, even with
        ignore_not_found."""
        rack = client.racks.create(name=str(uuid4()))
        make = str(uuid4())
        for n in range(2):
            dev = basic_device(rack.key, n)
            dev.make = make
            client.devices.create(dev)
        with pytest.raises(
            sy.MultipleFoundError,
            match=f"Expected one Device matching {make}, but found 2",
        ):
            client.devices.retrieve(make=make, ignore_not_found=True)

    def test_retrieve_no_filters(self, client: sy.Synnax) -> None:
        """Should return a list of every device when no filter is given."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))
        res = client.devices.retrieve()
        assert_type(res, list[sy.Device])
        assert dev.key in {d.key for d in res}

    def test_retrieve_singular_with_list_filter(self, client: sy.Synnax) -> None:
        """Should raise a ValueError when singular selectors are mixed with others."""
        retrieve = cast(Callable[..., object], client.devices.retrieve)
        with pytest.raises(ValueError, match="cannot be combined"):
            retrieve(model="test", racks=[1])
        with pytest.raises(ValueError, match="only one of"):
            retrieve(make="LabJack", model="T7")

    def test_retrieve_with_limit_and_offset(self, client: sy.Synnax) -> None:
        """Should page through the devices on a rack."""
        rack = client.racks.create(name=str(uuid4()))
        client.devices.create([basic_device(rack.key, i) for i in range(3)])
        assert len(client.devices.retrieve(racks=[rack.key], limit=2)) == 2
        assert len(client.devices.retrieve(racks=[rack.key], offset=2)) == 1

    def test_retrieve_by_search_term(self, client: sy.Synnax) -> None:
        """Should return the devices whose names match the search term."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))

        def check() -> None:
            res = client.devices.retrieve(search_term=dev.name)
            assert dev.key in {d.key for d in res}

        assert_eventually(check)

    def test_retrieve_with_status_and_parent(self, client: sy.Synnax) -> None:
        """Should attach the status and parent only when asked for them."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))
        plain = client.devices.retrieve(key=dev.key)
        assert plain.status is None
        assert plain.parent is None
        res = client.devices.retrieve(
            key=dev.key, include_status=True, include_parent=True
        )
        assert res.status is not None
        assert res.status.details is not None
        assert res.status.details.device == dev.key
        assert res.parent == sy.rack.ontology_id(rack.key)


@pytest.mark.device
class TestEnsureObject:
    """Tests for walking and creating nested device property objects."""

    def test_should_create_missing_objects(self) -> None:
        props: dict[str, JsonValue] = {}
        sy.device.ensure_object(props, "read", "channels")["port"] = 1
        assert props == {"read": {"channels": {"port": 1}}}

    def test_should_return_existing_object(self) -> None:
        props: dict[str, JsonValue] = {"read": {"index": 0, "channels": {"a": 1}}}
        sy.device.ensure_object(props, "read", "channels")["b"] = 2
        assert props == {"read": {"index": 0, "channels": {"a": 1, "b": 2}}}

    def test_should_raise_when_value_is_not_object(self) -> None:
        with pytest.raises(TypeError):
            sy.device.ensure_object({"read": [1]}, "read", "channels")
