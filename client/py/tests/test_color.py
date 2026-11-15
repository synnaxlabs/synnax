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
import x.color
from synnax import color


def test_reexports_x_color() -> None:
    """Should re-export every public name of x.color."""
    for name in x.color.__all__:
        assert getattr(color, name) is getattr(x.color, name)
        assert name in color.__all__


@pytest.mark.color
class TestColorClient:
    """Tests for the named color client."""

    def test_create_from_kwargs(self, client: sy.Synnax) -> None:
        """Should create a named color from a name and its theme values."""
        name = str(uuid4())
        named = client.colors.create(name=name, light="#FF0000", dark="#880000")
        assert named.name == name
        assert named.light == sy.Color("#FF0000")
        assert named.dark == sy.Color("#880000")

    @pytest.mark.parametrize("kwargs", [{"name": "Nominal"}, {"light": "#FF0000"}])
    def test_create_requires_name_and_light(
        self, client: sy.Synnax, kwargs: dict[str, str]
    ) -> None:
        """Should reject a keyword create that leaves out the name or the light
        color."""
        with pytest.raises(sy.ValidationError, match="name and a light color"):
            client.colors.create(**kwargs)

    def test_create_many(self, client: sy.Synnax) -> None:
        """Should create a list of named colors and return a list."""
        colors = [
            color.Named(name=str(uuid4()), light=sy.Color("#0000FF")) for _ in range(2)
        ]
        created = client.colors.create(colors)
        assert [n.key for n in created] == [n.key for n in colors]

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve a named color by its key."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        assert client.colors.retrieve(key=named.key) == named

    def test_retrieve_by_name(self, client: sy.Synnax) -> None:
        """Should retrieve a named color by its name."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        assert client.colors.retrieve(name=named.name) == named

    def test_retrieve_by_search_term(self, client: sy.Synnax) -> None:
        """Should retrieve named colors whose names match a search term."""
        named = client.colors.create(name=f"searchable {uuid4()}", light="#FF0000")
        res = client.colors.retrieve(search_term=named.name)
        assert named.key in {r.key for r in res}

    def test_retrieve_by_palette(self, client: sy.Synnax) -> None:
        """Should retrieve the named colors of a palette, or the ones in no palette."""
        palette = client.groups.create(str(uuid4()), scope="color")
        in_palette = client.colors.create(
            color.Named(name=str(uuid4()), light=sy.Color("#FF0000"), group=palette.key)
        )
        loose = client.colors.create(name=str(uuid4()), light="#FF0000")
        assert client.colors.retrieve(groups=palette.key) == [in_palette]
        ungrouped = client.colors.retrieve(
            key=[in_palette.key, loose.key], grouped=False
        )
        assert ungrouped == [loose]

    def test_retrieve_name_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when no named color has the name."""
        with pytest.raises(sy.NotFoundError, match="Found no named color"):
            client.colors.retrieve(name=str(uuid4()))

    def test_dispatch(self, client: sy.Synnax) -> None:
        """Should apply actions to a named color."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        client.colors.dispatch(
            named.key,
            [
                color.set(color.SetPayload(path=["name"], value="Anomaly")),
                color.set(color.SetPayload(path=["dark"], value="#00FF00")),
            ],
        )
        stored = client.colors.retrieve(key=named.key)
        assert stored.name == "Anomaly"
        assert stored.dark == sy.Color("#00FF00")

    def test_dispatch_invalid(self, client: sy.Synnax) -> None:
        """Should reject an action that leaves the named color invalid."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        with pytest.raises(sy.ValidationError, match="name: required"):
            client.colors.dispatch(
                named.key, color.set(color.SetPayload(path=["name"], value=""))
            )

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a named color."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        client.colors.delete(named.key)
        with pytest.raises(sy.NotFoundError):
            client.colors.retrieve(key=named.key)
