#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import assert_type
from uuid import uuid4

import pytest
from pydantic import JsonValue

import synnax as sy
from synnax import device
from x import filter
from x.testutil import assert_eventually


def basic_device(rack: sy.rack.Key, n: int) -> sy.Device:
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
        devices = client.devices.retrieve(model=d1.model)
        assert_type(devices, list[sy.Device])
        assert [d.key for d in devices] == [d1.key]

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

    def test_retrieve_where(self, client: sy.Synnax) -> None:
        """Should keep devices that match a filter tree and the other filters."""
        rack = client.racks.create(name=str(uuid4()))
        devs = [client.devices.create(basic_device(rack.key, i)) for i in range(3)]
        where = device.Where(
            or_=[
                device.Where(name=filter.String(eq=devs[0].name)),
                device.Where(
                    key=filter.Set[device.Key](eq=devs[1].key),
                    model=filter.String(regex="^test"),
                ),
            ]
        )
        res = client.devices.retrieve(where=where, racks=[rack.key])
        assert {d.key for d in res} == {devs[0].key, devs[1].key}

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

    def test_retrieve_single(self, client: sy.Synnax) -> None:
        """Should return one device for a plain name, or for any filter with one."""
        rack = client.racks.create(name=str(uuid4()))
        dev = basic_device(rack.key, 1)
        dev.model = str(uuid4())
        dev = client.devices.create(dev)
        by_name = client.devices.retrieve(name=dev.name, include_status=True)
        assert_type(by_name, sy.Device)
        assert by_name.key == dev.key
        by_model = client.devices.retrieve(model=dev.model, one=True)
        assert_type(by_model, sy.Device)
        assert by_model.model == dev.model
        missing = client.devices.retrieve(
            location=str(uuid4()), one=True, ignore_not_found=True
        )
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
            match=rf"Expected one device where .*{make}.*, but found 2",
        ):
            client.devices.retrieve(make=make, one=True, ignore_not_found=True)

    def test_retrieve_no_filters(self, client: sy.Synnax) -> None:
        """Should return a list of every device when no filter is given."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))
        res = client.devices.retrieve()
        assert_type(res, list[sy.Device])
        assert dev.key in {d.key for d in res}

    def test_retrieve_fields(self, client: sy.Synnax) -> None:
        """Should keep the devices that match every field."""
        rack = client.racks.create(name=str(uuid4()))
        devs = [basic_device(rack.key, n) for n in range(3)]
        devs[0].model = devs[1].model = str(uuid4())
        client.devices.create(devs)
        res = client.devices.retrieve(rack=rack.key, model=devs[0].model)
        assert {d.key for d in res} == {devs[0].key, devs[1].key}
        res = client.devices.retrieve(rack=rack.key, location={"in": ["dev0", "dev2"]})
        assert {d.key for d in res} == {devs[0].key, devs[2].key}

    def test_retrieve_plain_key_beside_fields(self, client: sy.Synnax) -> None:
        """Should return one device for a plain key beside other fields."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))
        assert client.devices.retrieve(key=dev.key, rack=rack.key).key == dev.key
        with pytest.raises(sy.NotFoundError):
            client.devices.retrieve(key=dev.key, make="other")

    def test_retrieve_one_with_limit(self, client: sy.Synnax) -> None:
        """Should reject one beside a limit, since a page could hide a match."""
        with pytest.raises(sy.ValidationError, match="cannot set a limit"):
            client.devices.retrieve(model="T7", one=True, limit=1)  # type: ignore[call-overload]

    def test_retrieve_none_field(self, client: sy.Synnax) -> None:
        """Should reject a field set to None, since it would select every device."""
        with pytest.raises(sy.ValidationError, match="model is None"):
            client.devices.retrieve(model=None)  # type: ignore[call-overload]

    def test_retrieve_with_limit(self, client: sy.Synnax) -> None:
        """Should return at most the limit of the devices on a rack."""
        rack = client.racks.create(name=str(uuid4()))
        client.devices.create([basic_device(rack.key, i) for i in range(3)])
        assert len(client.devices.retrieve(racks=[rack.key], limit=2)) == 2

    def test_retrieve_by_search_term(self, client: sy.Synnax) -> None:
        """Should return the devices whose names match the search term."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))

        def check() -> None:
            res = client.devices.retrieve(search_term=dev.name)
            assert dev.key in {d.key for d in res}

        assert_eventually(check)

    def test_retrieve_with_status(self, client: sy.Synnax) -> None:
        """Should attach the status only when asked for it."""
        rack = client.racks.create(name=str(uuid4()))
        dev = client.devices.create(basic_device(rack.key, 1))
        assert client.devices.retrieve(key=dev.key).status is None
        res = client.devices.retrieve(key=dev.key, include_status=True)
        assert res.status is not None
        assert res.status.details is not None
        assert res.status.owner == sy.device.resource_id(dev.key)


@pytest.mark.device
class TestDeviceParent:
    def test_create_under_parent(self, client: sy.Synnax) -> None:
        """Should create a device under another device and retrieve it by parent."""
        rack = client.racks.create(name=str(uuid4()))
        chassis = client.devices.create(basic_device(rack.key, 1))
        module = basic_device(rack.key, 2)
        module.parent = chassis.key
        module = client.devices.create(module)
        assert module.parent == chassis.key
        assert client.devices.retrieve(key=chassis.key).parent is None
        held = client.devices.retrieve(parents=[chassis.key])
        assert [d.key for d in held] == [module.key]

    def test_retrieve_nested(self, client: sy.Synnax) -> None:
        """Should keep only the devices that another device holds, or none holds."""
        rack = client.racks.create(name=str(uuid4()))
        chassis = client.devices.create(basic_device(rack.key, 1))
        module = basic_device(rack.key, 2)
        module.parent = chassis.key
        client.devices.create(module)
        nested = client.devices.retrieve(racks=[rack.key], nested=True)
        assert [d.key for d in nested] == [module.key]
        top = client.devices.retrieve(racks=[rack.key], nested=False)
        assert [d.key for d in top] == [chassis.key]

    def test_delete_parent(self, client: sy.Synnax) -> None:
        """Should delete the devices that a deleted device holds."""
        rack = client.racks.create(name=str(uuid4()))
        chassis = client.devices.create(basic_device(rack.key, 1))
        module = basic_device(rack.key, 2)
        module.parent = chassis.key
        client.devices.create(module)
        client.devices.delete([chassis.key])
        assert client.devices.retrieve(key=module.key, ignore_not_found=True) is None

    def test_create_under_missing_parent(self, client: sy.Synnax) -> None:
        """Should refuse a parent that does not exist."""
        rack = client.racks.create(name=str(uuid4()))
        dev = basic_device(rack.key, 1)
        dev.parent = str(uuid4())
        with pytest.raises(sy.ValidationError, match="does not exist"):
            client.devices.create(dev)


@pytest.mark.device
class TestDeviceGroup:
    def test_create_in_group(self, client: sy.Synnax) -> None:
        """Should create a device in a group under its rack and retrieve it by that
        group."""
        rack = client.racks.create(name=str(uuid4()))
        g = client.groups.create(str(uuid4()), parent=sy.rack.resource_id(rack.key))
        dev = client.devices.create(
            key=str(uuid4()),
            rack=rack.key,
            name=str(uuid4()),
            location="dev1",
            make="test_make",
            model="test_model",
            group=g.key,
        )
        assert dev.group == g.key
        assert [d.key for d in client.devices.retrieve(groups=[g.key])] == [dev.key]
        ungrouped = client.devices.retrieve(racks=[rack.key], grouped=False)
        assert dev.key not in {d.key for d in ungrouped}

    def test_set_group(self, client: sy.Synnax) -> None:
        """Should move a device into a group and back out of it."""
        rack = client.racks.create(name=str(uuid4()))
        g = client.groups.create(str(uuid4()), parent=sy.rack.resource_id(rack.key))
        dev = client.devices.create(basic_device(rack.key, 1))
        assert dev.group is None
        client.devices.set_group([dev.key], g.key)
        assert client.devices.retrieve(key=dev.key).group == g.key
        client.devices.set_group([dev.key])
        assert client.devices.retrieve(key=dev.key).group is None

    def test_set_group_under_other_rack(self, client: sy.Synnax) -> None:
        """Should refuse a group that sits under another rack."""
        rack = client.racks.create(name=str(uuid4()))
        other = client.racks.create(name=str(uuid4()))
        g = client.groups.create(str(uuid4()), parent=sy.rack.resource_id(other.key))
        dev = client.devices.create(basic_device(rack.key, 1))
        with pytest.raises(
            sy.ValidationError,
            match=f"group {g.name} does not sit under rack {rack.key}",
        ):
            client.devices.set_group([dev.key], g.key)


@pytest.mark.device
class TestDeviceDispatch:
    def test_set(self, client: sy.Synnax) -> None:
        """Should apply each set of a dispatch to the device."""
        dev = client.devices.create(basic_device(client.racks.create(name="a").key, 1))
        client.devices.dispatch(
            dev.key,
            [
                sy.device.set(sy.device.SetPayload(path=["configured"], value=True)),
                sy.device.set(
                    sy.device.SetPayload(path=["properties"], value={"foo": "baz"})
                ),
            ],
        )
        res = client.devices.retrieve(key=dev.key)
        assert res.configured
        assert res.properties == {"foo": "baz"}

    def test_set_rack(self, client: sy.Synnax) -> None:
        """Should move a device under a parent on another rack."""
        dev = client.devices.create(basic_device(client.racks.create(name="a").key, 1))
        other = client.racks.create(name="b")
        chassis = client.devices.create(basic_device(other.key, 2))
        client.devices.dispatch(
            dev.key,
            sy.device.set_rack(
                sy.device.SetRackPayload(rack=other.key, parent=chassis.key)
            ),
        )
        res = client.devices.retrieve(key=dev.key)
        assert res.rack == other.key
        assert res.parent == chassis.key


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
