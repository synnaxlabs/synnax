#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio

import numpy as np
import pandas as pd
import pytest

import synnax as sy
from tests.telem import seconds_linspace


@pytest.mark.framer
@pytest.mark.writer
class TestWriter:
    def test_basic_write(self, indexed_pair: list[sy.Channel], client: sy.Synnax):
        """Should write a small amount of data to Synnax"""
        idx_ch, data_ch = indexed_pair
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND,
            channels=indexed_pair,
        ) as w:
            w.write(
                {
                    idx_ch.key: 1 * sy.TimeSpan.SECOND,
                    data_ch.key: 123.4,
                }
            )
            w.write(
                {
                    idx_ch.key: 2 * sy.TimeSpan.SECOND,
                    data_ch.key: 123.5,
                }
            )
            w.commit()

    def test_write_by_name(self, indexed_pair: list[sy.Channel], client: sy.Synnax):
        """Should write data by name to the Synnax cluster"""
        idx_ch, data_ch = indexed_pair
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND, channels=indexed_pair
        ) as w:
            w.write(
                pd.DataFrame(
                    {
                        idx_ch.name: [1 * sy.TimeSpan.SECOND],
                        data_ch.name: 253.2,
                    }
                )
            )
            w.write(
                pd.DataFrame(
                    {
                        idx_ch.name: [2 * sy.TimeSpan.SECOND],
                        data_ch.name: 253.3,
                    }
                )
            )
            w.commit()

    def test_write_frame_unknown_channel_name(
        self,
        indexed_pair: list[sy.Channel],
        client: sy.Synnax,
    ):
        """Should throw a validation error when writing to an unknown channel"""
        with client.open_writer(start=sy.TimeStamp.now(), channels=indexed_pair) as w:
            data = np.random.rand(10).astype(np.float64)
            with pytest.raises(sy.ValidationError):
                w.write(pd.DataFrame({"missing": data}))

    def test_write_frame_unknown_channel_key(
        self,
        indexed_pair: list[sy.Channel],
        client: sy.Synnax,
    ):
        """Should throw a validation error when writing an unknown frame by key"""
        with client.open_writer(start=sy.TimeStamp.now(), channels=indexed_pair) as w:
            data = np.random.rand(10).astype(np.float64)
            with pytest.raises(sy.ValidationError):
                w.write(pd.DataFrame({123: data}))

    def test_write_frame_idx_no_timestamps(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ):
        """Should throw a validation error on close when writing an indexed frame
        w/o the correct timing
        """
        [idx, data_ch] = indexed_pair
        with pytest.raises(
            sy.ValidationError, match="received no data for index channel"
        ):
            with client.open_writer(0, [idx.key, data_ch.key]) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({data_ch.key: data}))

    def test_write_auto_commit(self, indexed_pair: list[sy.Channel], client: sy.synnax):
        """Should open an auto-committing writer to write data that persists after 1s"""
        idx_ch, data_ch = indexed_pair
        with client.open_writer(
            start=sy.TimeSpan.SECOND * 1, channels=indexed_pair
        ) as w:
            data = np.random.rand(3).astype(np.float64)
            w.write(
                pd.DataFrame({idx_ch.name: seconds_linspace(1, 3), data_ch.key: data})
            )
            w.write(
                pd.DataFrame({idx_ch.key: seconds_linspace(4, 3), data_ch.name: data})
            )

        f = client.read(sy.TimeRange.MAX, data_ch.key)
        assert len(f) == 6

    def test_write_err_on_unauthorized(
        self, indexed_pair: list[sy.Channel], client: sy.Synnax
    ):
        """Should throw an error when a writer is opened to error on unauthorized"""
        start = sy.TimeStamp.now()
        w1 = client.open_writer(start=start, channels=indexed_pair, name="writer1")
        with pytest.raises(sy.UnauthorizedError):
            with client.open_writer(
                start=start,
                channels=indexed_pair,
                err_on_unauthorized=True,
                name="writer2",
            ):
                ...

        assert w1.close() is None

    def test_write_err_first_timestamp_before_start(
        self, client: sy.Synnax, indexed_pair: list[sy.Channel]
    ):
        """Should raise a validation error when the first timestamp written to an index
        channel is before the start time of the writer."""
        time_ch, data_ch = indexed_pair
        with pytest.raises(sy.ValidationError, match="commit timestamp"):
            with client.open_writer(
                start=sy.TimeStamp.now(),
                channels=[time_ch.key, data_ch.key],
            ) as w:
                for i in range(100):
                    w.write({time_ch.key: [i], data_ch.key: [i]})

    def test_write_out_of_order_timestamps(
        self, client: sy.Synnax, indexed_pair: list[sy.Channel]
    ):
        """Should raise a validation error when writing timestamps that are out of
        order"""
        time_ch, data_ch = indexed_pair
        with pytest.raises(sy.ValidationError, match="commit timestamp"):
            with client.open_writer(
                start=sy.TimeSpan.SECOND,
                channels=[time_ch.key, data_ch.key],
            ) as w:
                for i in range(100):
                    time = sy.TimeSpan.SECOND * (101 - i)
                    w.write({time_ch.key: time, data_ch.key: [i]})

    def test_write_invalid_data_type(
        self,
        client: sy.Synnax,
        indexed_pair: list[sy.Channel],
    ):
        """Should raise a validation error when writing a series with an invalid
        data type"""
        time_ch, data_ch = indexed_pair
        with pytest.raises(sy.ValidationError, match="type"):
            with client.open_writer(
                start=sy.TimeSpan.SECOND,
                channels=[time_ch.key, data_ch.key],
                strict=True,
            ) as w:
                w.write(
                    {
                        time_ch.key: [sy.TimeStamp.now()],
                        data_ch.key: sy.Series([1], data_type=sy.DataType.INT64),
                    }
                )

    @pytest.mark.asyncio
    async def test_write_persist_only_mode(
        self,
        indexed_pair: list[sy.Channel],
        client: sy.Synnax,
    ):
        """Should not stream written data"""
        idx_ch, data_ch = indexed_pair
        with client.open_writer(
            start=sy.TimeSpan.SECOND * 1,
            channels=indexed_pair,
            mode=sy.WriterMode.PERSIST,
        ) as w:
            async with await client.open_async_streamer(indexed_pair) as s:
                data = np.random.rand(5).astype(np.float64)
                times = seconds_linspace(1, 5)
                w.write(pd.DataFrame({idx_ch.key: times, data_ch.key: data}))
                with pytest.raises(TimeoutError):
                    async with asyncio.timeout(0.2):
                        await s.read()

    def test_write_persist_stream_regression(self, client: sy.Synnax):
        """Should work"""
        idx = client.channels.create(
            name="idx",
            is_index=True,
            data_type="timestamp",
        )
        data = client.channels.create(
            name="data",
            data_type="float64",
            index=idx.key,
        )
        # Write some data
        start = sy.TimeStamp.now()
        with client.open_writer(start, [idx.key, data.key]) as w:
            w.write({idx.key: [start], data.key: [1]})

        # Read the data
        next_start = start + 5 * sy.TimeSpan.MILLISECOND
        f = client.read(
            sy.TimeRange(start - 5 * sy.TimeSpan.MILLISECOND, next_start), data.key
        )
        assert len(f) == 1

        data_2 = client.channels.create(
            name="data_2",
            data_type="float64",
            index=idx.key,
        )
        data_3 = client.channels.create(
            name="data_3",
            data_type="float64",
            index=idx.key,
        )
        with client.open_writer(
            next_start, [idx.key, data.key, data_2.key, data_3.key]
        ) as w:
            w.write(
                {idx.key: [next_start], data.key: [1], data_2.key: [2], data_3.key: [3]}
            )

        tr = sy.TimeRange(
            start - 5 * sy.TimeSpan.MILLISECOND,
            next_start + 5 * sy.TimeSpan.MILLISECOND,
        )
        f = client.read(tr, data_2.key)
        assert len(f) == 1
        f2 = client.read(tr, [data.key, data_2.key, data_3.key])
        assert len(f2[data.key]) == 2
        assert len(f2[data_2.key]) == 1
        assert len(f2[data_3.key]) == 1

    def test_set_authority(self, client: sy.Synnax, indexed_pair: list[sy.channel]):
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, 10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority({data_ch.key: 255, idx_ch.key: 255})
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, 10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_by_name(
        self, client: sy.Synnax, indexed_pair: list[sy.channel]
    ):
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, 10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority({data_ch.name: 255, idx_ch.name: 255})
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, 10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_by_name_value(
        self, client: sy.Synnax, indexed_pair: list[sy.channel]
    ):
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, 10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority(data_ch.name, 255)
            w1.set_authority(idx_ch.name, 255)
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, 10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_writer_overlap_err(
        self,
        client: sy.Synnax,
        indexed_pair: list[sy.channel],
    ):
        idx_ch, data_ch = indexed_pair
        start = sy.TimeSpan.SECOND * 30
        with client.open_writer(start=start, channels=indexed_pair) as w:
            w.write(
                {
                    idx_ch.key: seconds_linspace(30, 10),
                    data_ch.key: np.random.rand(10).astype(np.float64),
                }
            )

        with pytest.raises(sy.ValidationError):
            with client.open_writer(
                start=start + sy.TimeSpan.SECOND * 3, channels=indexed_pair
            ):
                ...

    def test_set_authority_on_all_channels(
        self, client: sy.Synnax, indexed_pair: list[sy.channel]
    ):
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, 10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority(255)
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, 10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_writer_close_idempotency(
        self, indexed_pair: list[sy.Channel], client: sy.Synnax
    ):
        """Should allow the caller to call close() as many times as they want"""
        idx_ch, data_ch = indexed_pair
        w = client.open_writer(
            start=1 * sy.TimeSpan.SECOND,
            channels=indexed_pair,
            use_experimental_codec=True,
        )
        w.write(
            {
                idx_ch.key: 2 * sy.TimeSpan.SECOND,
                data_ch.key: 123.5,
            }
        )
        w.commit()
        w.close()
        w.close()
        w.close()

    # def test_writer_close_error(
    #     self,
    #     indexed_pair: list[sy.Channel],
    #     client: sy.Synnax
    # ):
