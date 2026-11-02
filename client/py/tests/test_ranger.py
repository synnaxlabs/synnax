#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from collections.abc import Callable
from typing import cast
from uuid import UUID, uuid4

import numpy as np
import pydantic
import pytest

import synnax as sy
from synnax import label, ranger
from synnax.ontology import ID
from synnax.state import Latest, State
from tests.channel import RetrieveCounter
from x import filter
from x.params import RequiresNamedParams
from x.strings import random_name
from x.testutil import assert_eventually


class TestRangeNotCreated:
    def test_meta_data(self) -> None:
        """Should raise a ConfigurationError when reading from a range that was
        never created"""
        rng = sy.Range(
            name=random_name(),
            time_range=sy.TimeStamp.now().span_range(sy.TimeSpan.SECOND),
        )
        with pytest.raises(sy.ConfigurationError):
            rng.meta_data.get("key")


@pytest.mark.ranger
class TestRangeClient:
    @pytest.fixture(scope="class")
    @classmethod
    def two_ranges(cls, client: sy.Synnax) -> list[sy.Range]:
        return client.ranges.create(
            [
                sy.Range(
                    name=f"test-{uuid4()!s}",
                    time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
                ),
                sy.Range(
                    name=f"test-{uuid4()!s}",
                    time_range=sy.TimeStamp.now().span_range(30 * sy.TimeSpan.SECOND),
                ),
            ]
        )

    def test_create_single(self, client: sy.Synnax) -> None:
        """Should create a single valid range"""
        rng = client.ranges.create(
            name="test",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            color="#FF0000",
        )
        assert rng.name == "test"
        assert rng.key != ""
        assert rng.color == sy.Color(r=255, g=0, b=0, a=1.0)

    def test_create_retrieve_without_color(self, client: sy.Synnax) -> None:
        """Should keep a missing color as None instead of coercing it to black."""
        rng = client.ranges.create(
            sy.Range(
                name=random_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
                color=None,
            )
        )
        assert rng.color is None
        assert client.ranges.retrieve(key=rng.key).color is None

    def test_create_multiple(self, two_ranges: list[sy.Range]) -> None:
        """Should create multiple valid ranges"""
        assert len(two_ranges) == 2
        for rng in two_ranges:
            assert rng.name.startswith("test")
            assert rng.key != ""

    def test_create_reusing_resolved_parent(self, client: sy.Synnax) -> None:
        """Should accept a created range's resolved parent (a Payload) as the parent
        of another range. The server resolves parent into a Payload, and passing it
        back through create previously raised AttributeError because Range called
        to_payload() on it."""
        parent = client.ranges.create(
            name=f"parent-{uuid4()}",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
        )
        child = client.ranges.create(
            name=f"child-{uuid4()}",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            parent=parent,
        )
        assert child.parent is not None
        assert child.parent.key == parent.key
        sibling = client.ranges.create(
            name=f"sibling-{uuid4()}",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            parent=child.parent,
        )
        assert sibling.parent is not None
        assert sibling.parent.key == parent.key

    def test_retrieve_by_key(
        self, two_ranges: list[sy.Range], client: sy.Synnax
    ) -> None:
        """Should retrieve a range by key"""
        rng = client.ranges.retrieve(key=two_ranges[0].key)
        assert rng.name == two_ranges[0].name
        assert rng.key == two_ranges[0].key

    def test_retrieve_by_name(
        self, two_ranges: list[sy.Range], client: sy.Synnax
    ) -> None:
        """Should retrieve a range by name"""
        rng = client.ranges.retrieve(name=two_ranges[0].name)
        assert rng.name == two_ranges[0].name

    def test_retrieve_unnamed_parameter(self, client: sy.Synnax) -> None:
        """Should raise an error when unnamed parameters are passed in"""
        retrieve = cast(Callable[..., object], client.ranges.retrieve)
        with pytest.raises(RequiresNamedParams):
            retrieve("cat")

    def test_retrieve_by_name_not_found(
        self, two_ranges: list[sy.Range], client: sy.Synnax
    ) -> None:
        """Should raise an error when a range is not found"""
        with pytest.raises(
            sy.NotFoundError, match="Range matching not_found not found"
        ):
            client.ranges.retrieve(name="not_found")

    def test_retrieve_by_name_multiple_found(self, client: sy.Synnax) -> None:
        """Should raise an error when more than one range has the name"""
        name = random_name()
        tr = sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND)
        client.ranges.create([sy.Range(name=name, time_range=tr) for _ in range(2)])
        with pytest.raises(
            sy.MultipleFoundError,
            match=f"Expected one Range matching {name}, but found 2",
        ):
            client.ranges.retrieve(name=name)

    def test_search(self, two_ranges: list[sy.Range], client: sy.Synnax) -> None:
        """Should search for ranges"""

        def check() -> None:
            assert len(client.ranges.search(two_ranges[0].name)) > 0

        assert_eventually(check)

    def test_create_retrieve_if_name_exists(
        self, two_ranges: list[sy.Range], client: sy.Synnax
    ) -> None:
        """Should retrieve a range if it already exists"""
        rng = client.ranges.create(
            name=two_ranges[0].name,
            time_range=two_ranges[0].time_range,
            retrieve_if_name_exists=True,
        )
        assert rng.name == two_ranges[0].name
        assert rng.key == two_ranges[0].key

    def test_create_retrieve_if_name_exists_no_time_range_match(
        self, client: sy.Synnax
    ) -> None:
        """Should raise a NotFoundError when no range with the name has the same
        time range"""
        name = random_name()
        now = sy.TimeStamp.now()
        client.ranges.create(
            [
                sy.Range(name=name, time_range=now.span_range(sy.TimeSpan.SECOND)),
                sy.Range(name=name, time_range=now.span_range(2 * sy.TimeSpan.SECOND)),
            ]
        )
        with pytest.raises(sy.NotFoundError):
            client.ranges.create(
                name=name,
                time_range=now.span_range(3 * sy.TimeSpan.SECOND),
                retrieve_if_name_exists=True,
            )

    def test_create_retrieve_if_name_exists_multiple_time_range_matches(
        self, client: sy.Synnax
    ) -> None:
        """Should raise a MultipleFoundError when more than one range with the name
        has the same time range"""
        name = random_name()
        tr = sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND)
        client.ranges.create([sy.Range(name=name, time_range=tr) for _ in range(2)])
        with pytest.raises(sy.MultipleFoundError):
            client.ranges.create(name=name, time_range=tr, retrieve_if_name_exists=True)

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a range."""
        rng = client.ranges.create(
            name="Before Rename",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
        )
        client.ranges.rename(rng.key, "After Rename")
        assert client.ranges.retrieve(key=rng.key).name == "After Rename"

    def test_rename_rejects_empty_name(self, client: sy.Synnax) -> None:
        """Should reject renaming a range to an empty name."""
        rng = client.ranges.create(
            name="Before Rename",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
        )
        with pytest.raises(sy.ValidationError):
            client.ranges.rename(rng.key, "")
        assert client.ranges.retrieve(key=rng.key).name == "Before Rename"

    def test_create_rejects_empty_name(self, client: sy.Synnax) -> None:
        """Should reject creation of a range with an empty name (Pydantic
        min_length=1)
        """
        with pytest.raises(pydantic.ValidationError):
            client.ranges.create(
                name="",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )

    @pytest.mark.ranger
    class TestRangeDelete:
        @pytest.fixture(scope="class")
        @classmethod
        def rng(cls, client: sy.Synnax) -> sy.Range:
            return client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )

        def test_delete_by_key(self, rng: sy.Range, client: sy.Synnax) -> None:
            """Should delete a range by key"""
            client.ranges.delete(rng.key)
            with pytest.raises(sy.NotFoundError):
                client.ranges.retrieve(key=rng.key)

        def test_delete_multiple_keys(self, client: sy.Synnax) -> None:
            """Should delete multiple ranges in a single call"""
            ranges = client.ranges.create(
                [
                    sy.Range(
                        name=random_name(),
                        time_range=sy.TimeStamp.now().span_range(
                            10 * sy.TimeSpan.SECOND
                        ),
                    ),
                    sy.Range(
                        name=random_name(),
                        time_range=sy.TimeStamp.now().span_range(
                            5 * sy.TimeSpan.SECOND
                        ),
                    ),
                ]
            )
            keys = [r.key for r in ranges]
            client.ranges.delete(keys)
            for k in keys:
                with pytest.raises(sy.NotFoundError):
                    client.ranges.retrieve(key=k)

    @pytest.mark.ranger
    class TestRangeChannelResolution:
        @pytest.fixture(scope="class")
        @classmethod
        def rng(cls, client: sy.Synnax) -> sy.Range:
            name = f"test_{random_name()}"
            return client.ranges.create(
                name=name,
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )

        @pytest.fixture(scope="class", autouse=True)
        @classmethod
        def two_channels(cls, client: sy.Synnax, rng: sy.Range) -> list[sy.Channel]:
            prefix = re.sub(r"[^A-Za-z0-9_]", "", rng.name)
            return client.channels.create(
                [
                    sy.Channel(
                        name=f"test_{prefix}_1",
                        data_type=sy.DataType.FLOAT32,
                        virtual=True,
                    ),
                    sy.Channel(
                        name=f"test_{prefix}_2",
                        data_type=sy.DataType.FLOAT32,
                        virtual=True,
                    ),
                ]
            )

        def test_access_by_key(
            self, rng: sy.Range, two_channels: list[sy.Channel]
        ) -> None:
            """Should access a channel by key"""
            assert rng[two_channels[0].key].key == two_channels[0].key

        def test_access_by_name(
            self, rng: sy.Range, two_channels: list[sy.Channel]
        ) -> None:
            """Should access a channel by name"""
            assert rng[two_channels[0].name].name == two_channels[0].name

        def test_access_by_regex(
            self, rng: sy.Range, two_channels: list[sy.Channel]
        ) -> None:
            """Should access a channel by regex"""
            prefix = re.sub(r"[^A-Za-z0-9_]", "", rng.name)
            channels = rng[f"test_{prefix}_.*"]
            found = 0
            for _ in channels:
                found += 1
            assert found == 2

        def test_single_channel_operation_on_regex_matches(
            self, rng: sy.Range, two_channels: list[sy.Channel]
        ) -> None:
            """Should raise a MultipleFoundError when a single channel operation runs
            on a query that matches more than one channel"""
            prefix = re.sub(r"[^A-Za-z0-9_]", "", rng.name)
            with pytest.raises(sy.MultipleFoundError):
                rng[f"test_{prefix}_.*"].to_numpy()

        def test_repeated_access_retrieves_once(
            self,
            counted_client: tuple[sy.Synnax, RetrieveCounter],
            rng: sy.Range,
            two_channels: list[sy.Channel],
        ) -> None:
            """Should retrieve a channel once when accessed many times"""
            client, counter = counted_client
            r = client.ranges.retrieve(key=rng.key)
            counter.count = 0
            for _ in range(100):
                assert r[two_channels[0].name].key == two_channels[0].key
                assert r[two_channels[1].key].key == two_channels[1].key
            assert counter.count == 2

    @pytest.mark.ranger
    class TestRangeKV:
        def test_set_get_delete_single(self, client: sy.Synnax) -> None:
            rng = client.ranges.create(
                name=random_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            rng.meta_data.set("test", "test")
            assert rng.meta_data.get("test") == "test"
            rng.meta_data.delete("test")
            with pytest.raises(sy.exceptions.QueryError):
                rng.meta_data.get("test")

        def test_set_get_delete_multiple(self, client: sy.Synnax) -> None:
            rng = client.ranges.create(
                name=random_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            rng.meta_data.set({"test": "test", "test2": "test2"})
            assert rng.meta_data.get(["test", "test2"]) == {
                "test": "test",
                "test2": "test2",
            }
            rng.meta_data.delete(["test", "test2"])
            with pytest.raises(sy.exceptions.QueryError):
                rng.meta_data.get(["test", "test2"])

        def test_set_non_string_primitive(self, client: sy.Synnax) -> None:
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            rng.meta_data["key"] = 1
            assert rng.meta_data["key"] == "1"

        def test_set_non_string_non_primitive(self, client: sy.Synnax) -> None:
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            with pytest.raises(sy.ValidationError):
                rng.meta_data["key"] = object()

        def test_set_non_string_str_implemented_object(self, client: sy.Synnax) -> None:
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )

            class Test:
                def __str__(self) -> str:
                    return "test"

            rng.meta_data["key"] = Test()
            assert rng.meta_data["key"] == "test"

    @pytest.mark.ranger
    class TestRangeAlias:
        def test_basic_alias(self, client: sy.Synnax) -> None:
            ch = client.channels.create(
                name=random_name(), data_type=sy.DataType.INT8, virtual=True
            )
            rng = client.ranges.create(
                name=random_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            rng.set_alias(ch.key, "alt_test")
            assert rng["alt_test"].key == ch.key
            assert rng["alt_test"].name == ch.name

        def test_alias_not_found(self, client: sy.Synnax) -> None:
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            with pytest.raises(
                sy.NotFoundError, match="Channel matching not_found not found"
            ):
                rng["not_found"]

        def test_channel_not_found_on_set(self, client: sy.Synnax) -> None:
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            with pytest.raises(
                sy.NotFoundError, match="Channel matching not_found not found"
            ):
                rng.set_alias("not_found", "test")

        def test_scoped_channel_multiple_found(self, client: sy.Synnax) -> None:
            """Should raise MultipleFoundError when a scoped channel matches two
            channels and is used as one."""
            prefix = random_name()
            for suffix in ("a", "b"):
                client.channels.create(
                    name=f"{prefix}_{suffix}", data_type=sy.DataType.INT8, virtual=True
                )
            rng = client.ranges.create(
                name=random_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            with pytest.raises(sy.MultipleFoundError, match="Multiple channels found"):
                rng[f"{prefix}_.*"].to_numpy()

        def test_set_alias_multiple_found(self, client: sy.Synnax) -> None:
            """Should raise MultipleFoundError when a name matches two channels."""
            prefix = random_name()
            for suffix in ("a", "b"):
                client.channels.create(
                    name=f"{prefix}_{suffix}", data_type=sy.DataType.INT8, virtual=True
                )
            rng = client.ranges.create(
                name=random_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            with pytest.raises(
                sy.MultipleFoundError, match=f"Expected one Channel matching {prefix}_"
            ):
                rng.set_alias(f"{prefix}_.*", "test")

        def test_resolve_missing_alias(self, client: sy.Synnax) -> None:
            """Should raise NotFoundError when a single alias has no channel."""
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            aliases = ranger.alias.Client(rng=rng.key, client=client._transport.unary)
            with pytest.raises(
                sy.NotFoundError, match="Alias matching not_found not found"
            ):
                aliases.resolve("not_found")

    @pytest.mark.ranger
    class TestRangeChildren:
        def test_children_empty(self, client: sy.Synnax) -> None:
            """Should return empty list when range has no children"""
            rng = client.ranges.create(
                name="parent1",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            children = rng.children
            assert isinstance(children, list)
            assert len(children) == 0

        def test_children_with_child_ranges(self, client: sy.Synnax) -> None:
            """Should return list of child ranges"""
            parent_rng = client.ranges.create(
                name="parent2",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )

            parent_rng.create_child_range(
                name="child1",
                time_range=sy.TimeStamp.now().span_range(5 * sy.TimeSpan.SECOND),
            )
            parent_rng.create_child_range(
                name="child2",
                time_range=sy.TimeStamp.now().span_range(3 * sy.TimeSpan.SECOND),
            )

            children = parent_rng.children
            assert isinstance(children, list)
            assert len(children) == 2

            for child in children:
                assert isinstance(child, sy.Range)

            child_names = {child.name for child in children}
            assert child_names == {"child1", "child2"}

        def test_create_kwargs_with_parent_attaches(self, client: sy.Synnax) -> None:
            """Client.create(parent=range) should attach the child under the parent"""
            parent_rng = client.ranges.create(
                name=random_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            child_name = random_name()
            child = client.ranges.create(
                name=child_name,
                time_range=sy.TimeStamp.now().span_range(5 * sy.TimeSpan.SECOND),
                parent=parent_rng,
            )
            assert child.parent is not None
            assert child.parent.key == parent_rng.key
            children = parent_rng.children
            assert child_name in {c.name for c in children}

        def test_create_single_range_with_parent_attaches(
            self, client: sy.Synnax
        ) -> None:
            """Passing a Range with .parent set should attach it under the parent"""
            parent_rng = client.ranges.create(
                name=random_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            child_name = random_name()
            child = client.ranges.create(
                sy.Range(
                    name=child_name,
                    time_range=sy.TimeStamp.now().span_range(5 * sy.TimeSpan.SECOND),
                    parent=parent_rng,
                )
            )
            assert child.parent is not None
            assert child.parent.key == parent_rng.key
            children = parent_rng.children
            assert child_name in {c.name for c in children}

        def test_create_list_with_parent_attaches(self, client: sy.Synnax) -> None:
            """List-form create should preserve each Range's parent assignment"""
            parent_rng = client.ranges.create(
                name=random_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            names = [random_name(), random_name()]
            created = client.ranges.create(
                [
                    sy.Range(
                        name=names[0],
                        time_range=sy.TimeStamp.now().span_range(
                            5 * sy.TimeSpan.SECOND
                        ),
                        parent=parent_rng,
                    ),
                    sy.Range(
                        name=names[1],
                        time_range=sy.TimeStamp.now().span_range(
                            3 * sy.TimeSpan.SECOND
                        ),
                        parent=parent_rng,
                    ),
                ]
            )
            for c in created:
                assert c.parent is not None
                assert c.parent.key == parent_rng.key
            children = parent_rng.children
            child_names = {c.name for c in children}
            assert names[0] in child_names
            assert names[1] in child_names


class _LabelCreateRequest(pydantic.BaseModel):
    labels: list[label.Label]


class _LabelSetRequest(pydantic.BaseModel):
    id: ID
    labels: list[label.Key]


class _Empty(pydantic.BaseModel):
    pass


def _create_label(client: sy.Synnax, rng: sy.Range) -> label.Label:
    lbl = label.Label(name=random_name(), color=sy.Color(r=255, g=0, b=0, a=1.0))
    unary = client._transport.unary
    unary.send("/label/create", _LabelCreateRequest(labels=[lbl]), _Empty)
    req = _LabelSetRequest(id=rng.ontology_id, labels=[lbl.key])
    unary.send("/label/set", req, _Empty)
    return lbl


@pytest.fixture(scope="class")
def window() -> sy.TimeRange:
    start = sy.TimeStamp(int(uuid4().int % 10**9) * sy.TimeSpan.SECOND)
    return start.span_range(100 * sy.TimeSpan.SECOND)


@pytest.fixture(scope="class")
def window_ranges(client: sy.Synnax, window: sy.TimeRange) -> list[sy.Range]:
    start = window.start
    return client.ranges.create(
        [
            sy.Range(
                name=random_name(),
                time_range=(start + i * 40 * sy.TimeSpan.SECOND).span_range(
                    10 * sy.TimeSpan.SECOND
                ),
            )
            for i in range(3)
        ]
    )


@pytest.mark.ranger
class TestRangeRetrieveFilters:
    """Tests the filters accepted by the range client's retrieve method."""

    def test_rejects_mixed_selectors(self, client: sy.Synnax) -> None:
        """Should reject a key or name combined with another selector or a list
        filter
        """
        retrieve = cast(Callable[..., object], client.ranges.retrieve)
        key = uuid4()
        with pytest.raises(ValueError, match="cannot be combined; use keys"):
            retrieve(key=key, name="a")
        with pytest.raises(ValueError, match="cannot be combined with list filters"):
            retrieve(key=key, limit=1)
        with pytest.raises(ValueError, match="cannot be combined with list filters"):
            retrieve(name="a", keys=[key])

    def test_where(
        self,
        client: sy.Synnax,
        window: sy.TimeRange,
        window_ranges: list[sy.Range],
    ) -> None:
        """Should keep ranges that match a filter tree and the other filters"""
        where = ranger.Where(
            not_=ranger.Where(name=sy.filter.String(eq=window_ranges[0].name))
        )
        res = client.ranges.retrieve(where=where, overlaps_with=window)
        assert {r.key for r in res} == {r.key for r in window_ranges[1:]}

    def test_empty_keys(self, client: sy.Synnax, window_ranges: list[sy.Range]) -> None:
        """Should retrieve no ranges for an empty list of keys"""
        assert client.ranges.retrieve(keys=[]) == []

    def test_overlaps_with(
        self,
        client: sy.Synnax,
        window: sy.TimeRange,
        window_ranges: list[sy.Range],
    ) -> None:
        """Should keep only ranges that overlap the time range"""
        tr = window.start.span_range(50 * sy.TimeSpan.SECOND)
        res = client.ranges.retrieve(overlaps_with=tr)
        assert {r.key for r in res} == {r.key for r in window_ranges[:2]}

    def test_limit_offset(
        self,
        client: sy.Synnax,
        window: sy.TimeRange,
        window_ranges: list[sy.Range],
    ) -> None:
        """Should page through the matching ranges"""
        first = client.ranges.retrieve(overlaps_with=window, limit=2)
        rest = client.ranges.retrieve(overlaps_with=window, limit=2, offset=2)
        assert len(first) == 2
        assert len(rest) == 1
        keys = {r.key for r in first} | {r.key for r in rest}
        assert keys == {r.key for r in window_ranges}

    def test_search_term(
        self, client: sy.Synnax, window_ranges: list[sy.Range]
    ) -> None:
        """Should retrieve ranges whose names match the search term"""
        name = window_ranges[0].name

        def check() -> None:
            res = client.ranges.retrieve(search_term=name)
            assert window_ranges[0].key in {r.key for r in res}

        assert_eventually(check)

    def test_has_labels(self, client: sy.Synnax, window_ranges: list[sy.Range]) -> None:
        """Should keep only ranges with the label and attach labels on request"""
        rng = window_ranges[1]
        lbl = _create_label(client, rng)
        res = client.ranges.retrieve(has_labels=[lbl.key])
        assert [r.key for r in res] == [rng.key]
        assert res[0].labels is None
        res = client.ranges.retrieve(has_labels=[lbl.key], include_labels=True)
        assert res[0].labels is not None
        assert [lb.key for lb in res[0].labels] == [lbl.key]
        single = client.ranges.retrieve(key=rng.key, include_labels=True)
        assert single.labels is not None
        assert [lb.name for lb in single.labels] == [lbl.name]

    def test_include_parent(
        self, client: sy.Synnax, window_ranges: list[sy.Range]
    ) -> None:
        """Should attach the parent of each range on request"""
        parent = window_ranges[2]
        child = parent.create_child_range(
            name=random_name(), time_range=parent.time_range
        )
        assert client.ranges.retrieve(key=child.key).parent is None
        res = client.ranges.retrieve(key=child.key, include_parent=True)
        assert res.parent is not None
        assert res.parent.key == parent.key
        many = client.ranges.retrieve(keys=[child.key], include_parent=True)
        assert many[0].parent is not None
        assert many[0].parent.key == parent.key

    def test_include_labels_only(
        self, client: sy.Synnax, window_ranges: list[sy.Range]
    ) -> None:
        """Should return a list when given include_labels without a key or name"""
        res = client.ranges.retrieve(include_labels=True)
        assert {r.key for r in window_ranges} <= {r.key for r in res}

    def test_include_parent_only(
        self, client: sy.Synnax, window_ranges: list[sy.Range]
    ) -> None:
        """Should return a list when given include_parent without a key or name"""
        res = client.ranges.retrieve(include_parent=True)
        assert {r.key for r in window_ranges} <= {r.key for r in res}


@pytest.mark.ranger
class TestRangeConstructor:
    """Pure-Pydantic tests of Range construction — no server required."""

    def test_accepts_range_as_parent(self) -> None:
        """Range(parent=other_range) should store the parent's identity"""
        parent = sy.Range(
            name="parent",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
        )
        child = sy.Range(
            name="child",
            time_range=sy.TimeStamp.now().span_range(5 * sy.TimeSpan.SECOND),
            parent=parent,
        )
        assert child.parent is not None
        assert child.parent.key == parent.key
        assert child.parent.name == parent.name

    def test_strips_grandparent(self) -> None:
        """A parent that itself has a parent should be flattened to depth 1"""
        grandparent = sy.Range(
            name="grandparent",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
        )
        parent = sy.Range(
            name="parent",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            parent=grandparent,
        )
        child = sy.Range(
            name="child",
            time_range=sy.TimeStamp.now().span_range(5 * sy.TimeSpan.SECOND),
            parent=parent,
        )
        assert child.parent is not None
        assert child.parent.key == parent.key
        assert child.parent.parent is None

    def test_no_parent(self) -> None:
        """Range without a parent should have parent=None"""
        rng = sy.Range(
            name="solo",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
        )
        assert rng.parent is None


@pytest.mark.ranger
class TestRangeData:
    def test_basic_read(self, client: sy.Synnax) -> None:
        """It should correctly read data from a channel on a range"""
        idx_ch = client.channels.create(
            name=random_name(),
            data_type="timestamp",
            is_index=True,
        )
        data_ch = client.channels.create(
            name=random_name(),
            data_type="float32",
            index=idx_ch.key,
        )
        start = sy.TimeStamp.now()
        end = start + 3 * sy.TimeSpan.SECOND
        idx_ch.write(
            start,
            [
                start,
                start + 1 * sy.TimeSpan.SECOND,
                start + 2 * sy.TimeSpan.SECOND,
                end,
            ],
        )
        data_ch.write(start, [1.0, 2.0, 3.0, 4.0])
        rng = client.ranges.create(
            name=random_name(),
            time_range=start.span_range(4 * sy.TimeSpan.SECOND),
        )
        assert len(rng[data_ch.name]) == 4
        assert rng[data_ch.name][0] == 1.0
        assert rng[data_ch.name][1] == 2.0
        assert rng[data_ch.name][2] == 3.0
        assert rng[data_ch.name][3] == 4.0

    def test_array_operators(self, client: sy.Synnax) -> None:
        """It should apply NumPy operators to the channel's data on the range"""
        idx_ch = client.channels.create(
            name=random_name(),
            data_type="timestamp",
            is_index=True,
        )
        data_ch = client.channels.create(
            name=random_name(),
            data_type="float32",
            index=idx_ch.key,
        )
        start = sy.TimeStamp.now()
        idx_ch.write(start, [start + i * sy.TimeSpan.SECOND for i in range(3)])
        data_ch.write(start, [1.0, 2.0, 3.0])
        rng = client.ranges.create(
            name=random_name(),
            time_range=start.span_range(3 * sy.TimeSpan.SECOND),
        )
        scoped = rng[data_ch.name]
        assert np.array_equal(scoped > 1, [False, True, True])
        assert np.array_equal(scoped * 2, [2.0, 4.0, 6.0])
        assert np.array_equal(10 - scoped, [9.0, 8.0, 7.0])
        assert np.array_equal(-scoped, [-1.0, -2.0, -3.0])
        internal = next(iter(scoped))
        assert np.array_equal(internal == 2, [False, True, False])
        assert hash(internal) == hash(data_ch.key)
        with pytest.raises(TypeError):
            hash(scoped)

    def test_basic_write(self, client: sy.Synnax) -> None:
        """It should correctly write data to the range"""
        idx_ch = client.channels.create(
            name=random_name(),
            data_type="timestamp",
            is_index=True,
        )
        data_ch = client.channels.create(
            name=random_name(),
            data_type="float32",
            index=idx_ch.key,
        )
        start = sy.TimeStamp.now()
        end = start + 3 * sy.TimeSpan.SECOND
        rng = client.ranges.create(
            name=random_name(),
            time_range=start.span_range(4 * sy.TimeSpan.SECOND),
        )
        rng.write(
            {
                data_ch.name: [1.0, 2.0, 3.0, 4.0],
                idx_ch.name: [
                    start,
                    start + 1 * sy.TimeSpan.SECOND,
                    start + 2 * sy.TimeSpan.SECOND,
                    end,
                ],
            }
        )


@pytest.mark.ranger
class TestOnCreate:
    """Tests for the handler that on_create registers on the range set channel."""

    def latest_state(self, client: sy.Synnax, sample: sy.Series) -> Latest:
        ch = client.channels.retrieve("sy_range_set")
        state = State([ch])
        state.update(sy.Frame([ch.key], [sample]))
        return Latest(state)

    def test_valid_sample(self, client: sy.Synnax) -> None:
        """Should pass the range in a JSON object sample to the callback"""
        received: list[sy.Range] = []
        handler = client.ranges.on_create(received.append)
        key = uuid4()
        start = sy.TimeStamp.now()
        sample = {
            "key": str(key),
            "name": "on_create",
            "time_range": {
                "start": int(start),
                "end": int(start + sy.TimeSpan.NANOSECOND),
            },
        }
        handler(self.latest_state(client, sy.Series([sample], data_type="json")))
        assert len(received) == 1
        assert received[0].key == key
        assert received[0].name == "on_create"
        assert received[0].time_range == sy.TimeRange(
            start, start + sy.TimeSpan.NANOSECOND
        )

    def test_non_object_sample(self, client: sy.Synnax) -> None:
        """Should raise a ValidationError when the sample is not a range"""
        handler = client.ranges.on_create(lambda _: None)
        with pytest.raises(pydantic.ValidationError):
            handler(
                self.latest_state(
                    client, sy.Series(["not an object"], data_type="json")
                )
            )


class TestBuilder:
    def test_join(self) -> None:
        """Should join nested range conditions into one Where tree."""
        key = uuid4()
        label_key = uuid4()
        window = sy.TimeRange(sy.TimeStamp(1), sy.TimeStamp(2))
        where = filter.join(
            ranger.Where,
            [
                ranger.name.starts_with("hot", case_insensitive=True)
                | ranger.labels.has(label_key),
                ~ranger.key.in_([key]),
                ranger.overlaps(window),
            ],
        )
        assert where == ranger.Where(
            and_=[
                ranger.Where(
                    or_=[
                        ranger.Where(
                            name=filter.String(starts_with="hot", case_insensitive=True)
                        ),
                        ranger.Where(labels=[label_key]),
                    ]
                ),
                ranger.Where(not_=ranger.Where(key=filter.Set[UUID](in_=[key]))),
                ranger.Where(overlaps=window),
            ]
        )

    def test_another_entity(self) -> None:
        """Should fail the type check for a condition of another entity."""
        condition: label.Condition = ranger.name.eq("Hotfire")  # type: ignore[assignment]
        assert condition.tree == {"name": {"eq": "Hotfire"}}
