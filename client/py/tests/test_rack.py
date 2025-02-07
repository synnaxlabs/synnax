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


@pytest.mark.rack
class TestRackClient:
    def test_create_single(self, client: sy.Synnax):
        rack = client.hardware.racks.create(name="test")
        assert rack.key != 0

    def test_create_multiple(self, client: sy.Synnax):
        r1 = sy.Rack(name="test1")
        r2 = sy.Rack(name="test2")
        racks = client.hardware.racks.create(racks=[r1, r2])
        assert len(racks) == 2
        assert racks[0].name == "test1"
        assert racks[1].name == "test2"


    def test_retrieve_by_name(self, client: sy.Synnax):
        name = str(uuid4())
        rack = client.hardware.racks.create(name=name)
        res = client.hardware.racks.retrieve(name=name)
        assert res.name == name
        assert res.key == rack.key

    def test_retrieve_by_key(self, client: sy.Synnax):
        rack = client.hardware.racks.create(name="test")
        res = client.hardware.racks.retrieve(key=rack.key)
        assert res.key == rack.key
        assert res.name == rack.name

    def test_retrieve_multiple(self, client: sy.Synnax):
        r1 = client.hardware.racks.create(name="test1")
        r2 = client.hardware.racks.create(name="test2")
        racks = client.hardware.racks.retrieve(keys=[r1.key, r2.key])
        assert len(racks) == 2
        assert {r.key for r in racks} == {r1.key, r2.key}

    def test_delete(self, client: sy.Synnax):
        rack = client.hardware.racks.create(name="test")
        client.hardware.racks.delete([rack.key])
        with pytest.raises(sy.NotFoundError):
            client.hardware.racks.retrieve(key=rack.key)

    def test_retrieve_embedded_rack(self, client: sy.Synnax):
        rack = client.hardware.racks.retrieve_embedded_rack()
        assert isinstance(rack, sy.Rack)
        # Cache should return the same rack
        cached_rack = client.hardware.racks.retrieve_embedded_rack()
        assert rack.key == cached_rack.key
