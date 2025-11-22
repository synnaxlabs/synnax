#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import re
import time
from uuid import uuid4

import numpy as np
import pytest

import synnax as sy
from synnax.util.params import RequiresNamedParams
from synnax.util.rand import rand_name


@pytest.mark.ranger
class TestRangeClient:
    @pytest.fixture(scope="class")
    def two_ranges(self, client: sy.Synnax) -> list[sy.Range]:
        return client.ranges.create(
            [
                sy.Range(
                    name=f"test-{str(uuid4())}",
                    time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
                ),
                sy.Range(
                    name=f"test-{str(uuid4())}",
                    time_range=sy.TimeStamp.now().span_range(30 * sy.TimeSpan.SECOND),
                ),
            ]
        )

    def test_create_single(self, client: sy.Synnax):
        """Should create a single valid range"""
        rng = client.ranges.create(
            name="test",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            color="#FF0000",
        )
        assert rng.name == "test"
        assert rng.key != ""
        assert rng.color == "#FF0000"

    def test_create_multiple(self, two_ranges: list[sy.Range]):
        """Should create multiple valid ranges"""
        assert len(two_ranges) == 2
        for rng in two_ranges:
            assert rng.name.startswith("test")
            assert rng.key != ""

    def test_retrieve_by_key(self, two_ranges: list[sy.Range], client: sy.Synnax):
        """Should retrieve a range by key"""
        rng = client.ranges.retrieve(key=two_ranges[0].key)
        assert rng.name == two_ranges[0].name
        assert rng.key == two_ranges[0].key

    def test_retrieve_by_name(self, two_ranges: list[sy.Range], client: sy.Synnax):
        """Should retrieve a range by name"""
        rng = client.ranges.retrieve(name=two_ranges[0].name)
        assert rng.name == two_ranges[0].name

    def test_retrieve_unnamed_parameter(self, client: sy.Synnax):
        """Should raise an error when unnamed parameters are passed in"""
        with pytest.raises(RequiresNamedParams):
            client.ranges.retrieve("cat")

    def test_retrieve_by_name_not_found(
        self, two_ranges: list[sy.Range], client: sy.Synnax
    ):
        """Should raise an error when a range is not found"""
        with pytest.raises(sy.exceptions.QueryError):
            client.ranges.retrieve(name="not_found")

    def test_search(self, two_ranges: list[sy.Range], client: sy.Synnax):
        """Should search for ranges"""
        time.sleep(0.2)
        rng = client.ranges.search(two_ranges[0].name)
        assert len(rng) > 0

    def test_create_retrieve_if_name_exists(
        self, two_ranges: list[sy.Range], client: sy.Synnax
    ):
        """Should retrieve a range if it already exists"""
        rng = client.ranges.create(
            name=two_ranges[0].name,
            time_range=two_ranges[0].time_range,
            retrieve_if_name_exists=True,
        )
        assert rng.name == two_ranges[0].name
        assert rng.key == two_ranges[0].key

    @pytest.mark.ranger
    class TestRangeDelete:
        @pytest.fixture(scope="class")
        def rng(self, client: sy.Synnax) -> sy.Range:
            return client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )

        def test_delete_by_key(self, rng: sy.Range, client: sy.Synnax):
            """Should delete a range by key"""
            client.ranges.delete(rng.key)
            with pytest.raises(sy.exceptions.QueryError):
                client.ranges.retrieve(key=rng.key)

    @pytest.mark.ranger
    class TestRangeChannelResolution:
        @pytest.fixture(scope="class")
        def rng(self, client: sy.Synnax) -> sy.Range:
            name = f"test_{np.random.randint(0, 10000)}"
            return client.ranges.create(
                name=name,
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )

        @pytest.fixture(scope="class", autouse=True)
        def two_channels(self, client: sy.Synnax, rng: sy.Range) -> list[sy.Channel]:
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

        def test_access_by_key(self, rng: sy.Range, two_channels: list[sy.Channel]):
            """Should access a channel by key"""
            assert rng[two_channels[0].key].key == two_channels[0].key

        def test_access_by_name(self, rng: sy.Range, two_channels: list[sy.Channel]):
            """Should access a channel by name"""
            assert rng[two_channels[0].name].name == two_channels[0].name

        def test_access_by_regex(self, rng: sy.Range, two_channels: list[sy.Channel]):
            """Should access a channel by regex"""
            prefix = re.sub(r"[^A-Za-z0-9_]", "", rng.name)
            channels = rng[f"test_{prefix}_.*"]
            found = 0
            for _ in channels:
                found += 1
            assert found == 2

    @pytest.mark.ranger
    class TestRangeKV:
        def test_set_get_delete_single(self, client: sy.Synnax):
            rng = client.ranges.create(
                name=rand_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            rng.meta_data.set("test", "test")
            assert rng.meta_data.get("test") == "test"
            rng.meta_data.delete("test")
            with pytest.raises(sy.exceptions.QueryError):
                rng.meta_data.get("test")

        def test_set_get_delete_multiple(self, client: sy.Synnax):
            rng = client.ranges.create(
                name=rand_name(),
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

        def test_set_non_string_primitive(self, client: sy.Synnax):
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            rng.meta_data["key"] = 1
            assert rng.meta_data["key"] == "1"

        def test_set_non_string_non_primitive(self, client: sy.Synnax):
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            with pytest.raises(sy.ValidationError):
                rng.meta_data["key"] = object()

        def test_set_non_string_str_implemented_object(self, client: sy.Synnax):
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )

            class Test:
                def __str__(self):
                    return "test"

            rng.meta_data["key"] = Test()
            assert rng.meta_data["key"] == "test"

    @pytest.mark.ranger
    class TestRangeAlias:
        def test_basic_alias(self, client: sy.Synnax):
            ch = client.channels.create(
                name=rand_name(), data_type=sy.DataType.INT8, virtual=True
            )
            rng = client.ranges.create(
                name=rand_name(),
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            rng.set_alias(ch.key, "alt_test")
            assert rng["alt_test"].key == ch.key
            assert rng["alt_test"].name == ch.name

        def test_alias_not_found(self, client: sy.Synnax):
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            with pytest.raises(sy.exceptions.QueryError):
                rng["not_found"]

        def test_channel_not_found_on_set(self, client: sy.Synnax):
            rng = client.ranges.create(
                name="test",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            with pytest.raises(sy.exceptions.QueryError):
                rng.set_alias("not_found", "test")

    @pytest.mark.ranger
    class TestRangeChildren:
        def test_children_empty(self, client: sy.Synnax):
            """Should return empty list when range has no children"""
            rng = client.ranges.create(
                name="parent1",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )
            children = rng.children
            assert isinstance(children, list)
            assert len(children) == 0

        def test_children_with_child_ranges(self, client: sy.Synnax):
            """Should return list of child ranges"""
            parent_rng = client.ranges.create(
                name="parent2",
                time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
            )

            child1 = parent_rng.create_child_range(
                name="child1",
                time_range=sy.TimeStamp.now().span_range(5 * sy.TimeSpan.SECOND),
            )
            child2 = parent_rng.create_child_range(
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


@pytest.mark.ranger
class TestRangeData:
    def test_basic_read(self, client: sy.Synnax):
        """It should correctly read data from a channel on a range"""
        name = f"test_{random.randint(0, 100000)}"
        idx_ch = client.channels.create(
            name=f"{name}_idx",
            data_type="timestamp",
            is_index=True,
        )
        data_ch = client.channels.create(
            name=f"{name}_data",
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
            name=name,
            time_range=start.span_range(4 * sy.TimeSpan.SECOND),
        )
        assert len(rng[data_ch.name]) == 4
        assert rng[data_ch.name][0] == 1.0
        assert rng[data_ch.name][1] == 2.0
        assert rng[data_ch.name][2] == 3.0
        assert rng[data_ch.name][3] == 4.0

    def test_basic_write(self, client: sy.Synnax):
        """It should correctly write data to the range"""
        name = f"test_{random.randint(0, 100000)}"
        idx_ch = client.channels.create(
            name=f"{name}_idx",
            data_type="timestamp",
            is_index=True,
        )
        data_ch = client.channels.create(
            name=f"{name}_data",
            data_type="float32",
            index=idx_ch.key,
        )
        start = sy.TimeStamp.now()
        end = start + 3 * sy.TimeSpan.SECOND
        rng = client.ranges.create(
            name=name,
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
