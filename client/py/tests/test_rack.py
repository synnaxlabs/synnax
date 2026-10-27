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
            match=f"Expected one Rack matching {name}, but found 2",
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
            match="cannot delete rack when devices are still attached",
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
        with pytest.raises(sy.ValidationError, match="tasks are still attached"):
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

    def test_update_integrations(self, client: sy.Synnax) -> None:
        """Should update integrations on upsert"""
        rack = client.racks.create(
            racks=sy.Rack(name="py-upsert-integ", integrations=["ni"])
        )
        client.racks.create(
            racks=sy.Rack(
                key=rack.key, name="py-upsert-integ", integrations=["ni", "opc", "arc"]
            )
        )
        res = client.racks.retrieve(key=rack.key)
        assert res.integrations == ["ni", "opc", "arc"]

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
        assert res.status.details is not None
        assert res.status.details.rack == rack.key

    def test_retrieve_with_limit_and_offset(self, client: sy.Synnax) -> None:
        """Should page through the matching racks."""
        racks = client.racks.create(
            racks=[sy.Rack(name=str(uuid4())) for _ in range(3)]
        )
        keys = [r.key for r in racks]
        assert len(client.racks.retrieve(keys=keys, limit=2)) == 2
        assert len(client.racks.retrieve(keys=keys, offset=2)) == 1

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

    def test_retrieve_singular_with_list_filter(self, client: sy.Synnax) -> None:
        """Should raise a ValueError when singular selectors are mixed with others."""
        retrieve = cast(Callable[..., object], client.racks.retrieve)
        with pytest.raises(ValueError, match="cannot be combined"):
            retrieve(name="test", limit=1)
        with pytest.raises(ValueError, match="only one of"):
            retrieve(key=1, name="test")

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


class TestBuilder:
    def test_join(self) -> None:
        """Should join rack conditions into one Where tree."""
        where = filter.join(
            rack.Where,
            [rack.integrations.has("ni"), rack.host_is_node.eq(True)],
        )
        assert where == rack.Where(
            and_=[
                rack.Where(integrations=filter.Contains[str](any=["ni"])),
                rack.Where(host_is_node=True),
            ]
        )
