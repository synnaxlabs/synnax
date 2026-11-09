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

import synnax as sy
from synnax import rack
from x import filter
from x.testutil import assert_eventually


@pytest.mark.rack
class TestRackClient:
    def test_create_single(self, client: sy.Synnax) -> None:
        """It should correctly create a single rack"""
        rack = client.racks.create(name="test")
        assert rack.key != 0

    def test_create_driver_role(self, client: sy.Synnax) -> None:
        """Should give a new rack the Driver role."""
        driver = client.access.roles.retrieve(name="Driver")
        rack = client.racks.create(name="test")
        assert client.racks.retrieve(key=rack.key).role == driver.key

    def test_create_multiple(self, client: sy.Synnax) -> None:
        """Should create multiple racks"""
        r1 = sy.Rack(name="test1")
        r2 = sy.Rack(name="test2")
        racks = client.racks.create(racks=[r1, r2])
        assert len(racks) == 2
        assert racks[0].name == "test1"
        assert racks[1].name == "test2"

    def test_retrieve_by_name(self, client: sy.Synnax) -> None:
        """Should retrieve a rack by name"""
        name = str(uuid4())
        rack = client.racks.create(name=name)
        res = client.racks.retrieve(name=name)
        assert res.name == name
        assert res.key == rack.key

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve a rack by key"""
        rack = client.racks.create(name="test")
        res = client.racks.retrieve(key=rack.key)
        assert res.key == rack.key
        assert res.name == rack.name

    def test_retrieve_multiple(self, client: sy.Synnax) -> None:
        """Should retrieve multiple racks"""
        r1 = client.racks.create(name="test1")
        r2 = client.racks.create(name="test2")
        racks = client.racks.retrieve(keys=[r1.key, r2.key])
        assert len(racks) == 2
        assert {r.key for r in racks} == {r1.key, r2.key}

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a rack."""
        rack = client.racks.create(name="Before Rename")
        client.racks.rename(rack.key, "After Rename")
        assert client.racks.retrieve(key=rack.key).name == "After Rename"

    def test_rename_embedded(self, client: sy.Synnax) -> None:
        """Should rename the embedded rack and keep it embedded."""
        rack = client.racks.retrieve_embedded_rack()
        try:
            client.racks.rename(rack.key, "After Rename")
            renamed = client.racks.retrieve(key=rack.key)
            assert renamed.name == "After Rename"
            assert renamed.embedded
            assert client.racks.retrieve_embedded_rack().name == "After Rename"
        finally:
            client.racks.rename(rack.key, rack.name)

    def test_retrieve_by_name_multiple_found(self, client: sy.Synnax) -> None:
        """Should raise when more than one rack has the name"""
        name = str(uuid4())
        client.racks.create([rack.Rack(name=name), rack.Rack(name=name)])
        with pytest.raises(
            sy.MultipleFoundError,
            match=rf"Expected one rack where .*{name}.*, but found 2",
        ):
            client.racks.retrieve(name=name)

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a rack"""
        rack = client.racks.create(name="test")
        client.racks.delete([rack.key])
        with pytest.raises(sy.NotFoundError):
            client.racks.retrieve(key=rack.key)

    def test_delete_rack_attached(self, client: sy.Synnax) -> None:
        """Should raise a validation error if devices are attached to the rack"""
        rack = client.racks.create(name="test")
        client.devices.create(
            key=str(uuid4()),
            name="test",
            make="ni",
            model="pxi",
            rack=rack.key,
            location="dev1",
            properties={"foo": "bar"},
        )
        with pytest.raises(
            sy.ValidationError,
            match="cannot delete rack test because device test uses it as its rack",
        ):
            client.racks.delete([rack.key])

    def test_delete_task_attached(self, client: sy.Synnax) -> None:
        """Should raise a validation error if tasks are attached to the rack"""
        rack = client.racks.create(name="test")
        client.tasks.create(
            name="test",
            rack=rack.key,
            type="pagerduty_alert",
            config={"routing_key": "rk"},
        )
        with pytest.raises(sy.ValidationError, match="still holds tasks"):
            client.racks.delete([rack.key])

    def test_create_with_integrations(self, client: sy.Synnax) -> None:
        """Should create a rack with integrations and persist them"""
        rack = client.racks.create(
            racks=sy.Rack(name="integ-rack", integrations=["ni", "opc", "modbus"])
        )
        res = client.racks.retrieve(key=rack.key)
        assert res.integrations == ["ni", "opc", "modbus"]

    def test_retrieve_by_integration(self, client: sy.Synnax) -> None:
        """Should filter racks by supported integration"""
        r1 = client.racks.create(
            racks=sy.Rack(name="py-ni-rack", integrations=["ni", "opc"])
        )
        r2 = client.racks.create(
            racks=sy.Rack(name="py-modbus-rack", integrations=["modbus"])
        )
        results = client.racks.retrieve(keys=[r1.key, r2.key], integration="ni")
        result_keys = {r.key for r in results}
        assert r1.key in result_keys
        assert r2.key not in result_keys

    def test_retrieve_where(self, client: sy.Synnax) -> None:
        """Should keep racks that match a filter tree and the other filters"""
        racks = client.racks.create(
            racks=[
                sy.Rack(name=f"py-where-{uuid4()}", integrations=["labjack"]),
                sy.Rack(name=f"py-where-{uuid4()}"),
                sy.Rack(name=f"py-where-{uuid4()}"),
            ]
        )
        where = rack.Where(
            or_=[
                rack.Where(integrations=filter.Contains[str](any=["labjack"])),
                rack.Where(name=filter.String(eq=racks[1].name)),
            ]
        )
        res = client.racks.retrieve(where=where, keys=[r.key for r in racks])
        assert {r.key for r in res} == {racks[0].key, racks[1].key}

    def test_retrieve_missing_key(self, client: sy.Synnax) -> None:
        """Should treat a missing key as a non-match in a list retrieve"""
        r = client.racks.create(name="py-missing-key")
        res = client.racks.retrieve(keys=[r.key, 4123456789])
        assert [x.key for x in res] == [r.key]

    def test_create_existing_key(self, client: sy.Synnax) -> None:
        """Should refuse a rack whose key exists and keep the stored rack."""
        rack = client.racks.create(
            racks=sy.Rack(name="py-existing-key", integrations=["ni"])
        )
        with pytest.raises(sy.ValidationError, match=f"rack {rack.key} already exists"):
            client.racks.create(
                racks=sy.Rack(
                    key=rack.key, name="py-existing-key", integrations=["ni", "opc"]
                )
            )
        assert client.racks.retrieve(key=rack.key).integrations == ["ni"]

    def test_retrieve_embedded_rack(self, client: sy.Synnax) -> None:
        rack = client.racks.retrieve_embedded_rack()
        assert isinstance(rack, sy.Rack)
        # Cache should return the same rack
        cached_rack = client.racks.retrieve_embedded_rack()
        assert rack.key == cached_rack.key

    def test_retrieve_with_status(self, client: sy.Synnax) -> None:
        """Should attach the rack status when asked for it."""
        rack = client.racks.create(name=str(uuid4()))
        assert client.racks.retrieve(key=rack.key).status is None
        res = client.racks.retrieve(key=rack.key, include_status=True)
        assert res.status is not None
        assert res.status.owner == sy.rack.resource_id(rack.key)

    def test_retrieve_with_limit(self, client: sy.Synnax) -> None:
        """Should return at most the limit of the matching racks."""
        racks = client.racks.create(
            racks=[sy.Rack(name=str(uuid4())) for _ in range(3)]
        )
        keys = [r.key for r in racks]
        assert len(client.racks.retrieve(keys=keys, limit=2)) == 2

    def test_retrieve_by_names_with_limit(self, client: sy.Synnax) -> None:
        """Should return a list when names are combined with a limit."""
        name = str(uuid4())
        rack = client.racks.create(name=name)
        res = client.racks.retrieve(names=[name], limit=1)
        assert_type(res, list[sy.Rack])
        assert [r.key for r in res] == [rack.key]

    def test_retrieve_no_filters(self, client: sy.Synnax) -> None:
        """Should return a list of every rack when no filter is given."""
        rack = client.racks.create(name=str(uuid4()))
        res = client.racks.retrieve()
        assert_type(res, list[sy.Rack])
        assert rack.key in {r.key for r in res}

    def test_retrieve_integrations(self, client: sy.Synnax) -> None:
        """Should keep racks that support an integration given as a plain value."""
        r1 = client.racks.create(racks=sy.Rack(name="a", integrations=["ni", "opc"]))
        r2 = client.racks.create(racks=sy.Rack(name="b", integrations=["modbus"]))
        keys = [r1.key, r2.key]
        res = client.racks.retrieve(key=keys, integrations="ni")
        assert [r.key for r in res] == [r1.key]
        res = client.racks.retrieve(key=keys, integrations=["opc", "modbus"])
        assert {r.key for r in res} == {r1.key, r2.key}

    def test_retrieve_plain_key_beside_fields(self, client: sy.Synnax) -> None:
        """Should return one rack for a plain key beside other fields."""
        rack = client.racks.create(name=str(uuid4()))
        assert client.racks.retrieve(key=rack.key, embedded=False).key == rack.key
        with pytest.raises(sy.NotFoundError):
            client.racks.retrieve(key=rack.key, name="other")

    def test_retrieve_one_with_limit(self, client: sy.Synnax) -> None:
        """Should reject a plain name beside a limit, since a page could hide a
        match."""
        with pytest.raises(sy.ValidationError, match="cannot set a limit"):
            client.racks.retrieve(name="test", limit=1)  # type: ignore[call-overload]

    def test_retrieve_none_field(self, client: sy.Synnax) -> None:
        """Should reject a field set to None, since it would select every rack."""
        with pytest.raises(sy.ValidationError, match="embedded is None"):
            client.racks.retrieve(embedded=None)  # type: ignore[call-overload]

    def test_retrieve_single_with_status_typed(self, client: sy.Synnax) -> None:
        """Should return one rack when a key or name is given."""
        rack = client.racks.create(name=str(uuid4()))
        res = client.racks.retrieve(name=rack.name, include_status=True)
        assert_type(res, sy.Rack)
        assert res.key == rack.key
        by_key = client.racks.retrieve(rack.key, include_status=True)
        assert_type(by_key, sy.Rack)
        assert by_key.key == rack.key

    def test_retrieve_by_search_term(self, client: sy.Synnax) -> None:
        """Should return a list of the racks whose names match the search term."""
        name = f"searchable-rack-{uuid4()}"
        rack = client.racks.create(name=name)

        def check() -> None:
            res = client.racks.retrieve(search_term=name)
            assert rack.key in {r.key for r in res}

        assert_eventually(check)


@pytest.mark.rack
class TestRackGroup:
    def test_create_in_group(self, client: sy.Synnax) -> None:
        """Should create a rack in a group and retrieve it by that group."""
        g = client.groups.create(str(uuid4()), scope="device")
        rack = client.racks.create(name=str(uuid4()), group=g.key)
        assert rack.group == g.key
        assert [r.key for r in client.racks.retrieve(groups=[g.key])] == [rack.key]

    def test_set_group(self, client: sy.Synnax) -> None:
        """Should move a rack into a group and back out of it."""
        g = client.groups.create(str(uuid4()), scope="device")
        rack = client.racks.create(name=str(uuid4()))
        assert rack.group is None
        client.racks.set_group([rack.key], g.key)
        assert client.racks.retrieve(rack.key).group == g.key
        client.racks.set_group(rack.key)
        assert client.racks.retrieve(rack.key).group is None
