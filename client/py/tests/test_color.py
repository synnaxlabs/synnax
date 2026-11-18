#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pydantic
import pytest

import synnax as sy
import x.color
from synnax import color
from x.testutil import untyped


def test_reexports_x_color() -> None:
    """Should re-export every public name of x.color."""
    for name in x.color.__all__:
        assert getattr(color, name) is getattr(x.color, name)
        assert name in color.__all__


class TestToRef:
    """Tests for converting a color to a reference."""

    def test_hex(self) -> None:
        """Should store a hex color as a fixed color."""
        assert color.to_ref("#FF0000") == color.FixedRef(value=color.Color("#FF0000"))

    def test_ref(self) -> None:
        """Should return a reference as it is."""
        ref = color.NamedRef(key=uuid4())
        assert color.to_ref(ref) is ref

    def test_invalid(self) -> None:
        """Should raise a validation error for a value that is not a color."""
        with pytest.raises(pydantic.ValidationError, match="Invalid color"):
            color.to_ref("not a color")


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

    def test_create_one(self, client: sy.Synnax) -> None:
        """Should create a single named color and return it."""
        named = color.Named(name=str(uuid4()), light=sy.Color("#00FF00"))
        assert client.colors.create(named) == named

    def test_create_many(self, client: sy.Synnax) -> None:
        """Should create a list of named colors and return a list."""
        colors = [
            color.Named(name=str(uuid4()), light=sy.Color("#0000FF")) for _ in range(2)
        ]
        created = client.colors.create(colors)
        assert [n.key for n in created] == [n.key for n in colors]

    def test_create_existing_key(self, client: sy.Synnax) -> None:
        """Should refuse a named color whose key exists and keep the stored one."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        with pytest.raises(
            sy.ValidationError, match=f"color {named.key} already exists"
        ):
            client.colors.create(named.model_copy(update={"name": str(uuid4())}))
        assert client.colors.retrieve(key=named.key).name == named.name

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve a named color by its key."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        assert client.colors.retrieve(key=named.key) == named

    def test_retrieve_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve several named colors by their keys."""
        a = client.colors.create(name=str(uuid4()), light="#FF0000")
        b = client.colors.create(name=str(uuid4()), light="#FF0000")
        res = client.colors.retrieve(keys=[a.key, b.key])
        assert {n.key for n in res} == {a.key, b.key}

    def test_retrieve_where(self, client: sy.Synnax) -> None:
        """Should keep the named colors that match the where and the other filters."""
        a = client.colors.create(name=str(uuid4()), light="#FF0000")
        b = client.colors.create(name=str(uuid4()), light="#FF0000")
        where = color.Where(not_=color.Where(key=sy.filter.Set[color.Key](eq=a.key)))
        res = client.colors.retrieve(where=where, names=[a.name, b.name])
        assert [n.key for n in res] == [b.key]

    def test_retrieve_by_name(self, client: sy.Synnax) -> None:
        """Should retrieve a named color by its name."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        assert client.colors.retrieve(name=named.name) == named

    def test_retrieve_by_names(self, client: sy.Synnax) -> None:
        """Should retrieve several named colors by their names."""
        a = client.colors.create(name=str(uuid4()), light="#FF0000")
        b = client.colors.create(name=str(uuid4()), light="#FF0000")
        res = client.colors.retrieve(names=[a.name, b.name])
        assert {n.key for n in res} == {a.key, b.key}

    def test_retrieve_or(self, client: sy.Synnax) -> None:
        """Should retrieve the named colors that match any filter in or_."""
        a = client.colors.create(name=str(uuid4()), light="#FF0000")
        b = client.colors.create(name=str(uuid4()), light="#FF0000")
        res = client.colors.retrieve(or_=[{"name": a.name}, {"key": [b.key]}])
        assert sorted(n.key for n in res) == sorted([a.key, b.key])

    def test_retrieve_one(self, client: sy.Synnax) -> None:
        """Should return one named color for any filter when one is set."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        found = client.colors.retrieve(name={"starts_with": named.name}, one=True)
        assert found == named
        assert client.colors.retrieve(keys=[named.key], one=True) == named

    @pytest.mark.parametrize(
        "page",
        [{"limit": 1}, {"after": color.Cursor(key=uuid4())}],
    )
    def test_retrieve_one_with_page(
        self, client: sy.Synnax, page: dict[str, object]
    ) -> None:
        """Should reject one beside a limit or a cursor, since a page could hide a
        match."""
        with pytest.raises(sy.ValidationError, match="cannot set a limit or a cursor"):
            untyped(client.colors.retrieve)(name={"starts_with": "a"}, one=True, **page)

    def test_retrieve_name_multiple_found(self, client: sy.Synnax) -> None:
        """Should raise MultipleFoundError when several named colors share the
        name."""
        name = str(uuid4())
        client.colors.create(name=name, light="#FF0000")
        client.colors.create(name=name, light="#00FF00")
        with pytest.raises(sy.MultipleFoundError):
            client.colors.retrieve(name=name)

    def test_retrieve_order_and_after(self, client: sy.Synnax) -> None:
        """Should page through the matching named colors in name order from a
        cursor."""
        prefix = uuid4()
        created = client.colors.create(
            [
                color.Named(name=f"{prefix} {n}", light=sy.Color("#FF0000"))
                for n in "CAB"
            ]
        )
        keys = [n.key for n in created]
        order = color.Order(field="name")
        first = client.colors.retrieve(key=keys, order=order, limit=2)
        assert [n.name for n in first] == [f"{prefix} A", f"{prefix} B"]
        after = color.Cursor(key=first[-1].key, name=first[-1].name)
        second = client.colors.retrieve(key=keys, order=order, after=after)
        assert [n.name for n in second] == [f"{prefix} C"]

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

    def test_retrieve_every(self, client: sy.Synnax) -> None:
        """Should retrieve every named color when given no arguments."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        assert named.key in {n.key for n in client.colors.retrieve()}

    def test_retrieve_drops_missing_keys(self, client: sy.Synnax) -> None:
        """Should leave the keys with no named color out of a list."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        assert client.colors.retrieve(keys=[named.key, uuid4()]) == [named]

    def test_retrieve_key_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when no named color has the key."""
        with pytest.raises(sy.NotFoundError):
            client.colors.retrieve(key=uuid4())

    def test_retrieve_none_field(self, client: sy.Synnax) -> None:
        """Should reject a field set to None, since it would select every named
        color."""
        with pytest.raises(sy.ValidationError, match="name is None"):
            untyped(client.colors.retrieve)(name=None)

    def test_retrieve_unknown_field(self, client: sy.Synnax) -> None:
        """Should reject a field that is not a named color filter line."""
        with pytest.raises(pydantic.ValidationError, match="nme"):
            untyped(client.colors.retrieve)(nme="a")

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

    def test_dispatch_missing_key(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when no named color has the key."""
        with pytest.raises(sy.NotFoundError):
            client.colors.dispatch(
                uuid4(), color.set(color.SetPayload(path=["name"], value="Anomaly"))
            )

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a named color."""
        named = client.colors.create(name=str(uuid4()), light="#FF0000")
        client.colors.delete(named.key)
        with pytest.raises(sy.NotFoundError):
            client.colors.retrieve(key=named.key)

    def test_delete_many(self, client: sy.Synnax) -> None:
        """Should delete several named colors at once."""
        a = client.colors.create(name=str(uuid4()), light="#FF0000")
        b = client.colors.create(name=str(uuid4()), light="#FF0000")
        client.colors.delete([a.key, b.key])
        assert client.colors.retrieve(names=[a.name, b.name]) == []

    def test_delete_missing(self, client: sy.Synnax) -> None:
        """Should not fail on a key with no named color."""
        client.colors.delete(uuid4())
