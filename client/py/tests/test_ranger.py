#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import synnax as sy
import numpy as np


@pytest.mark.ranger
class TestRangeClient:
    @pytest.fixture(scope="class")
    def two_ranges(self, client: sy.Synnax) -> list[sy.Range]:
        return client.ranges.create(
            [
                sy.Range(
                    name="test",
                    time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
                ),
                sy.Range(
                    name="test2",
                    time_range=sy.TimeStamp.now().span_range(30 * sy.TimeSpan.SECOND),
                ),
            ]
        )

    def test_create_single(self, client: sy.Synnax):
        """Should create a single valid range"""
        rng = client.ranges.create(
            name="test",
            time_range=sy.TimeStamp.now().span_range(10 * sy.TimeSpan.SECOND),
        )
        assert rng.name == "test"
        assert rng.key != ""

    def test_create_multiple(self, two_ranges: list[sy.Range]):
        """Should create multiple valid ranges"""
        assert len(two_ranges) == 2
        for rng in two_ranges:
            assert rng.name.startswith("test")
            assert rng.key != ""

    def test_retrieve_by_key(self, two_ranges: list[sy.Range], client: sy.Synnax):
        """Should retrieve a range by key"""
        rng = client.ranges.retrieve(two_ranges[0].key)
        assert rng.name == two_ranges[0].name
        assert rng.key == two_ranges[0].key

    def test_retrieve_by_name(self, two_ranges: list[sy.Range], client: sy.Synnax):
        """Should retrieve a range by name"""
        rng = client.ranges.retrieve([two_ranges[0].name])[0]
        assert rng.name == two_ranges[0].name

    def test_search(self, two_ranges: list[sy.Range], client: sy.Synnax):
        """Should search for ranges"""
        rng = client.ranges.search(two_ranges[0].name)
        assert len(rng) > 0

    def test_read(self, client: sy.Synnax):
        tr = sy.TimeStamp.now().span_range(100 * sy.TimeSpan.SECOND)
        stamps = np.linspace(int(tr.start), int(tr.end), 100, dtype=np.int64)
        client.channels.create(
            name="test_idx", data_type=sy.DataType.TIMESTAMP, is_index=True
        ).write(tr.start, stamps)
        rng = client.ranges.create(
            name="test",
            time_range=(tr.start + 10 * sy.TimeSpan.SECOND).span_range(
                10 * sy.TimeSpan.SECOND
            ),
        )
        res: sy.Series = rng.test_idx
        assert len(res) == 10
