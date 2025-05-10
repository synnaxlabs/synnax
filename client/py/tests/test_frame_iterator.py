#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import pytest

import synnax as sy
from tests.telem import seconds_linspace


@pytest.mark.framer
@pytest.mark.iterator
class TestIterator:
    def test_basic_iterate(self, indexed_pair: list[sy.Channel], client: sy.Synnax):
        idx_ch, _ = indexed_pair
        d = seconds_linspace(1, 50)
        idx_ch.write(sy.TimeSpan.SECOND * 1, d)
        with client.open_iterator(sy.TimeRange.MAX, idx_ch.key) as i:
            for f in i:
                assert np.array_equal(f.get(idx_ch.key), d)

    def test_auto_chunk(self, indexed_pair: sy.Channel, client: sy.Synnax):
        d = seconds_linspace(1, 10)
        idx_ch, _ = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, d)
        with client.open_iterator(sy.TimeRange.MAX, idx_ch.key, chunk_size=4) as i:
            assert i.seek_first()
            i.next(sy.framer.AUTO_SPAN)
            l = i.value.get(idx_ch.key).to_numpy().tolist()
            assert np.array_equal(l, seconds_linspace(1, 4))

            i.next(sy.framer.AUTO_SPAN)
            l = i.value.get(idx_ch.key).to_numpy().tolist()
            assert np.array_equal(l, seconds_linspace(5, 4))

            i.next(sy.framer.AUTO_SPAN)
            l = i.value.get(idx_ch.key).to_numpy().tolist()
            assert np.array_equal(l, seconds_linspace(9, 2))

            assert not i.next(sy.framer.AUTO_SPAN)

    def test_advanced_iterate(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ):
        idx_ch, data_ch = indexed_pair
        idx_ch.write(
            0,
            np.array(
                [
                    0,
                    1 * sy.TimeSpan.SECOND,
                    2 * sy.TimeSpan.SECOND,
                    3 * sy.TimeSpan.SECOND,
                    4 * sy.TimeSpan.SECOND,
                    5 * sy.TimeSpan.SECOND,
                ]
            ).astype(np.int64),
        )
        data_ch.write(0, np.array([0, 1, 2, 3, 4, 5]).astype(np.int64))
        idx_ch.write(
            sy.TimeStamp(10 * sy.TimeSpan.SECOND),
            np.array(
                [
                    10 * sy.TimeSpan.SECOND,
                    11 * sy.TimeSpan.SECOND,
                    12 * sy.TimeSpan.SECOND,
                    13 * sy.TimeSpan.SECOND,
                ]
            ).astype(np.int64),
        )
        data_ch.write(
            sy.TimeStamp(10 * sy.TimeSpan.SECOND),
            np.array([10, 11, 12, 13]).astype(np.int64),
        )
        idx_ch.write(
            sy.TimeStamp(15 * sy.TimeSpan.SECOND),
            np.array(
                [
                    15 * sy.TimeSpan.SECOND,
                    16 * sy.TimeSpan.SECOND,
                    17 * sy.TimeSpan.SECOND,
                    18 * sy.TimeSpan.SECOND,
                    19 * sy.TimeSpan.SECOND,
                ]
            ).astype(np.int64),
        )
        data_ch.write(
            sy.TimeStamp(15 * sy.TimeSpan.SECOND),
            np.array([15, 16, 17, 18, 19]).astype(np.int64),
        )
        with client.open_iterator(sy.TimeRange.MAX, data_ch.key) as i:
            assert i.seek_ge(sy.TimeStamp(16 * sy.TimeSpan.SECOND))
            assert i.next(4 * sy.TimeSpan.SECOND)
            l = i.value.get(data_ch.key).to_numpy().tolist()
            assert l == [16, 17, 18, 19]

            assert i.seek_last()
            assert i.prev(4 * sy.TimeSpan.SECOND)
            l = i.value.get(data_ch.key).to_numpy().tolist()
            assert l == [16, 17, 18, 19]

            assert i.prev(11 * sy.TimeSpan.SECOND)
            l = i.value.get(data_ch.key).to_numpy().tolist()
            assert l == [5, 10, 11, 12, 13, 15]

            assert i.seek_le(sy.TimeStamp(6 * sy.TimeSpan.SECOND))
            assert i.prev(3 * sy.TimeSpan.SECOND)
            l = i.value.get(data_ch.key).to_numpy().tolist()
            assert l == [3, 4, 5]

            assert not i.seek_le(-1)
            assert not i.seek_ge(sy.TimeStamp(20 * sy.TimeSpan.SECOND))

    def test_calculator(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ):
        idx_ch, data_ch = indexed_pair
        idx_ch.write(0, np.array([0, 1, 2, 3, 4, 5]).astype(np.int64))
        data_ch.write(0, np.array([0, 1, 2, 3, 4, 5]).astype(np.int64))

        calc = client.channels.create(
            name="calc",
            expression=f"return 2 * {data_ch.name}",
            requires=[data_ch.key],
            data_type=data_ch.data_type,
        )
        data = calc.read(sy.TimeRange.MAX)
        assert np.array_equal(data, np.array([0, 2, 4, 6, 8, 10]))
