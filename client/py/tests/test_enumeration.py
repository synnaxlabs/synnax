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
from synnax import color, enumeration
from x.testutil import untyped


def _entries() -> list[enumeration.Entry]:
    return [
        enumeration.Entry(key="closed", value=0, name="Closed"),
        enumeration.Entry(key="open", value=1, name="Open"),
    ]


@pytest.mark.enumeration
class TestEnumerationClient:
    """Tests for the enumeration client."""

    def test_create_from_kwargs(self, client: sy.Synnax) -> None:
        """Should create an enumeration from a name and its entries."""
        name = str(uuid4())
        created = client.enumerations.create(name=name, entries=_entries())
        assert created.name == name
        assert [e.name for e in created.entries] == ["Closed", "Open"]

    def test_create_requires_name(self, client: sy.Synnax) -> None:
        """Should reject a keyword create that leaves out the name."""
        with pytest.raises(sy.ValidationError, match="needs a name"):
            untyped(client.enumerations.create)(entries=_entries())

    def test_create_many(self, client: sy.Synnax) -> None:
        """Should create a list of enumerations and return a list."""
        enumerations = [enumeration.Enumeration(name=str(uuid4())) for _ in range(2)]
        created = client.enumerations.create(enumerations)
        assert [e.key for e in created] == [e.key for e in enumerations]

    def test_create_duplicate_value(self, client: sy.Synnax) -> None:
        """Should refuse two entries with one value."""
        entries = [
            enumeration.Entry(key="a", value=1, name="A"),
            enumeration.Entry(key="b", value=1, name="B"),
        ]
        with pytest.raises(sy.ValidationError, match="another entry has the value 1"):
            client.enumerations.create(name=str(uuid4()), entries=entries)

    @pytest.mark.parametrize("value", [float("nan"), float("inf"), float("-inf")])
    def test_entry_refuses_non_finite_value(self, value: float) -> None:
        """Should refuse an entry value that is NaN or infinite."""
        with pytest.raises(pydantic.ValidationError, match="finite number"):
            enumeration.Entry(key="a", value=value, name="A")

    def test_create_entry_color(self, client: sy.Synnax) -> None:
        """Should store a fixed color on an entry."""
        entries = [
            enumeration.Entry(
                key="tripped",
                value=2,
                name="Tripped",
                abnormal=True,
                color=color.FixedRef(value=sy.Color("#FF0000")),
            )
        ]
        created = client.enumerations.create(name=str(uuid4()), entries=entries)
        stored = client.enumerations.retrieve(key=created.key)
        assert stored.entries == entries

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve an enumeration by its key."""
        created = client.enumerations.create(name=str(uuid4()), entries=_entries())
        assert client.enumerations.retrieve(key=created.key) == created

    def test_retrieve_by_name(self, client: sy.Synnax) -> None:
        """Should retrieve an enumeration by its name."""
        created = client.enumerations.create(name=str(uuid4()))
        assert client.enumerations.retrieve(name=created.name) == created

    def test_retrieve_by_names(self, client: sy.Synnax) -> None:
        """Should retrieve several enumerations by their names."""
        a = client.enumerations.create(name=str(uuid4()))
        b = client.enumerations.create(name=str(uuid4()))
        res = client.enumerations.retrieve(names=[a.name, b.name])
        assert {e.key for e in res} == {a.key, b.key}

    def test_retrieve_builtins(self, client: sy.Synnax) -> None:
        """Should retrieve the built-in enumerations that the Core creates."""
        valve = client.enumerations.retrieve(name="Valve")
        assert [(e.value, e.name) for e in valve.entries] == [
            (0, "Closed"),
            (1, "Open"),
        ]

    def test_retrieve_key_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when no enumeration has the key."""
        with pytest.raises(sy.NotFoundError):
            client.enumerations.retrieve(key=uuid4())

    def test_dispatch(self, client: sy.Synnax) -> None:
        """Should apply actions to an enumeration."""
        created = client.enumerations.create(name=str(uuid4()), entries=_entries())
        client.enumerations.dispatch(
            created.key,
            [
                enumeration.set(enumeration.SetPayload(path=["name"], value="Gate")),
                enumeration.insert(
                    enumeration.InsertPayload(
                        path=["entries"],
                        value={
                            "key": "fault",
                            "value": 2,
                            "name": "Fault",
                            "abnormal": True,
                        },
                        after="open",
                    )
                ),
            ],
        )
        stored = client.enumerations.retrieve(key=created.key)
        assert stored.name == "Gate"
        assert [e.key for e in stored.entries] == ["closed", "open", "fault"]

    def test_dispatch_duplicate_key(self, client: sy.Synnax) -> None:
        """Should reject an action that gives two entries one key."""
        created = client.enumerations.create(name=str(uuid4()), entries=_entries())
        with pytest.raises(
            sy.ValidationError, match='another entry has the key "open"'
        ):
            client.enumerations.dispatch(
                created.key,
                enumeration.set(
                    enumeration.SetPayload(
                        path=["entries", "closed", "key"], value="open"
                    )
                ),
            )

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete an enumeration."""
        created = client.enumerations.create(name=str(uuid4()))
        client.enumerations.delete(created.key)
        with pytest.raises(sy.NotFoundError):
            client.enumerations.retrieve(key=created.key)

    def test_delete_builtin(self, client: sy.Synnax) -> None:
        """Should refuse to delete a built-in enumeration."""
        valve = client.enumerations.retrieve(name="Valve")
        with pytest.raises(sy.ValidationError, match="built-in enumeration Valve"):
            client.enumerations.delete(valve.key)

    def test_delete_missing(self, client: sy.Synnax) -> None:
        """Should not fail on a key with no enumeration."""
        client.enumerations.delete(uuid4())
