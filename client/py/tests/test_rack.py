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


@pytest.mark.rack
class TestRackClient:
    def test_create_single(self, client: sy.Synnax):
        """It should correctly create a single rack"""
        rack = client.racks.create(name="test")
        assert rack.key != 0

    def test_create_multiple(self, client: sy.Synnax):
        """Should create multiple racks"""
        r1 = sy.Rack(name="test1")
        r2 = sy.Rack(name="test2")
        racks = client.racks.create(racks=[r1, r2])
        assert len(racks) == 2
        assert racks[0].name == "test1"
        assert racks[1].name == "test2"

    def test_retrieve_by_name(self, client: sy.Synnax):
        """Should retrieve a rack by name"""
        name = str(uuid4())
        rack = client.racks.create(name=name)
        res = client.racks.retrieve(name=name)
        assert res.name == name
        assert res.key == rack.key

    def test_retrieve_by_key(self, client: sy.Synnax):
        """Should retrieve a rack by key"""
        rack = client.racks.create(name="test")
        res = client.racks.retrieve(key=rack.key)
        assert res.key == rack.key
        assert res.name == rack.name

    def test_retrieve_multiple(self, client: sy.Synnax):
        """Should retrieve multiple racks"""
        r1 = client.racks.create(name="test1")
        r2 = client.racks.create(name="test2")
        racks = client.racks.retrieve(keys=[r1.key, r2.key])
        assert len(racks) == 2
        assert {r.key for r in racks} == {r1.key, r2.key}

    def test_delete(self, client: sy.Synnax):
        """Should delete a rack"""
        rack = client.racks.create(name="test")
        client.racks.delete([rack.key])
        with pytest.raises(sy.NotFoundError):
            client.racks.retrieve(key=rack.key)

    def test_delete_rack_attached(self, client: sy.Synnax):
        """Should raise a validation error if devices are attached to the rack"""
        rack = client.racks.create(name="test")
        client.devices.create(
            key=str(uuid4()), name="test", rack=rack.key, location="dev1"
        )
        with pytest.raises(
            sy.ValidationError,
            match="cannot delete rack when devices are still attached",
        ):
            client.racks.delete([rack.key])

    def test_delete_task_attached(self, client: sy.Synnax):
        """Should raise a validation error if tasks are attached to the rack"""
        rack = client.racks.create(name="test")
        client.tasks.create(name="test", rack=rack.key)
        with pytest.raises(sy.ValidationError, match="tasks are still attached"):
            client.racks.delete([rack.key])

    def test_retrieve_embedded_rack(self, client: sy.Synnax):
        rack = client.racks.retrieve_embedded_rack()
        assert isinstance(rack, sy.Rack)
        # Cache should return the same rack
        cached_rack = client.racks.retrieve_embedded_rack()
        assert rack.key == cached_rack.key
