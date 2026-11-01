#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
import json
import pathlib
import uuid

import numpy as np
import pytest
from nptdms import TdmsChannel, TdmsFile, TdmsGroup, types

import synnax as sy
from x.strings import random_name
from x.telem import seconds_linspace

_TDMS_EPOCH_OFFSET = 2082844800

_TDMS_TYPES: dict[sy.DataType, type[types.TdmsType]] = {
    sy.DataType.INT8: types.Int8,
    sy.DataType.INT16: types.Int16,
    sy.DataType.INT32: types.Int32,
    sy.DataType.INT64: types.Int64,
    sy.DataType.UINT8: types.Uint8,
    sy.DataType.UINT16: types.Uint16,
    sy.DataType.UINT32: types.Uint32,
    sy.DataType.UINT64: types.Uint64,
    sy.DataType.FLOAT32: types.SingleFloat,
    sy.DataType.FLOAT64: types.DoubleFloat,
    sy.DataType.BOOLEAN: types.Boolean,
    sy.DataType.TIMESTAMP: types.TimeStamp,
    sy.DataType.STRING: types.String,
    sy.DataType.JSON: types.String,
    sy.DataType.UUID: types.String,
    sy.DataType.BYTES: types.String,
}


def _create_index(client: sy.Synnax) -> sy.Channel:
    return client.channels.create(
        name=f"test_{random_name()}_time",
        is_index=True,
        data_type=sy.DataType.TIMESTAMP,
    )


def _read_tdms(
    client: sy.Synnax,
    channels: list[sy.Channel],
    dest: pathlib.Path,
    reduction: sy.StrideReduction | None = None,
) -> TdmsFile:
    client.read_tdms(
        sy.TimeRange.MAX, [ch.key for ch in channels], dest=dest, reduction=reduction
    )
    return TdmsFile.read(dest, raw_timestamps=True)


def _tdms_timestamps(ch: TdmsChannel) -> list[int]:
    """Decodes a timestamp channel to nanoseconds since the Unix epoch, rounding each
    fraction down."""
    data = ch[:]
    return [
        (int(s) - _TDMS_EPOCH_OFFSET) * 10**9 + (int(f) * 10**9 >> 64)
        for s, f in zip(data["seconds"], data["second_fractions"], strict=True)
    ]


def _tdms_group(file: TdmsFile, ch: sy.Channel) -> TdmsGroup:
    [group] = [g for g in file.groups() if g.name == ch.name]
    return group


def _tdms_segment_count(content: bytes) -> int:
    """Counts the segments of a TDMS file, checking that each lead-in points at the
    next and that the last ends the file."""
    pos, count = 0, 0
    while pos < len(content):
        assert content[pos : pos + 4] == b"TDSm"
        pos += 28 + int.from_bytes(content[pos + 12 : pos + 20], "little")
        count += 1
    assert pos == len(content)
    return count


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
                    idx_ch.key: seconds_linspace(1, count=10),
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
        assert np.array_equal(f[idx_ch.key], seconds_linspace(1, count=10))

    def test_read_by_key_get_by_name(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should find a channel by name in a frame read by key"""
        idx_ch, data_ch = written_pair
        f = client.read(sy.TimeRange.MAX, [idx_ch.key, data_ch.key])
        assert data_ch.name in f
        assert np.array_equal(f[data_ch.name], np.arange(10, dtype=np.float64))

    @pytest.mark.parametrize("by_name", [True, False])
    def test_read_to_df(
        self,
        written_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
        by_name: bool,
    ) -> None:
        """Should convert the frame to a DataFrame with channel name columns in the
        requested order"""
        idx_ch, data_ch = written_pair
        requested = [data_ch, idx_ch]
        if by_name:
            fr = client.read(sy.TimeRange.MAX, [ch.name for ch in requested])
        else:
            fr = client.read(sy.TimeRange.MAX, [ch.key for ch in requested])
        df = fr.to_df()
        assert list(df.columns) == [data_ch.name, idx_ch.name]
        assert np.array_equal(df[data_ch.name], np.arange(10, dtype=np.float64))

    def test_read_by_name(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should key the frame by channel name"""
        idx_ch, data_ch = written_pair
        f = client.read(sy.TimeRange.MAX, [idx_ch.name, data_ch.name])
        assert np.array_equal(f[data_ch.name], np.arange(10, dtype=np.float64))

    def test_read_single_key(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return a MultiSeries for a single key"""
        _, data_ch = written_pair
        series = client.read(sy.TimeRange.MAX, data_ch.key)
        assert isinstance(series, sy.MultiSeries)
        assert np.array_equal(series, np.arange(10, dtype=np.float64))

    def test_read_single_name(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return a MultiSeries for a single name"""
        _, data_ch = written_pair
        series = client.read(sy.TimeRange.MAX, data_ch.name)
        assert isinstance(series, sy.MultiSeries)
        assert np.array_equal(series, np.arange(10, dtype=np.float64))

    def test_read_one_key_list(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return a Frame for a list with one key"""
        _, data_ch = written_pair
        f = client.read(sy.TimeRange.MAX, [data_ch.key])
        assert isinstance(f, sy.Frame)
        assert f.channels == [data_ch.key]
        assert np.array_equal(f[data_ch.key], np.arange(10, dtype=np.float64))

    def test_read_one_name_tuple(
        self, written_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return a Frame for a tuple with one name"""
        _, data_ch = written_pair
        f = client.read(sy.TimeRange.MAX, (data_ch.name,))
        assert isinstance(f, sy.Frame)
        assert f.channels == [data_ch.name]
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
            sy.TimeRange.MAX,
            data_ch.key,
            dest=dest,
            reduction=sy.StrideReduction(factor=5),
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
        client.read_csv(sy.TimeRange.MAX, data_ch.key, dest=dest)
        lines = dest.read_text().splitlines()
        assert lines[0] == f"{idx_ch.name},{data_ch.name}"
        assert lines[1] == "1000000000,0"
        assert len(lines) == 11

    def test_read_tdms(self, client: sy.Synnax, tmp_path: pathlib.Path) -> None:
        """Should write every data type so that a TDMS reader recovers each sample"""
        index = _create_index(client)
        start = 1_700_000_000_123_456_789
        timestamps = [
            start,
            start + 1,
            start + 999_999_999,
            start + 10**9,
            start + 10**9 + 1,
        ]
        numeric: dict[sy.DataType, np.ndarray] = {}
        for it in (
            np.int8,
            np.int16,
            np.int32,
            np.int64,
            np.uint8,
            np.uint16,
            np.uint32,
            np.uint64,
        ):
            info = np.iinfo(it)
            numeric[sy.DataType(np.dtype(it))] = np.array(
                [info.min, info.min + 1, info.max // 2, info.max - 1, info.max],
                dtype=it,
            )
        for ft in (np.float32, np.float64):
            finfo = np.finfo(ft)
            numeric[sy.DataType(np.dtype(ft))] = np.array(
                [-np.inf, finfo.min, finfo.tiny, finfo.max, np.nan], dtype=ft
            )
        numeric[sy.DataType.BOOLEAN] = np.array([True, False, False, True, True])
        uuids = [
            uuid.UUID(int=0),
            uuid.UUID(int=2**128 - 1),
            *(uuid.uuid4() for _ in range(3)),
        ]
        docs: list[dict[str, object]] = [
            {},
            {"a": 1},
            {"b": [1, 2]},
            {"c": "it's"},
            {"d": None},
        ]
        payloads = [b"", b"\x00", b"\xff\xfe", b"synnax", bytes(range(256))]
        strings = ["", "a", "héllo wörld", "it's", "x" * 1000]
        text: dict[sy.DataType, tuple[sy.Series, list[str]]] = {
            sy.DataType.STRING: (
                sy.Series(strings, data_type=sy.DataType.STRING),
                strings,
            ),
            sy.DataType.JSON: (
                sy.Series(docs, data_type=sy.DataType.JSON),
                [json.dumps(d) for d in docs],
            ),
            sy.DataType.UUID: (
                sy.Series(uuids, data_type=sy.DataType.UUID),
                [str(u) for u in uuids],
            ),
            sy.DataType.BYTES: (
                sy.Series(
                    b"".join(len(p).to_bytes(4, "little") + p for p in payloads),
                    data_type=sy.DataType.BYTES,
                ),
                [base64.b64encode(p).decode() for p in payloads],
            ),
        }
        stamps = [0, 1, -1, 999_999_999, start]
        data_types = [*numeric, *text, sy.DataType.TIMESTAMP]
        channels = client.channels.create(
            [
                sy.Channel(
                    name=f"test_{random_name()}_{dt}", data_type=dt, index=index.key
                )
                for dt in data_types
            ]
        )
        by_type = {ch.data_type: ch for ch in channels}
        with client.open_writer(
            start=start, channels=[index.key, *(ch.key for ch in channels)]
        ) as w:
            w.write(
                {
                    index.key: sy.Series(timestamps, data_type=sy.DataType.TIMESTAMP),
                    **{
                        by_type[dt].key: sy.Series(v, data_type=dt)
                        for dt, v in numeric.items()
                    },
                    **{by_type[dt].key: v for dt, (v, _) in text.items()},
                    by_type[sy.DataType.TIMESTAMP].key: sy.Series(
                        stamps, data_type=sy.DataType.TIMESTAMP
                    ),
                }
            )
            w.commit()

        tdms = _read_tdms(client, channels, tmp_path / "out.tdms")

        assert tdms.tdms_version == 4713
        floats = {sy.DataType.FLOAT32, sy.DataType.FLOAT64}
        own = [ch for ch in channels if ch.data_type not in floats]
        assert tdms.groups()[0].name == index.name
        assert {g.name for g in tdms.groups()} == {index.name, *(c.name for c in own)}
        [tdms_index, *tdms_channels] = _tdms_group(tdms, index).channels()
        assert tdms_index.name == index.name
        assert tdms_index.data_type is types.TimeStamp
        assert _tdms_timestamps(tdms_index) == timestamps
        by_name = {c.name: c for c in tdms_channels}
        assert by_name.keys() == {ch.name for ch in channels if ch.data_type in floats}
        for ch in own:
            tdms_time, by_name[ch.name] = _tdms_group(tdms, ch).channels()
            assert tdms_time.name == index.name
            assert _tdms_timestamps(tdms_time) == timestamps
        for dt, ch in by_type.items():
            assert by_name[ch.name].data_type is _TDMS_TYPES[dt]
        for dt, values in numeric.items():
            read = by_name[by_type[dt].name][:]
            assert read.dtype == values.dtype
            np.testing.assert_array_equal(read, values)
        for dt, (_, expected) in text.items():
            assert by_name[by_type[dt].name][:].tolist() == expected
        tdms_stamps = by_name[by_type[sy.DataType.TIMESTAMP].name]
        assert _tdms_timestamps(tdms_stamps) == stamps

    def test_read_tdms_groups(self, client: sy.Synnax, tmp_path: pathlib.Path) -> None:
        """Should write the channels of each index to their own group"""
        fast_index, slow_index = _create_index(client), _create_index(client)
        fast, slow = client.channels.create(
            [
                sy.Channel(
                    name=f"test_{random_name()}_fast",
                    data_type=sy.DataType.FLOAT64,
                    index=fast_index.key,
                ),
                sy.Channel(
                    name=f"test_{random_name()}_slow",
                    data_type=sy.DataType.INT32,
                    index=slow_index.key,
                ),
            ]
        )
        fast_times, slow_times = (
            seconds_linspace(1, count=10),
            seconds_linspace(2, count=3),
        )
        for index, ch, times, values in (
            (fast_index, fast, fast_times, np.arange(10, dtype=np.float64)),
            (slow_index, slow, slow_times, np.array([-1, 0, 1], dtype=np.int32)),
        ):
            with client.open_writer(start=times[0], channels=[index.key, ch.key]) as w:
                w.write({index.key: times, ch.key: values})
                w.commit()

        tdms = _read_tdms(client, [fast, slow], tmp_path / "out.tdms")

        assert {g.name for g in tdms.groups()} == {
            fast_index.name,
            slow_index.name,
            slow.name,
        }
        tdms_index, tdms_fast = _tdms_group(tdms, fast_index).channels()
        assert (tdms_index.name, tdms_fast.name) == (fast_index.name, fast.name)
        assert _tdms_timestamps(tdms_index) == [int(t) for t in fast_times]
        np.testing.assert_array_equal(tdms_fast[:], np.arange(10, dtype=np.float64))
        [tdms_index] = _tdms_group(tdms, slow_index).channels()
        assert _tdms_timestamps(tdms_index) == [int(t) for t in slow_times]
        tdms_time, tdms_slow = _tdms_group(tdms, slow).channels()
        assert (tdms_time.name, tdms_slow.name) == (slow_index.name, slow.name)
        assert _tdms_timestamps(tdms_time) == [int(t) for t in slow_times]
        np.testing.assert_array_equal(
            tdms_slow[:], np.array([-1, 0, 1], dtype=np.int32)
        )

    def test_read_tdms_late_channels(
        self, client: sy.Synnax, tmp_path: pathlib.Path
    ) -> None:
        """Should pair the samples of channels that start late with their
        timestamps"""
        index = _create_index(client)
        full, late_float, late_int = client.channels.create(
            [
                sy.Channel(
                    name=f"test_{random_name()}_{dt}", data_type=dt, index=index.key
                )
                for dt in (sy.DataType.FLOAT64, sy.DataType.FLOAT32, sy.DataType.INT32)
            ]
        )
        times = seconds_linspace(1, count=6)
        with client.open_writer(start=times[0], channels=[index.key, full.key]) as w:
            w.write({index.key: times[:3], full.key: np.arange(3, dtype=np.float64)})
            w.commit()
        with client.open_writer(
            start=times[3],
            channels=[index.key, full.key, late_float.key, late_int.key],
        ) as w:
            w.write(
                {
                    index.key: times[3:],
                    full.key: np.arange(3, 6, dtype=np.float64),
                    late_float.key: np.array([3, 4, 5], dtype=np.float32),
                    late_int.key: np.array([-3, -4, -5], dtype=np.int32),
                }
            )
            w.commit()

        tdms = _read_tdms(client, [full, late_float, late_int], tmp_path / "out.tdms")

        tdms_index, *tdms_floats = _tdms_group(tdms, index).channels()
        assert _tdms_timestamps(tdms_index) == [int(t) for t in times]
        by_name = {c.name: c for c in tdms_floats}
        np.testing.assert_array_equal(
            by_name[full.name][:], np.arange(6, dtype=np.float64)
        )
        np.testing.assert_array_equal(
            by_name[late_float.name][:],
            np.array([np.nan, np.nan, np.nan, 3, 4, 5], dtype=np.float32),
        )
        tdms_time, tdms_int = _tdms_group(tdms, late_int).channels()
        assert _tdms_timestamps(tdms_time) == [int(t) for t in times[3:]]
        np.testing.assert_array_equal(
            tdms_int[:], np.array([-3, -4, -5], dtype=np.int32)
        )

    def test_read_tdms_segments(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
        tmp_path: pathlib.Path,
    ) -> None:
        """Should write a large read as a chain of segments"""
        index, data = indexed_pair
        timestamps = np.arange(150_000, dtype=np.int64) * 1_000_007 + 10**18
        values = np.random.default_rng(0).standard_normal(150_000)
        for part in np.array_split(np.arange(150_000), 3):
            with client.open_writer(
                start=int(timestamps[part[0]]), channels=[index.key, data.key]
            ) as w:
                w.write(
                    {
                        index.key: sy.Series(
                            timestamps[part], data_type=sy.DataType.TIMESTAMP
                        ),
                        data.key: values[part],
                    }
                )
                w.commit()
        dest = tmp_path / "out.tdms"

        tdms = _read_tdms(client, [data], dest)

        assert _tdms_segment_count(dest.read_bytes()) > 2
        tdms_index, tdms_data = _tdms_group(tdms, index).channels()
        assert _tdms_timestamps(tdms_index) == timestamps.tolist()
        np.testing.assert_array_equal(tdms_data[:], values)

    def test_read_tdms_downsampled(
        self,
        written_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
        tmp_path: pathlib.Path,
    ) -> None:
        """Should keep one sample of the channel and its index in every factor"""
        index, data = written_pair
        tdms = _read_tdms(
            client, [data], tmp_path / "out.tdms", sy.StrideReduction(factor=2)
        )
        tdms_index, tdms_data = _tdms_group(tdms, index).channels()
        assert _tdms_timestamps(tdms_index) == [
            int(t) for t in seconds_linspace(1, count=10)[::2]
        ]
        np.testing.assert_array_equal(
            tdms_data[:], np.arange(0, 10, 2, dtype=np.float64)
        )

    def test_read_tdms_empty(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
        tmp_path: pathlib.Path,
    ) -> None:
        """Should declare the channel and its index when the range holds no samples"""
        index, data = indexed_pair
        dest = tmp_path / "out.tdms"
        tdms = _read_tdms(client, [data], dest)
        assert _tdms_segment_count(dest.read_bytes()) == 1
        assert [g.name for g in tdms.groups()] == [index.name]
        channels = tdms.groups()[0].channels()
        assert [c.name for c in channels] == [index.name, data.name]
        assert [len(c) for c in channels] == [0, 0]

    def test_read_empty(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return no samples when the channel has no data in the range"""
        _, data_ch = indexed_pair
        assert len(client.read(sy.TimeRange.MAX, data_ch.key)) == 0

    def test_read_empty_list(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return an empty series for a listed channel with no data"""
        _, data_ch = indexed_pair
        f = client.read(sy.TimeRange.MAX, [data_ch.key])
        assert len(f[data_ch.key]) == 0
