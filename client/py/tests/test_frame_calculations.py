#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
import time

import numpy as np
import pytest

import synnax as sy


@pytest.mark.framer
@pytest.mark.calculations
class TestCalculatedChannelStreaming:
    def test_basic_calculated_channel_stream(self, client: sy.Synnax):
        """Should correctly create and read from a basic calculated channel using streaming"""
        timestamp_channel = client.channels.create(
            name="test_timestamp",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src_channels = client.channels.create(
            [
                sy.Channel(
                    name=f"test_a_54732",
                    index=timestamp_channel.key,
                    data_type=sy.DataType.FLOAT32,
                ),
                sy.Channel(
                    name=f"test_b_67832",
                    index=timestamp_channel.key,
                    data_type=sy.DataType.FLOAT32,
                ),
            ]
        )
        calc_channel = client.channels.create(
            name="test_calc",
            data_type=sy.DataType.FLOAT32,
            expression=f"return {src_channels[0].name} + {src_channels[1].name}",
        )
        start = sy.TimeStamp.now()
        value = np.array(
            [2.0],
            dtype=np.float32,
        )
        with client.open_streamer(calc_channel.key) as streamer:
            with client.open_writer(
                start,
                [timestamp_channel.key, src_channels[0].key, src_channels[1].key],
            ) as writer:
                alignment_hwm = 0
                for i in range(5):
                    time.sleep(0.01)
                    writer.write(
                        {
                            timestamp_channel.key: sy.TimeStamp.now(),
                            src_channels[0].key: value / 2,
                            src_channels[1].key: value / 2,
                        }
                    )
                    frame = streamer.read(timeout=100)
                    assert len(frame.channels) == 1
                    ser = frame[calc_channel.key]
                    assert ser.alignment != 0
                    if i != 0:
                        alignment_hwm += 1
                        assert ser.alignment == alignment_hwm
                    else:
                        alignment_hwm = ser.alignment
                    assert frame is not None
                    assert np.array_equal(frame[calc_channel.key], value)

    def test_stream_passthrough_virtual_channel(self, client: sy.Synnax):
        virt = client.channels.create(
            name=f"virtual_channel_4463",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
        )
        calc = client.channels.create(
            name="calc_channel",
            data_type=sy.DataType.FLOAT32,
            expression=f"return {virt.name} * 2",
        )
        start = sy.TimeStamp.now()
        value = np.array([3], dtype=np.uint8)
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(
                start,
                [virt.key],
            ) as writer:
                writer.write(
                    {
                        virt.key: value,
                    }
                )
                writer.commit()
                frame = streamer.read(timeout=2)
                assert frame is not None
                assert np.array_equal(frame[calc.key], value * 2)

    def test_stream_passthrough_virtual_u8_channel(self, client: sy.Synnax):
        virt = client.channels.create(
            name=f"virtual_channel_4463",
            data_type=sy.DataType.UINT8,
            virtual=True,
        )
        calc = client.channels.create(
            name="calc_channel",
            data_type=sy.DataType.UINT8,
            expression=f"return {virt.name} * 2",
        )
        start = sy.TimeStamp.now()
        value = np.array([3], dtype=np.uint8)
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(
                start,
                [virt.key],
            ) as writer:
                writer.write(
                    {
                        virt.key: value,
                    }
                )
                writer.commit()
                frame = streamer.read(timeout=2)
                assert frame is not None
                assert np.array_equal(frame[calc.key], value * 2)


@pytest.mark.framer
@pytest.mark.calculations
class TestCalculatedChannelIteration:
    def test_basic_calculated_channel_iterate(self, client: sy.Synnax):
        """Should correctly create and read from a basic calculated channel using iteration"""
        timestamp_channel = client.channels.create(
            name="test_timestamp_iter",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src_channel = client.channels.create(
            sy.Channel(
                name=f"test_a_iter_54732",
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        calc_channel = client.channels.create(
            name="test_calc_iter",
            data_type=sy.DataType.FLOAT32,
            expression=f"return {src_channel.name}",
        )
        idx_data_1 = [
            5 * sy.TimeSpan.SECOND,
            6 * sy.TimeSpan.SECOND,
            7 * sy.TimeSpan.SECOND,
        ]
        src_data_1 = [10.0, 20.0, 30.0]
        client.write(
            5 * sy.TimeSpan.SECOND,
            {
                timestamp_channel.key: idx_data_1,
                src_channel.key: src_data_1,
            },
        )
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[
                calc_channel.key,
                calc_channel.index,
            ],
        )
        assert len(res.channels) == 2
        ts_ser = res[calc_channel.index]
        data_ser = res[calc_channel.key]
        assert len(ts_ser) == 3
        assert ts_ser.alignment == data_ser.alignment
        assert np.array_equal(ts_ser, np.array(idx_data_1, dtype=ts_ser.data_type.np))
        assert np.array_equal(
            data_ser, np.array(src_data_1, dtype=data_ser.data_type.np)
        )

        idx_data_2 = [
            9 * sy.TimeSpan.SECOND,
            10 * sy.TimeSpan.SECOND,
            11 * sy.TimeSpan.SECOND,
        ]
        src_data_2 = [50.0, 60.0, 70.0]
        client.write(
            9 * sy.TimeSpan.SECOND,
            {
                timestamp_channel.key: idx_data_2,
                src_channel.key: src_data_2,
            },
        )
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[calc_channel.key, calc_channel.index],
        )
        assert len(res.channels) == 4
        ts_ser = res[calc_channel.index]
        data_ser = res[calc_channel.key]
        assert len(ts_ser) == 6
        assert ts_ser.alignment == data_ser.alignment
        assert np.array_equal(
            ts_ser,
            np.array(idx_data_1 + idx_data_2, dtype=ts_ser.data_type.np),
        )

        idx_data_0 = [1 * sy.TimeSpan.SECOND, 2 * sy.TimeSpan.SECOND]
        src_data_0 = [0.0, 5.0]
        client.write(
            1 * sy.TimeSpan.SECOND,
            {
                timestamp_channel.key: idx_data_0,
                src_channel.key: src_data_0,
            },
        )
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[calc_channel.key, calc_channel.index],
        )
        assert len(res.channels) == 6
        data_ser = res[calc_channel.key]
        ts_ser = res[calc_channel.index]
        assert len(ts_ser) == 8
        assert ts_ser.alignment == data_ser.alignment
        assert np.array_equal(
            ts_ser,
            np.array(idx_data_0 + idx_data_1 + idx_data_2, dtype=ts_ser.data_type.np),
        )
        assert np.array_equal(
            data_ser,
            np.array(src_data_0 + src_data_1 + src_data_2, dtype=data_ser.data_type.np),
        )
