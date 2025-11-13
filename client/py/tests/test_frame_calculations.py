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
from synnax.util.rand import rand_name


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
                    name=rand_name(),
                    index=timestamp_channel.key,
                    data_type=sy.DataType.FLOAT32,
                ),
                sy.Channel(
                    name=rand_name(),
                    index=timestamp_channel.key,
                    data_type=sy.DataType.FLOAT32,
                ),
            ]
        )
        calc_channel = client.channels.create(
            name="test_calc",
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
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            virtual=True,
        )
        calc = client.channels.create(
            name="calc_channel",
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
            name=rand_name(),
            data_type=sy.DataType.UINT8,
            virtual=True,
        )
        calc = client.channels.create(
            name="calc_channel",
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
                name=rand_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        calc_channel = client.channels.create(
            name="test_calc_iter",
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

    def test_conditional_calculated_channel(self, client: sy.Synnax):
        """Should correctly create and read from a basic calculated channel using iteration"""
        timestamp_channel = client.channels.create(
            name="test_timestamp_iter",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src_channel = client.channels.create(
            sy.Channel(
                name=rand_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        calc_channel = client.channels.create(
            name="test_calc_iter",
            expression=f"""
            if ({src_channel.name} > 15) {{
                return 4
            }} else {{
                return 5
            }}
            """,
        )
        idx_data = [
            5 * sy.TimeSpan.SECOND,
            6 * sy.TimeSpan.SECOND,
            7 * sy.TimeSpan.SECOND,
        ]
        src_data = [10.0, 20.0, 30.0]
        client.write(
            5 * sy.TimeSpan.SECOND,
            {
                timestamp_channel.key: idx_data,
                src_channel.key: src_data,
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
        assert np.array_equal(ts_ser, np.array(idx_data, dtype=ts_ser.data_type.np))
        assert np.array_equal(
            data_ser, np.array([5, 4, 4], dtype=data_ser.data_type.np)
        )

    def test_calculation_deleted_channel(self, client: sy.Synnax):
        """Should correctly create and read from a basic calculated channel using iteration"""
        timestamp_channel = client.channels.create(
            name="test_timestamp_iter",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src_channel = client.channels.create(
            sy.Channel(
                name=rand_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        calc_channel = client.channels.create(
            name="test_calc_iter",
            expression=f"""
              if ({src_channel.name} > 15) {{
                  return 4
              }} else {{
                  return 5
              }}""",
        )
        idx_data = [
            5 * sy.TimeSpan.SECOND,
            6 * sy.TimeSpan.SECOND,
            7 * sy.TimeSpan.SECOND,
        ]
        src_data = [10.0, 20.0, 30.0]
        client.write(
            5 * sy.TimeSpan.SECOND,
            {
                timestamp_channel.key: idx_data,
                src_channel.key: src_data,
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
        assert np.array_equal(ts_ser, np.array(idx_data, dtype=ts_ser.data_type.np))
        assert np.array_equal(
            data_ser, np.array([5, 4, 4], dtype=data_ser.data_type.np)
        )
        client.channels.delete(src_channel.key)
        with pytest.raises(Exception, match="undefined symbol"):
            client.read(
                tr=sy.TimeRange.MAX,
                channels=[
                    calc_channel.key,
                    calc_channel.index,
                ],
            )

    def test_nested_calculated_channels_2_level(self, client: sy.Synnax):
        """Should correctly handle 2-level nested calculated channels (C → B → A)"""
        # Create index and base concrete channel
        timestamp_channel = client.channels.create(
            name="test_timestamp_nested",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        sensor_1 = client.channels.create(
            sy.Channel(
                name=rand_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )

        # Create B: calculated channel that depends on concrete channel A (sensor_1)
        calc_b = client.channels.create(
            name="calc_b_2level",
            expression=f"return {sensor_1.name} * 2",
        )

        # Create C: calculated channel that depends on calculated channel B
        calc_c = client.channels.create(
            name="calc_c_2level",
            expression=f"return {calc_b.name} + 10",
        )

        # Write data
        idx_data = [
            1 * sy.TimeSpan.SECOND,
            2 * sy.TimeSpan.SECOND,
            3 * sy.TimeSpan.SECOND,
            4 * sy.TimeSpan.SECOND,
            5 * sy.TimeSpan.SECOND,
        ]
        src_data = [1.0, 2.0, 3.0, 4.0, 5.0]
        client.write(
            1 * sy.TimeSpan.SECOND,
            {
                timestamp_channel.key: idx_data,
                sensor_1.key: src_data,
            },
        )

        # Read the top-level calculated channel C
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[calc_c.key, calc_c.index],
        )

        # Verify results
        # sensor_1 = [1, 2, 3, 4, 5]
        # calc_b = sensor_1 * 2 = [2, 4, 6, 8, 10]
        # calc_c = calc_b + 10 = [12, 14, 16, 18, 20]

        # Verify requested channels are present and have correct data
        assert calc_c.key in res.channels
        assert calc_c.index in res.channels

        data_ser = res[calc_c.key]
        ts_ser = res[calc_c.index]
        assert len(ts_ser) == 5
        assert np.array_equal(
            data_ser, np.array([12.0, 14.0, 16.0, 18.0, 20.0], dtype=np.float32)
        )
        assert np.array_equal(ts_ser, np.array(idx_data, dtype=ts_ser.data_type.np))

    def test_nested_calculated_channels_3_level(self, client: sy.Synnax):
        """Should correctly handle 3-level nested calculated channels (D → C → B → A)"""
        # Create index and base concrete channel
        timestamp_channel = client.channels.create(
            name="test_timestamp_nested_3",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        sensor_1 = client.channels.create(
            sy.Channel(
                name=rand_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )

        # Create B: depends on sensor_1 (concrete)
        calc_b = client.channels.create(
            name="calc_b_3level",
            expression=f"return {sensor_1.name} * 2",
        )

        # Create C: depends on B (calculated)
        calc_c = client.channels.create(
            name="calc_c_3level",
            expression=f"return {calc_b.name} + 5",
        )

        # Create D: depends on C (calculated)
        calc_d = client.channels.create(
            name="calc_d_3level",
            expression=f"return {calc_c.name} * 3",
        )

        # Write data
        idx_data = [
            1 * sy.TimeSpan.SECOND,
            2 * sy.TimeSpan.SECOND,
            3 * sy.TimeSpan.SECOND,
            4 * sy.TimeSpan.SECOND,
            5 * sy.TimeSpan.SECOND,
        ]
        src_data = [1.0, 2.0, 3.0, 4.0, 5.0]
        client.write(
            1 * sy.TimeSpan.SECOND,
            {
                timestamp_channel.key: idx_data,
                sensor_1.key: src_data,
            },
        )

        # Read the top-level calculated channel D
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[calc_d.key, calc_d.index],
        )

        # Verify results
        # sensor_1 = [1, 2, 3, 4, 5]
        # calc_b = sensor_1 * 2 = [2, 4, 6, 8, 10]
        # calc_c = calc_b + 5 = [7, 9, 11, 13, 15]
        # calc_d = calc_c * 3 = [21, 27, 33, 39, 45]
        assert calc_d.key in res.channels
        assert calc_d.index in res.channels

        data_ser = res[calc_d.key]
        ts_ser = res[calc_d.index]
        assert len(ts_ser) == 5
        assert np.array_equal(
            data_ser, np.array([21.0, 27.0, 33.0, 39.0, 45.0], dtype=np.float32)
        )
        assert np.array_equal(ts_ser, np.array(idx_data, dtype=ts_ser.data_type.np))

    def test_nested_calculated_channels_diamond(self, client: sy.Synnax):
        """Should correctly handle diamond dependency pattern (E → C & D → A)"""
        # Create index and base concrete channel
        timestamp_channel = client.channels.create(
            name="test_timestamp_diamond",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        sensor_1 = client.channels.create(
            sy.Channel(
                name=rand_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )

        # Create C: depends on sensor_1 (concrete)
        calc_c = client.channels.create(
            name="calc_c_diamond",
            expression=f"return {sensor_1.name} + 10",
        )

        # Create D: also depends on sensor_1 (concrete)
        calc_d = client.channels.create(
            name="calc_d_diamond",
            expression=f"return {sensor_1.name} * 5",
        )

        # Create E: depends on both C and D (calculated)
        calc_e = client.channels.create(
            name="calc_e_diamond",
            expression=f"return {calc_c.name} + {calc_d.name}",
        )

        # Write data
        idx_data = [
            1 * sy.TimeSpan.SECOND,
            2 * sy.TimeSpan.SECOND,
            3 * sy.TimeSpan.SECOND,
            4 * sy.TimeSpan.SECOND,
            5 * sy.TimeSpan.SECOND,
        ]
        src_data = [1.0, 2.0, 3.0, 4.0, 5.0]
        client.write(
            1 * sy.TimeSpan.SECOND,
            {
                timestamp_channel.key: idx_data,
                sensor_1.key: src_data,
            },
        )

        # Read the top-level calculated channel E
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[calc_e.key, calc_e.index],
        )

        # Verify results
        # sensor_1 = [1, 2, 3, 4, 5]
        # calc_c = sensor_1 + 10 = [11, 12, 13, 14, 15]
        # calc_d = sensor_1 * 5 = [5, 10, 15, 20, 25]
        # calc_e = calc_c + calc_d = [16, 22, 28, 34, 40]
        assert calc_e.key in res.channels
        assert calc_e.index in res.channels

        data_ser = res[calc_e.key]
        ts_ser = res[calc_e.index]
        assert len(ts_ser) == 5
        assert np.array_equal(
            data_ser, np.array([16.0, 22.0, 28.0, 34.0, 40.0], dtype=np.float32)
        )
        assert np.array_equal(ts_ser, np.array(idx_data, dtype=ts_ser.data_type.np))

    def test_circular_dependency_detection(self, client: sy.Synnax):
        """Should detect circular dependencies at channel creation time"""
        # In Python/API, Arc validates channel references at creation time,
        # so circular dependencies are caught immediately when trying to create
        # a channel that references a non-existent channel.

        # Trying to create a channel that references a non-existent channel fails
        with pytest.raises(Exception, match="undefined symbol"):
            client.channels.create(
                name="calc_circ_invalid",
                expression="return nonexistent_channel + 1",
            )

    def test_mixed_calculated_and_concrete_channels(self, client: sy.Synnax):
        """Should correctly handle requesting both calculated and concrete channels together"""
        # Create index and base concrete channels
        timestamp_channel = client.channels.create(
            name="test_timestamp_mixed",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        sensor_1 = client.channels.create(
            sy.Channel(
                name=rand_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        sensor_2 = client.channels.create(
            sy.Channel(
                name=rand_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )

        # Create calculated channel that depends on both sensors
        calc_mixed = client.channels.create(
            name="calc_mixed",
            expression=f"return {sensor_1.name} + {sensor_2.name}",
        )

        # Create nested calculated channel
        calc_mixed_nested = client.channels.create(
            name="calc_mixed_nested",
            expression=f"return {calc_mixed.name} * 2",
        )

        # Write data
        idx_data = [
            1 * sy.TimeSpan.SECOND,
            2 * sy.TimeSpan.SECOND,
            3 * sy.TimeSpan.SECOND,
            4 * sy.TimeSpan.SECOND,
            5 * sy.TimeSpan.SECOND,
        ]
        src_data_1 = [1.0, 2.0, 3.0, 4.0, 5.0]
        src_data_2 = [-2.0, -3.0, -4.0, -5.0, -6.0]
        client.write(
            1 * sy.TimeSpan.SECOND,
            {
                timestamp_channel.key: idx_data,
                sensor_1.key: src_data_1,
                sensor_2.key: src_data_2,
            },
        )

        # Read both concrete and calculated channels together
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[
                sensor_1.key,
                sensor_2.key,
                calc_mixed_nested.key,
                calc_mixed_nested.index,
            ],
        )

        # Verify concrete channels have original values
        assert sensor_1.key in res.channels
        data_ser = res[sensor_1.key]
        assert np.array_equal(data_ser, np.array(src_data_1, dtype=np.float32))

        assert sensor_2.key in res.channels
        data_ser = res[sensor_2.key]
        assert np.array_equal(data_ser, np.array(src_data_2, dtype=np.float32))

        # Verify calculated channel has correct values
        # sensor_1 = [1, 2, 3, 4, 5]
        # sensor_2 = [-2, -3, -4, -5, -6]
        # calc_mixed = sensor_1 + sensor_2 = [-1, -1, -1, -1, -1]
        # calc_mixed_nested = calc_mixed * 2 = [-2, -2, -2, -2, -2]
        assert calc_mixed_nested.key in res.channels
        data_ser = res[calc_mixed_nested.key]
        assert np.array_equal(
            data_ser, np.array([-2.0, -2.0, -2.0, -2.0, -2.0], dtype=np.float32)
        )


@pytest.mark.framer
@pytest.mark.calculations
class TestCalculationOperations:
    def test_avg_operation_accumulates_across_batches(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="avg")],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                                start + 3 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([10.0, 20.0, 30.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(20.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 4 * sy.TimeSpan.SECOND,
                                start + 5 * sy.TimeSpan.SECOND,
                                start + 6 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([40.0, 50.0, 60.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(35.0)

    def test_avg_duration_reset_triggers_on_boundary(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[
                sy.channel.Operation(type="avg", duration=5 * sy.TimeSpan.SECOND)
            ],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                                start + 3 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([10.0, 20.0, 30.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(20.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 6 * sy.TimeSpan.SECOND,
                                start + 7 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([40.0, 50.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(45.0)

    def test_min_operation_duration_reset(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[
                sy.channel.Operation(type="min", duration=5 * sy.TimeSpan.SECOND)
            ],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                                start + 3 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([30.0, 10.0, 20.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(10.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 6 * sy.TimeSpan.SECOND,
                                start + 7 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([50.0, 40.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(40.0)

    def test_max_operation_duration_reset(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[
                sy.channel.Operation(type="max", duration=5 * sy.TimeSpan.SECOND)
            ],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                                start + 3 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([10.0, 30.0, 20.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(30.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 6 * sy.TimeSpan.SECOND,
                                start + 7 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([15.0, 25.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(25.0)

    def test_avg_signal_reset_triggers_on_channel(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=rand_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="avg", reset_channel=reset.key)],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key, reset.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                                start + 3 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([10.0, 20.0, 30.0], dtype=np.float32),
                        reset.key: np.array([0, 0, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(20.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 4 * sy.TimeSpan.SECOND,
                                start + 5 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([40.0, 50.0], dtype=np.float32),
                        reset.key: np.array([1, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(45.0)

    def test_min_signal_reset_clears_state(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=rand_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="min", reset_channel=reset.key)],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key, reset.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                                start + 3 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([30.0, 10.0, 20.0], dtype=np.float32),
                        reset.key: np.array([0, 0, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(10.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 4 * sy.TimeSpan.SECOND,
                                start + 5 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([50.0, 40.0], dtype=np.float32),
                        reset.key: np.array([1, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(40.0)

    def test_max_signal_reset_clears_state(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=rand_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="max", reset_channel=reset.key)],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key, reset.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                                start + 3 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([10.0, 30.0, 20.0], dtype=np.float32),
                        reset.key: np.array([0, 0, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(30.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 4 * sy.TimeSpan.SECOND,
                                start + 5 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([15.0, 25.0], dtype=np.float32),
                        reset.key: np.array([1, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert len(frame[calc.key]) == 1
                assert frame[calc.key][0] == pytest.approx(25.0)

    def test_operations_with_single_sample(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc_min = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="min")],
        )
        calc_max = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="max")],
        )
        calc_avg = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="avg")],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(
            [calc_min.key, calc_max.key, calc_avg.key]
        ) as streamer:
            with client.open_writer(start, [idx.key, data.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [start + 1 * sy.TimeSpan.SECOND], dtype=np.int64
                        ),
                        data.key: np.array([42.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc_min.key][0] == pytest.approx(42.0)
                assert frame[calc_max.key][0] == pytest.approx(42.0)
                assert frame[calc_avg.key][0] == pytest.approx(42.0)

    def test_avg_multiple_duration_resets(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[
                sy.channel.Operation(type="avg", duration=3 * sy.TimeSpan.SECOND)
            ],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [start + 1 * sy.TimeSpan.SECOND], dtype=np.int64
                        ),
                        data.key: np.array([10.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc.key][0] == pytest.approx(10.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [start + 4 * sy.TimeSpan.SECOND], dtype=np.int64
                        ),
                        data.key: np.array([20.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc.key][0] == pytest.approx(20.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [start + 7 * sy.TimeSpan.SECOND], dtype=np.int64
                        ),
                        data.key: np.array([30.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc.key][0] == pytest.approx(30.0)

    def test_avg_combined_duration_and_signal_reset(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=rand_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[
                sy.channel.Operation(
                    type="avg",
                    duration=10 * sy.TimeSpan.SECOND,
                    reset_channel=reset.key,
                )
            ],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key, reset.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([10.0, 20.0], dtype=np.float32),
                        reset.key: np.array([0, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc.key][0] == pytest.approx(15.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [start + 3 * sy.TimeSpan.SECOND], dtype=np.int64
                        ),
                        data.key: np.array([30.0], dtype=np.float32),
                        reset.key: np.array([1], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc.key][0] == pytest.approx(30.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [start + 4 * sy.TimeSpan.SECOND], dtype=np.int64
                        ),
                        data.key: np.array([40.0], dtype=np.float32),
                        reset.key: np.array([0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc.key][0] == pytest.approx(35.0)

    def test_operations_with_identical_values(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc_min = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="min")],
        )
        calc_max = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="max")],
        )
        calc_avg = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="avg")],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(
            [calc_min.key, calc_max.key, calc_avg.key]
        ) as streamer:
            with client.open_writer(start, [idx.key, data.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                                start + 3 * sy.TimeSpan.SECOND,
                                start + 4 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([5.0, 5.0, 5.0, 5.0], dtype=np.float32),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc_min.key][0] == pytest.approx(5.0)
                assert frame[calc_max.key][0] == pytest.approx(5.0)
                assert frame[calc_avg.key][0] == pytest.approx(5.0)

    def test_avg_signal_reset_fast_pulse(self, client: sy.Synnax):
        idx = client.channels.create(
            name=rand_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=rand_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=rand_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=rand_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="avg", reset_channel=reset.key)],
        )
        start = sy.TimeStamp.now()
        with client.open_streamer(calc.key) as streamer:
            with client.open_writer(start, [idx.key, data.key, reset.key]) as writer:
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 1 * sy.TimeSpan.SECOND,
                                start + 2 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([10.0, 20.0], dtype=np.float32),
                        reset.key: np.array([0, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc.key][0] == pytest.approx(15.0)
                writer.write(
                    {
                        idx.key: np.array(
                            [
                                start + 3 * sy.TimeSpan.SECOND,
                                start + 4 * sy.TimeSpan.SECOND,
                                start + 5 * sy.TimeSpan.SECOND,
                            ],
                            dtype=np.int64,
                        ),
                        data.key: np.array([30.0, 40.0, 50.0], dtype=np.float32),
                        reset.key: np.array([1, 0, 1], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=1)
                assert frame is not None
                assert frame[calc.key][0] == pytest.approx(40.0)
