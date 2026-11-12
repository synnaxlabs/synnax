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
from synnax import label
from x.strings import random_name


@pytest.fixture
def rng(client: sy.Synnax) -> sy.Range:
    return client.ranges.create(
        name=random_name(),
        time_range=sy.TimeStamp.now().span_range(sy.TimeSpan.SECOND),
    )


def _labels_of(client: sy.Synnax, rng: sy.Range) -> list[sy.label.Key]:
    return client.ranges.retrieve(key=rng.key).labels


@pytest.mark.label
class TestLabelClient:
    """Tests for the label client."""

    def test_create_from_kwargs(self, client: sy.Synnax) -> None:
        """Should create a label from a name and a color."""
        name = str(uuid4())
        lbl = client.labels.create(name=name, color="#FF0000")
        assert lbl.name == name
        assert lbl.color == sy.Color("#FF0000")

    @pytest.mark.parametrize("kwargs", [{"name": "Nominal"}, {"color": "#FF0000"}, {}])
    def test_create_requires_name_and_color(
        self, client: sy.Synnax, kwargs: dict[str, str]
    ) -> None:
        """Should reject a keyword create that leaves out the name or the color."""
        with pytest.raises(sy.ValidationError, match="name and a color"):
            client.labels.create(**kwargs)

    def test_create_one(self, client: sy.Synnax) -> None:
        """Should create a single label and return it."""
        lbl = sy.Label(name=str(uuid4()), color=sy.Color("#00FF00"))
        assert client.labels.create(lbl) == lbl

    def test_create_many(self, client: sy.Synnax) -> None:
        """Should create a list of labels and return a list."""
        labels = [
            sy.Label(name=str(uuid4()), color=sy.Color("#0000FF")) for _ in range(2)
        ]
        created = client.labels.create(labels)
        assert [lbl.key for lbl in created] == [lbl.key for lbl in labels]

    def test_create_existing_key(self, client: sy.Synnax) -> None:
        """Should refuse a label whose key exists and keep the stored label."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        with pytest.raises(sy.ValidationError, match=f"label {lbl.key} already exists"):
            client.labels.create(lbl.model_copy(update={"name": str(uuid4())}))
        assert client.labels.retrieve(key=lbl.key).name == lbl.name

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve a label by its key."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        assert client.labels.retrieve(key=lbl.key) == lbl

    def test_retrieve_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve several labels by their keys."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        res = client.labels.retrieve(keys=[a.key, b.key])
        assert {lbl.key for lbl in res} == {a.key, b.key}

    def test_retrieve_where(self, client: sy.Synnax) -> None:
        """Should keep the labels that match the where and the other filters."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        where = label.Where(not_=label.Where(key=sy.filter.Set[label.Key](eq=a.key)))
        res = client.labels.retrieve(where=where, names=[a.name, b.name])
        assert [r.key for r in res] == [b.key]

    def test_retrieve_by_name(self, client: sy.Synnax) -> None:
        """Should retrieve a label by its name."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        assert client.labels.retrieve(name=lbl.name) == lbl

    def test_retrieve_by_names(self, client: sy.Synnax) -> None:
        """Should retrieve several labels by their names."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        res = client.labels.retrieve(names=[a.name, b.name])
        assert {lbl.key for lbl in res} == {a.key, b.key}

    def test_retrieve_by_search_term(self, client: sy.Synnax) -> None:
        """Should retrieve labels whose names match a search term."""
        lbl = client.labels.create(name=f"searchable {uuid4()}", color="#FF0000")
        res = client.labels.retrieve(search_term=lbl.name)
        assert lbl.key in {r.key for r in res}

    def test_retrieve_name_multiple_found(self, client: sy.Synnax) -> None:
        """Should raise MultipleFoundError when several labels share the name."""
        name = str(uuid4())
        client.labels.create(name=name, color="#FF0000")
        client.labels.create(name=name, color="#00FF00")
        with pytest.raises(sy.MultipleFoundError):
            client.labels.retrieve(name=name)

    def test_retrieve_key_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when no label has the key."""
        with pytest.raises(sy.NotFoundError):
            client.labels.retrieve(key=uuid4())

    def test_retrieve_name_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when no label has the name."""
        with pytest.raises(sy.NotFoundError):
            client.labels.retrieve(name=str(uuid4()))

    def test_retrieve_name_operator(self, client: sy.Synnax) -> None:
        """Should retrieve the labels whose name matches an operator."""
        prefix = str(uuid4())
        a = client.labels.create(name=f"{prefix}-a", color="#FF0000")
        b = client.labels.create(name=f"{prefix}-b", color="#FF0000")
        res = client.labels.retrieve(name={"starts_with": prefix})
        assert sorted(lbl.key for lbl in res) == sorted([a.key, b.key])

    def test_retrieve_name_list(self, client: sy.Synnax) -> None:
        """Should return a list for a list of names, even with one name."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        assert client.labels.retrieve(name=[lbl.name]) == [lbl]

    def test_retrieve_plain_key_beside_fields(self, client: sy.Synnax) -> None:
        """Should return one label for a plain key beside other fields."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        assert client.labels.retrieve(key=lbl.key, name={"in": [lbl.name]}) == lbl
        with pytest.raises(sy.NotFoundError):
            client.labels.retrieve(key=lbl.key, name="other")

    def test_retrieve_or(self, client: sy.Synnax) -> None:
        """Should retrieve the labels that match any filter in or_."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        res = client.labels.retrieve(or_=[{"name": a.name}, {"key": [b.key]}])
        assert sorted(lbl.key for lbl in res) == sorted([a.key, b.key])

    def test_retrieve_one(self, client: sy.Synnax) -> None:
        """Should return one label for any filter when one is set."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        assert client.labels.retrieve(name={"starts_with": lbl.name}, one=True) == lbl
        assert client.labels.retrieve(keys=[lbl.key], one=True) == lbl

    def test_retrieve_one_search_term_not_found(self, client: sy.Synnax) -> None:
        """Should name the search term when a search with one set matches nothing."""
        term = uuid4().hex
        with pytest.raises(
            sy.NotFoundError, match=rf'^Found no label matching "{term}"$'
        ):
            client.labels.retrieve(search_term=term, one=True)

    def test_retrieve_one_with_limit(self, client: sy.Synnax) -> None:
        """Should reject one beside a limit, since a page could hide a match."""
        with pytest.raises(sy.ValidationError, match="cannot set a limit"):
            client.labels.retrieve(name={"starts_with": "a"}, one=True, limit=1)  # type: ignore[call-overload]

    def test_retrieve_none_field(self, client: sy.Synnax) -> None:
        """Should reject a field set to None, since it would select every label."""
        with pytest.raises(sy.ValidationError, match="name is None"):
            client.labels.retrieve(name=None)  # type: ignore[call-overload]

    def test_retrieve_unknown_field(self, client: sy.Synnax) -> None:
        """Should reject a field that is not a label filter line."""
        with pytest.raises(pydantic.ValidationError, match="nme"):
            client.labels.retrieve(nme="a")  # type: ignore[call-overload]

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a label."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        client.labels.delete(lbl.key)
        with pytest.raises(sy.NotFoundError):
            client.labels.retrieve(key=lbl.key)

    def test_delete_many(self, client: sy.Synnax) -> None:
        """Should delete several labels at once."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        client.labels.delete([a.key, b.key])
        assert client.labels.retrieve(names=[a.name, b.name]) == []


@pytest.mark.label
class TestLabelResource:
    """Tests for adding labels to and removing labels from a range or a status."""

    def test_label(self, client: sy.Synnax, rng: sy.Range) -> None:
        """Should add labels to a range."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        client.labels.label(rng.resource_id, [a.key, b.key])
        assert set(_labels_of(client, rng)) == {a.key, b.key}

    def test_label_adds(self, client: sy.Synnax, rng: sy.Range) -> None:
        """Should keep the existing labels of a range when replace is false."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        client.labels.label(rng.resource_id, a.key)
        client.labels.label(rng.resource_id, b.key)
        assert set(_labels_of(client, rng)) == {a.key, b.key}

    def test_label_replace(self, client: sy.Synnax, rng: sy.Range) -> None:
        """Should remove every other label from the range when replace is true."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        client.labels.label(rng.resource_id, a.key)
        client.labels.label(rng.resource_id, b.key, replace=True)
        assert _labels_of(client, rng) == [b.key]

    def test_remove(self, client: sy.Synnax, rng: sy.Range) -> None:
        """Should remove labels from a range."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        client.labels.label(rng.resource_id, [a.key, b.key])
        client.labels.remove(rng.resource_id, a.key)
        assert _labels_of(client, rng) == [b.key]

    def test_label_string_id(self, client: sy.Synnax, rng: sy.Range) -> None:
        """Should accept the resource ID of the range as a string."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        client.labels.label(str(rng.resource_id), lbl.key)
        assert _labels_of(client, rng) == [lbl.key]

    def test_label_status(self, client: sy.Synnax) -> None:
        """Should add labels to and remove labels from a status."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        st = client.statuses.set(
            sy.Status(variant=sy.status.VARIANT_INFO, message="Labeled status")
        )
        client.labels.label(sy.status.resource_id(st.key), [a.key, b.key])
        client.labels.remove(sy.status.resource_id(st.key), a.key)
        assert client.statuses.retrieve(key=st.key).labels == [b.key]

    def test_label_other_type(
        self, client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        """Should reject a resource that is not a range or a status."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        with pytest.raises(sy.ValidationError, match="a channel cannot have labels"):
            client.labels.label(virtual_channel.resource_id, lbl.key)

    def test_delete_removes_from_range(self, client: sy.Synnax, rng: sy.Range) -> None:
        """Should remove a deleted label from every range that holds it."""
        a = client.labels.create(name=str(uuid4()), color="#FF0000")
        b = client.labels.create(name=str(uuid4()), color="#FF0000")
        client.labels.label(rng.resource_id, [a.key, b.key])
        client.labels.delete(a.key)
        assert _labels_of(client, rng) == [b.key]

    def test_delete_removes_from_status(self, client: sy.Synnax) -> None:
        """Should remove a deleted label from every status that holds it."""
        lbl = client.labels.create(name=str(uuid4()), color="#FF0000")
        st = client.statuses.set(
            sy.Status(
                variant=sy.status.VARIANT_INFO,
                message="Labeled status",
                labels=[lbl.key],
            )
        )
        client.labels.delete(lbl.key)
        assert client.statuses.retrieve(key=st.key).labels == []
