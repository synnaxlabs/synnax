#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pathlib

import numpy as np
import pytest

import synnax as sy
from x.telem import seconds_linspace


@pytest.mark.framer
@pytest.mark.reader
class TestReader:
    @pytest.fixture
    def written_pair(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> tuple[sy.Channel, sy.Channel]:
        idx_ch, data_ch = indexed_pair
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND, channels=indexed_pair
        ) as w:
            w.write(
                {
                    idx_ch.key: seconds_linspace(1, 10),
                    data_ch.key: np.arange(10, dtype=np.float64),
                }
            )
            w.commit()
        return idx_ch, data_ch

    def test_read_by_key(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should key the frame by channel key"""
        idx_ch, data_ch = written_pair
        f = client.read(sy.TimeRange.MAX, [idx_ch.key, data_ch.key])
        assert np.array_equal(f[data_ch.key], np.arange(10, dtype=np.float64))
        assert np.array_equal(f[idx_ch.key], seconds_linspace(1, 10))

    def test_read_by_name(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should key the frame by channel name"""
        idx_ch, data_ch = written_pair
        f = client.read(sy.TimeRange.MAX, [idx_ch.name, data_ch.name])
        assert np.array_equal(f[data_ch.name], np.arange(10, dtype=np.float64))

    def test_read_strided(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should keep every factor-th sample under a stride"""
        _, data_ch = written_pair
        series = client.read(
            sy.TimeRange.MAX, data_ch.key, reduction=sy.StrideReduction(factor=2)
        )
        assert np.array_equal(series, np.arange(0, 10, 2, dtype=np.float64))
        assert series.series[0].alignment_multiple == 2

    def test_read_limited(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should reduce each channel to the point limit with its alignment multiple"""
        _, data_ch = written_pair
        series = client.read(
            sy.TimeRange.MAX,
            data_ch.key,
            reduction=sy.LimitReduction(aggregation="decimate", point_limit=5),
        )
        assert np.array_equal(series, np.arange(0, 10, 2, dtype=np.float64))
        assert series.series[0].alignment_multiple == 2

    def test_read_csv_strided(
        self,
        written_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
        tmp_path: pathlib.Path,
    ) -> None:
        """Should write every factor-th row under a stride"""
        _, data_ch = written_pair
        dest = tmp_path / "out.csv"
        client.read_csv(
            sy.TimeRange.MAX, data_ch.key, dest, sy.StrideReduction(factor=5)
        )
        lines = dest.read_text().splitlines()
        assert lines[1:] == ["1000000000,0", "6000000000,5"]

    def test_read_csv(
        self,
        written_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
        tmp_path: pathlib.Path,
    ) -> None:
        """Should write the channel and its index to a CSV file"""
        idx_ch, data_ch = written_pair
        dest = tmp_path / "out.csv"
        client.read_csv(sy.TimeRange.MAX, data_ch.key, dest)
        lines = dest.read_text().splitlines()
        assert lines[0] == f"{idx_ch.name},{data_ch.name}"
        assert lines[1] == "1000000000,0"
        assert len(lines) == 11

    def test_read_empty(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return no samples when the channel has no data in the range"""
        _, data_ch = indexed_pair
        assert len(client.read(sy.TimeRange.MAX, data_ch.key)) == 0
