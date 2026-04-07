#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy


@pytest.mark.view
class TestView:
    def test_create(self, client: sy.Synnax):
        """Should create a single view."""
        v = client.views.create(
            name="Test View",
            type="lineplot",
            query={"channels": ["test-1", "test-2"]},
        )
        assert v.name == "Test View"
        assert v.type == "lineplot"
        assert v.query == {"channels": ["test-1", "test-2"]}

    def test_create_from_view(self, client: sy.Synnax):
        """Should create a view from a View object."""
        v = client.views.create(
            sy.View(
                key=sy.view.Key(int=0),
                name="From Object",
                type="table",
                query={"columns": ["col-1"]},
            )
        )
        assert v.name == "From Object"
        assert v.type == "table"

    def test_create_multiple(self, client: sy.Synnax):
        """Should create multiple views."""
        views = client.views.create(
            [
                sy.View(
                    key=sy.view.Key(int=0),
                    name="Multi 1",
                    type="lineplot",
                    query={"channels": ["ch-1"]},
                ),
                sy.View(
                    key=sy.view.Key(int=0),
                    name="Multi 2",
                    type="table",
                    query={"columns": ["col-1"]},
                ),
            ]
        )
        assert len(views) == 2
        assert views[0].name == "Multi 1"
        assert views[1].name == "Multi 2"

    def test_retrieve_by_key(self, client: sy.Synnax):
        """Should retrieve a view by key."""
        created = client.views.create(
            name="Retrieve Me",
            type="lineplot",
            query={"channels": ["test"]},
        )
        retrieved = client.views.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == "Retrieve Me"
        assert retrieved.query == {"channels": ["test"]}

    def test_retrieve_by_keys(self, client: sy.Synnax):
        """Should retrieve multiple views by keys."""
        v1 = client.views.create(
            name="View A",
            type="lineplot",
            query={"channels": ["a"]},
        )
        v2 = client.views.create(
            name="View B",
            type="table",
            query={"columns": ["b"]},
        )
        views = client.views.retrieve(keys=[v1.key, v2.key])
        assert len(views) == 2

    def test_retrieve_by_type(self, client: sy.Synnax):
        """Should retrieve views filtered by type."""
        unique_type = f"test_type_{sy.view.Key(int=0)}"
        client.views.create(
            name="Typed View",
            type=unique_type,
            query={},
        )
        views = client.views.retrieve(types=[unique_type])
        assert len(views) >= 1
        assert all(v.type == unique_type for v in views)

    def test_delete(self, client: sy.Synnax):
        """Should delete a view."""
        v = client.views.create(
            name="To Delete",
            type="table",
            query={},
        )
        client.views.delete(v.key)
        with pytest.raises(Exception):
            client.views.retrieve(key=v.key)

    def test_delete_multiple(self, client: sy.Synnax):
        """Should delete multiple views."""
        v1 = client.views.create(
            name="Delete 1",
            type="table",
            query={},
        )
        v2 = client.views.create(
            name="Delete 2",
            type="lineplot",
            query={},
        )
        client.views.delete([v1.key, v2.key])
        with pytest.raises(Exception):
            client.views.retrieve(keys=[v1.key, v2.key])
