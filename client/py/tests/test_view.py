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
from synnax.view import RangeQuery, RangeView, StatusQuery, StatusView
from x.testutil import assert_eventually


@pytest.mark.view
class TestView:
    def test_create(self, client: sy.Synnax) -> None:
        """Should create a single view."""
        label = uuid4()
        v = client.views.create(
            RangeView(name="Test View", query=RangeQuery(has_labels=[label]))
        )
        assert isinstance(v, RangeView)
        assert v.name == "Test View"
        assert v.query.has_labels == [label]

    def test_create_multiple(self, client: sy.Synnax) -> None:
        """Should create multiple views of different types."""
        views = client.views.create(
            [
                RangeView(name="Multi 1", query=RangeQuery()),
                StatusView(name="Multi 2", query=StatusQuery(variants=["error"])),
            ]
        )
        assert len(views) == 2
        assert isinstance(views[0], RangeView)
        assert isinstance(views[1], StatusView)
        assert views[1].query.variants == ["error"]

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve a view by key."""
        created = client.views.create(
            StatusView(name="Retrieve Me", query=StatusQuery(variants=["success"]))
        )
        assert client.views.retrieve(key=created.key) == created

    def test_retrieve_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve multiple views by keys."""
        v1 = client.views.create(RangeView(name="View A", query=RangeQuery()))
        v2 = client.views.create(StatusView(name="View B", query=StatusQuery()))
        views = client.views.retrieve(keys=[v1.key, v2.key])
        assert len(views) == 2

    def test_retrieve_with_search_term(self, client: sy.Synnax) -> None:
        """Should search for views by name."""
        prefix = f"searchable-view-{uuid4()}"
        client.views.create(
            [
                RangeView(name=f"{prefix}-1", query=RangeQuery()),
                StatusView(name=f"{prefix}-2", query=StatusQuery()),
            ]
        )

        def check() -> None:
            results = client.views.retrieve(search_term=prefix)
            assert len(results) >= 2
            assert all(prefix in v.name for v in results)

        assert_eventually(check)

    def test_retrieve_by_type(self, client: sy.Synnax) -> None:
        """Should retrieve views filtered by type."""
        views = client.views.create(
            [
                RangeView(name="Range", query=RangeQuery()),
                StatusView(name="Status", query=StatusQuery()),
            ]
        )
        res = client.views.retrieve(keys=[v.key for v in views], types=["status"])
        assert len(res) == 1
        assert res[0].name == "Status"

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a view."""
        v = client.views.create(RangeView(name="To Delete", query=RangeQuery()))
        client.views.delete(v.key)
        with pytest.raises(sy.NotFoundError):
            client.views.retrieve(key=v.key)

    def test_delete_multiple(self, client: sy.Synnax) -> None:
        """Should delete multiple views."""
        v1 = client.views.create(RangeView(name="Delete 1", query=RangeQuery()))
        v2 = client.views.create(StatusView(name="Delete 2", query=StatusQuery()))
        client.views.delete([v1.key, v2.key])
        with pytest.raises(sy.NotFoundError):
            client.views.retrieve(keys=[v1.key, v2.key])
