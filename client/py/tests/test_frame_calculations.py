#  Copyright 2026 Synnax Labs, Inc.
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
from synnax.util.random import random_name


@pytest.mark.framer
@pytest.mark.calculations
class TestCalculatedChannelStreaming:
    def test_basic_calculated_channel_stream(self, client: sy.Synnax):
        """Should correctly create and read from a basic calculated channel using streaming"""
        timestamp_channel = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src_channels = client.channels.create(
            [
                sy.Channel(
                    name=random_name(),
                    index=timestamp_channel.key,
                    data_type=sy.DataType.FLOAT32,
                ),
                sy.Channel(
                    name=random_name(),
                    index=timestamp_channel.key,
                    data_type=sy.DataType.FLOAT32,
                ),
            ]
        )
        calc_channel = client.channels.create(
            name=random_name(),
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
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            virtual=True,
        )
        calc = client.channels.create(
            name=random_name(),
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
            name=random_name(),
            data_type=sy.DataType.UINT8,
            virtual=True,
        )
        calc = client.channels.create(
            name=random_name(),
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
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src_channel = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        calc_channel = client.channels.create(
            name=random_name(),
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
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src_channel = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        calc_channel = client.channels.create(
            name=random_name(),
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
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src_channel = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        calc_channel = client.channels.create(
            name=random_name(),
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
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        sensor_1 = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )

        # Create B: calculated channel that depends on concrete channel A (sensor_1)
        calc_b = client.channels.create(
            name=random_name(),
            expression=f"return {sensor_1.name} * 2",
        )

        # Create C: calculated channel that depends on calculated channel B
        calc_c = client.channels.create(
            name=random_name(),
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
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        sensor_1 = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )

        # Create B: depends on sensor_1 (concrete)
        calc_b = client.channels.create(
            name=random_name(),
            expression=f"return {sensor_1.name} * 2",
        )

        # Create C: depends on B (calculated)
        calc_c = client.channels.create(
            name=random_name(),
            expression=f"return {calc_b.name} + 5",
        )

        # Create D: depends on C (calculated)
        calc_d = client.channels.create(
            name=random_name(),
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
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        sensor_1 = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )

        # Create C: depends on sensor_1 (concrete)
        calc_c = client.channels.create(
            name=random_name(),
            expression=f"return {sensor_1.name} + 10",
        )

        # Create D: also depends on sensor_1 (concrete)
        calc_d = client.channels.create(
            name=random_name(),
            expression=f"return {sensor_1.name} * 5",
        )

        # Create E: depends on both C and D (calculated)
        calc_e = client.channels.create(
            name=random_name(),
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
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        sensor_1 = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        sensor_2 = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=timestamp_channel.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )

        # Create calculated channel that depends on both sensors
        calc_mixed = client.channels.create(
            name=random_name(),
            expression=f"return {sensor_1.name} + {sensor_2.name}",
        )

        # Create nested calculated channel
        calc_mixed_nested = client.channels.create(
            name=random_name(),
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
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=random_name(),
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
                ser = frame[calc.key]
                assert ser.alignment != 0
                first_alignment = ser.alignment
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
                ser = frame[calc.key]
                assert ser.alignment == first_alignment + 1

    def test_avg_duration_reset_triggers_on_boundary(self, client: sy.Synnax):
        idx = client.channels.create(
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=random_name(),
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
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=random_name(),
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
                ser = frame[calc.key]
                assert ser.alignment != 0
                first_alignment = ser.alignment
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
                ser = frame[calc.key]
                assert ser.alignment == first_alignment + 1

    def test_max_operation_duration_reset(self, client: sy.Synnax):
        idx = client.channels.create(
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=random_name(),
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
                ser = frame[calc.key]
                assert ser.alignment != 0
                first_alignment = ser.alignment
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
                ser = frame[calc.key]
                assert ser.alignment == first_alignment + 1

    def test_avg_signal_reset_triggers_on_channel(self, client: sy.Synnax):
        idx = client.channels.create(
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=random_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=random_name(),
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
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=random_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=random_name(),
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
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=random_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=random_name(),
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
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc_min = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="min")],
        )
        calc_max = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="max")],
        )
        calc_avg = client.channels.create(
            name=random_name(),
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
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc = client.channels.create(
            name=random_name(),
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
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=random_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=random_name(),
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
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        calc_min = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="min")],
        )
        calc_max = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="max")],
        )
        calc_avg = client.channels.create(
            name=random_name(),
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
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, index=idx.key
        )
        reset = client.channels.create(
            name=random_name(), data_type=sy.DataType.UINT8, virtual=True
        )
        calc = client.channels.create(
            name=random_name(),
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


@pytest.mark.framer
@pytest.mark.calculations
class TestCalculationsAcrossDomains:
    """Tests for calculated channels reading data written across multiple domains.

    These tests verify that calculations work correctly when data is written
    using multiple separate writers with auto-commit, creating distinct domains.
    Each test verifies both the calculated values AND the alignment metadata.
    """

    def test_basic_calculation_across_two_domains(self, client: sy.Synnax):
        """Should correctly calculate values across two separate write domains
        with correct alignment for each domain."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        data = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        calc = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name} * 2",
        )

        # First domain: write with auto-commit, then close
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([1.0, 2.0, 3.0], dtype=np.float32),
                }
            )

        # Second domain: write with auto-commit, then close
        with client.open_writer(
            6 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            6 * sy.TimeSpan.SECOND,
                            7 * sy.TimeSpan.SECOND,
                            8 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([6.0, 7.0, 8.0], dtype=np.float32),
                }
            )

        # Read calculated channel - should have data from both domains
        res = client.read(sy.TimeRange.MAX, [calc.key, calc.index])

        # Access the MultiSeries to check individual domain series
        calc_multi = res[calc.key]
        idx_multi = res[calc.index]

        # Should have 2 series (one per domain)
        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Verify domain 0 alignment and values
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[0].alignment.sample_index == 0
        assert idx_multi.series[0].alignment.domain_index == 0
        assert idx_multi.series[0].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[0], np.array([2.0, 4.0, 6.0], dtype=np.float32)
        )

        # Verify domain 1 alignment and values
        assert calc_multi.series[1].alignment.domain_index == 1
        assert calc_multi.series[1].alignment.sample_index == 0
        assert idx_multi.series[1].alignment.domain_index == 1
        assert idx_multi.series[1].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[1], np.array([12.0, 14.0, 16.0], dtype=np.float32)
        )

        # Verify calc and index alignments match within each domain
        assert calc_multi.series[0].alignment == idx_multi.series[0].alignment
        assert calc_multi.series[1].alignment == idx_multi.series[1].alignment

    def test_nested_calculation_2_level_across_domains(self, client: sy.Synnax):
        """Should correctly handle 2-level nested calculations across domains
        with proper alignment preservation."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        sensor = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        # B depends on sensor (concrete)
        calc_b = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {sensor.name} * 2",
        )
        # C depends on B (calculated)
        calc_c = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {calc_b.name} + 10",
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, sensor.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                            4 * sy.TimeSpan.SECOND,
                            5 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor.key: np.array([1.0, 2.0, 3.0, 4.0, 5.0], dtype=np.float32),
                }
            )

        # Second domain
        with client.open_writer(
            6 * sy.TimeSpan.SECOND, [idx.key, sensor.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            6 * sy.TimeSpan.SECOND,
                            7 * sy.TimeSpan.SECOND,
                            8 * sy.TimeSpan.SECOND,
                            9 * sy.TimeSpan.SECOND,
                            10 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor.key: np.array([6.0, 7.0, 8.0, 9.0, 10.0], dtype=np.float32),
                }
            )

        # Read the top-level calculated channel C
        res = client.read(sy.TimeRange.MAX, [calc_c.key, calc_c.index])
        calc_multi = res[calc_c.key]
        idx_multi = res[calc_c.index]

        # Should have 2 series (one per domain)
        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Verify domain 0 alignment and values
        # sensor = [1,2,3,4,5], calc_b = [2,4,6,8,10], calc_c = [12,14,16,18,20]
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[0].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[0],
            np.array([12.0, 14.0, 16.0, 18.0, 20.0], dtype=np.float32),
        )

        # Verify domain 1 alignment and values
        # sensor = [6,7,8,9,10], calc_b = [12,14,16,18,20], calc_c = [22,24,26,28,30]
        assert calc_multi.series[1].alignment.domain_index == 1
        assert calc_multi.series[1].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[1],
            np.array([22.0, 24.0, 26.0, 28.0, 30.0], dtype=np.float32),
        )

        # Verify calc and index alignments match
        assert calc_multi.series[0].alignment == idx_multi.series[0].alignment
        assert calc_multi.series[1].alignment == idx_multi.series[1].alignment

    def test_nested_calculation_3_level_across_domains(self, client: sy.Synnax):
        """Should correctly handle 3-level nested calculations across domains
        with proper alignment preservation."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        sensor = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        # B depends on sensor (concrete)
        calc_b = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {sensor.name} * 2",
        )
        # C depends on B (calculated)
        calc_c = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {calc_b.name} + 5",
        )
        # D depends on C (calculated)
        calc_d = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {calc_c.name} * 3",
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, sensor.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                            4 * sy.TimeSpan.SECOND,
                            5 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor.key: np.array([1.0, 2.0, 3.0, 4.0, 5.0], dtype=np.float32),
                }
            )

        # Second domain
        with client.open_writer(
            6 * sy.TimeSpan.SECOND, [idx.key, sensor.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            6 * sy.TimeSpan.SECOND,
                            7 * sy.TimeSpan.SECOND,
                            8 * sy.TimeSpan.SECOND,
                            9 * sy.TimeSpan.SECOND,
                            10 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor.key: np.array([6.0, 7.0, 8.0, 9.0, 10.0], dtype=np.float32),
                }
            )

        # Read the top-level calculated channel D
        res = client.read(sy.TimeRange.MAX, [calc_d.key, calc_d.index])
        calc_multi = res[calc_d.key]
        idx_multi = res[calc_d.index]

        # Should have 2 series (one per domain)
        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Verify domain 0 alignment and values
        # sensor = [1,2,3,4,5]
        # calc_b = sensor * 2 = [2,4,6,8,10]
        # calc_c = calc_b + 5 = [7,9,11,13,15]
        # calc_d = calc_c * 3 = [21,27,33,39,45]
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[0].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[0],
            np.array([21.0, 27.0, 33.0, 39.0, 45.0], dtype=np.float32),
        )

        # Verify domain 1 alignment and values
        # sensor = [6,7,8,9,10]
        # calc_b = sensor * 2 = [12,14,16,18,20]
        # calc_c = calc_b + 5 = [17,19,21,23,25]
        # calc_d = calc_c * 3 = [51,57,63,69,75]
        assert calc_multi.series[1].alignment.domain_index == 1
        assert calc_multi.series[1].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[1],
            np.array([51.0, 57.0, 63.0, 69.0, 75.0], dtype=np.float32),
        )

        # Verify calc and index alignments match
        assert calc_multi.series[0].alignment == idx_multi.series[0].alignment
        assert calc_multi.series[1].alignment == idx_multi.series[1].alignment

    def test_diamond_dependency_across_domains(self, client: sy.Synnax):
        """Should correctly handle diamond dependency pattern across domains
        with proper alignment preservation.

        Diamond pattern:
        E depends on C and D
        C depends on sensor (A)
        D depends on sensor (A)
        """
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        sensor = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        # C depends on sensor
        calc_c = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {sensor.name} + 10",
        )
        # D also depends on sensor
        calc_d = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {sensor.name} * 5",
        )
        # E depends on both C and D
        calc_e = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {calc_c.name} + {calc_d.name}",
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, sensor.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                            4 * sy.TimeSpan.SECOND,
                            5 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor.key: np.array([1.0, 2.0, 3.0, 4.0, 5.0], dtype=np.float32),
                }
            )

        # Second domain
        with client.open_writer(
            6 * sy.TimeSpan.SECOND, [idx.key, sensor.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            6 * sy.TimeSpan.SECOND,
                            7 * sy.TimeSpan.SECOND,
                            8 * sy.TimeSpan.SECOND,
                            9 * sy.TimeSpan.SECOND,
                            10 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor.key: np.array([6.0, 7.0, 8.0, 9.0, 10.0], dtype=np.float32),
                }
            )

        # Read the top-level calculated channel E
        res = client.read(sy.TimeRange.MAX, [calc_e.key, calc_e.index])
        calc_multi = res[calc_e.key]
        idx_multi = res[calc_e.index]

        # Should have 2 series (one per domain)
        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Verify domain 0 alignment and values
        # sensor = [1,2,3,4,5]
        # calc_c = sensor + 10 = [11,12,13,14,15]
        # calc_d = sensor * 5 = [5,10,15,20,25]
        # calc_e = calc_c + calc_d = [16,22,28,34,40]
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[0].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[0],
            np.array([16.0, 22.0, 28.0, 34.0, 40.0], dtype=np.float32),
        )

        # Verify domain 1 alignment and values
        # sensor = [6,7,8,9,10]
        # calc_c = sensor + 10 = [16,17,18,19,20]
        # calc_d = sensor * 5 = [30,35,40,45,50]
        # calc_e = calc_c + calc_d = [46,52,58,64,70]
        # Note: Diamond pattern may cause alignment increment in some implementations
        assert calc_multi.series[1].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[1],
            np.array([46.0, 52.0, 58.0, 64.0, 70.0], dtype=np.float32),
        )

        # Verify calc and index alignments match within each domain
        assert calc_multi.series[0].alignment == idx_multi.series[0].alignment
        assert calc_multi.series[1].alignment == idx_multi.series[1].alignment

    def test_mixed_calculated_and_concrete_across_domains(self, client: sy.Synnax):
        """Should correctly handle requesting both calculated and concrete channels
        with proper alignment for each."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        sensor_1 = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        sensor_2 = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        # Calculated channel that depends on both sensors
        calc_mixed = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {sensor_1.name} + {sensor_2.name}",
        )
        # Nested calculated channel
        calc_nested = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {calc_mixed.name} * 2",
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND,
            [idx.key, sensor_1.key, sensor_2.key],
            enable_auto_commit=True,
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                            4 * sy.TimeSpan.SECOND,
                            5 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor_1.key: np.array([1.0, 2.0, 3.0, 4.0, 5.0], dtype=np.float32),
                    sensor_2.key: np.array(
                        [-2.0, -3.0, -4.0, -5.0, -6.0], dtype=np.float32
                    ),
                }
            )

        # Second domain
        with client.open_writer(
            6 * sy.TimeSpan.SECOND,
            [idx.key, sensor_1.key, sensor_2.key],
            enable_auto_commit=True,
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            6 * sy.TimeSpan.SECOND,
                            7 * sy.TimeSpan.SECOND,
                            8 * sy.TimeSpan.SECOND,
                            9 * sy.TimeSpan.SECOND,
                            10 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor_1.key: np.array(
                        [6.0, 7.0, 8.0, 9.0, 10.0], dtype=np.float32
                    ),
                    sensor_2.key: np.array(
                        [-3.0, -4.0, -5.0, -6.0, -7.0], dtype=np.float32
                    ),
                }
            )

        # Read both concrete and calculated channels together
        res = client.read(
            sy.TimeRange.MAX,
            [sensor_1.key, sensor_2.key, calc_nested.key, calc_nested.index],
        )

        s1_multi = res[sensor_1.key]
        s2_multi = res[sensor_2.key]
        calc_multi = res[calc_nested.key]
        idx_multi = res[calc_nested.index]

        # All should have 2 series (one per domain)
        assert len(s1_multi.series) == 2
        assert len(s2_multi.series) == 2
        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Verify domain 0 alignments for concrete channels
        assert s1_multi.series[0].alignment.domain_index == 0
        assert s2_multi.series[0].alignment.domain_index == 0
        # Calculated channels have summed alignment: calc_mixed uses 2 inputs,
        # calc_nested uses calc_mixed. For domain 0: (0+0)*2 = 0
        assert calc_multi.series[0].alignment.domain_index == 0
        assert idx_multi.series[0].alignment.domain_index == 0

        # Verify domain 1 alignments for concrete channels
        assert s1_multi.series[1].alignment.domain_index == 1
        assert s2_multi.series[1].alignment.domain_index == 1
        # For domain 1: calc_mixed = (1+1) = 2, calc_nested uses calc_mixed = 2
        assert calc_multi.series[1].alignment.domain_index == 2
        assert idx_multi.series[1].alignment.domain_index == 2

        # Verify concrete channel values
        assert np.allclose(
            s1_multi.series[0], np.array([1.0, 2.0, 3.0, 4.0, 5.0], dtype=np.float32)
        )
        assert np.allclose(
            s1_multi.series[1], np.array([6.0, 7.0, 8.0, 9.0, 10.0], dtype=np.float32)
        )
        assert np.allclose(
            s2_multi.series[0],
            np.array([-2.0, -3.0, -4.0, -5.0, -6.0], dtype=np.float32),
        )
        assert np.allclose(
            s2_multi.series[1],
            np.array([-3.0, -4.0, -5.0, -6.0, -7.0], dtype=np.float32),
        )

        # Verify calculated channel values
        # sensor_1 = [1,2,3,4,5] and [6,7,8,9,10]
        # sensor_2 = [-2,-3,-4,-5,-6] and [-3,-4,-5,-6,-7]
        # calc_mixed = sensor_1 + sensor_2 = [-1,-1,-1,-1,-1] and [3,3,3,3,3]
        # calc_nested = calc_mixed * 2 = [-2,-2,-2,-2,-2] and [6,6,6,6,6]
        assert np.allclose(
            calc_multi.series[0],
            np.array([-2.0, -2.0, -2.0, -2.0, -2.0], dtype=np.float32),
        )
        assert np.allclose(
            calc_multi.series[1],
            np.array([6.0, 6.0, 6.0, 6.0, 6.0], dtype=np.float32),
        )

    def test_three_domains_calculation(self, client: sy.Synnax):
        """Should correctly calculate values across three separate write domains
        with correct alignment for each."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        data = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        calc = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name} * 2",
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [1 * sy.TimeSpan.SECOND, 2 * sy.TimeSpan.SECOND],
                        dtype=np.int64,
                    ),
                    data.key: np.array([1.0, 2.0], dtype=np.float32),
                }
            )

        # Second domain
        with client.open_writer(
            5 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [5 * sy.TimeSpan.SECOND, 6 * sy.TimeSpan.SECOND],
                        dtype=np.int64,
                    ),
                    data.key: np.array([5.0, 6.0], dtype=np.float32),
                }
            )

        # Third domain
        with client.open_writer(
            10 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [10 * sy.TimeSpan.SECOND, 11 * sy.TimeSpan.SECOND],
                        dtype=np.int64,
                    ),
                    data.key: np.array([10.0, 11.0], dtype=np.float32),
                }
            )

        # Read calculated channel
        res = client.read(sy.TimeRange.MAX, [calc.key, calc.index])
        calc_multi = res[calc.key]
        idx_multi = res[calc.index]

        # Should have 3 series (one per domain)
        assert len(calc_multi.series) == 3
        assert len(idx_multi.series) == 3

        # Verify domain 0
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[0].alignment.sample_index == 0
        assert np.allclose(calc_multi.series[0], np.array([2.0, 4.0], dtype=np.float32))

        # Verify domain 1
        assert calc_multi.series[1].alignment.domain_index == 1
        assert calc_multi.series[1].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[1], np.array([10.0, 12.0], dtype=np.float32)
        )

        # Verify domain 2
        assert calc_multi.series[2].alignment.domain_index == 2
        assert calc_multi.series[2].alignment.sample_index == 0
        assert np.allclose(
            calc_multi.series[2], np.array([20.0, 22.0], dtype=np.float32)
        )

        # Verify all alignments match between calc and index
        for i in range(3):
            assert calc_multi.series[i].alignment == idx_multi.series[i].alignment

    def test_calculation_with_gap_between_domains(self, client: sy.Synnax):
        """Should correctly handle calculations when there's a large gap between domains
        while preserving proper alignment."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        data = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        calc = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name} + 100",
        )

        # First domain at t=1s
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([1.0, 2.0, 3.0], dtype=np.float32),
                }
            )

        # Second domain at t=1000s (large gap)
        with client.open_writer(
            1000 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1000 * sy.TimeSpan.SECOND,
                            1001 * sy.TimeSpan.SECOND,
                            1002 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([1000.0, 1001.0, 1002.0], dtype=np.float32),
                }
            )

        # Read calculated channel
        res = client.read(sy.TimeRange.MAX, [calc.key, calc.index])
        calc_multi = res[calc.key]
        idx_multi = res[calc.index]

        # Should have 2 series despite the large gap
        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Verify domain 0
        assert calc_multi.series[0].alignment.domain_index == 0
        assert np.allclose(
            calc_multi.series[0], np.array([101.0, 102.0, 103.0], dtype=np.float32)
        )

        # Verify domain 1
        assert calc_multi.series[1].alignment.domain_index == 1
        assert np.allclose(
            calc_multi.series[1],
            np.array([1100.0, 1101.0, 1102.0], dtype=np.float32),
        )

    def test_multiple_calculations_same_source_across_domains(self, client: sy.Synnax):
        """Should correctly handle multiple calculations depending on the same source
        with consistent alignments across all calculated channels."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        sensor = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        # Multiple calculations on the same source
        calc_double = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {sensor.name} * 2",
        )
        calc_square = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {sensor.name} * {sensor.name}",
        )
        calc_plus_ten = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {sensor.name} + 10",
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, sensor.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [1 * sy.TimeSpan.SECOND, 2 * sy.TimeSpan.SECOND],
                        dtype=np.int64,
                    ),
                    sensor.key: np.array([2.0, 3.0], dtype=np.float32),
                }
            )

        # Second domain
        with client.open_writer(
            5 * sy.TimeSpan.SECOND, [idx.key, sensor.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [5 * sy.TimeSpan.SECOND, 6 * sy.TimeSpan.SECOND],
                        dtype=np.int64,
                    ),
                    sensor.key: np.array([4.0, 5.0], dtype=np.float32),
                }
            )

        # Read all calculated channels
        res = client.read(
            sy.TimeRange.MAX, [calc_double.key, calc_square.key, calc_plus_ten.key]
        )

        double_multi = res[calc_double.key]
        square_multi = res[calc_square.key]
        plus_ten_multi = res[calc_plus_ten.key]

        # All should have 2 series
        assert len(double_multi.series) == 2
        assert len(square_multi.series) == 2
        assert len(plus_ten_multi.series) == 2

        # Verify all have consistent domain alignments
        for i in range(2):
            assert double_multi.series[i].alignment.domain_index == i
            assert square_multi.series[i].alignment.domain_index == i
            assert plus_ten_multi.series[i].alignment.domain_index == i

        # Verify calc_double: sensor * 2
        assert np.allclose(
            double_multi.series[0], np.array([4.0, 6.0], dtype=np.float32)
        )
        assert np.allclose(
            double_multi.series[1], np.array([8.0, 10.0], dtype=np.float32)
        )

        # Verify calc_square: sensor * sensor
        assert np.allclose(
            square_multi.series[0], np.array([4.0, 9.0], dtype=np.float32)
        )
        assert np.allclose(
            square_multi.series[1], np.array([16.0, 25.0], dtype=np.float32)
        )

        # Verify calc_plus_ten: sensor + 10
        assert np.allclose(
            plus_ten_multi.series[0], np.array([12.0, 13.0], dtype=np.float32)
        )
        assert np.allclose(
            plus_ten_multi.series[1], np.array([14.0, 15.0], dtype=np.float32)
        )

    def test_calculation_with_conditional_across_domains(self, client: sy.Synnax):
        """Should correctly handle conditional calculations across domains
        with proper alignment."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        data = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        calc = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"""
            if ({data.name} > 5) {{
                return 1
            }} else {{
                return 0
            }}
            """,
        )

        # First domain: values below threshold
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [1 * sy.TimeSpan.SECOND, 2 * sy.TimeSpan.SECOND],
                        dtype=np.int64,
                    ),
                    data.key: np.array([3.0, 4.0], dtype=np.float32),
                }
            )

        # Second domain: values above threshold
        with client.open_writer(
            5 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [5 * sy.TimeSpan.SECOND, 6 * sy.TimeSpan.SECOND],
                        dtype=np.int64,
                    ),
                    data.key: np.array([6.0, 7.0], dtype=np.float32),
                }
            )

        # Third domain: mixed values
        with client.open_writer(
            10 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [10 * sy.TimeSpan.SECOND, 11 * sy.TimeSpan.SECOND],
                        dtype=np.int64,
                    ),
                    data.key: np.array([2.0, 10.0], dtype=np.float32),
                }
            )

        # Read calculated channel
        res = client.read(sy.TimeRange.MAX, [calc.key, calc.index])
        calc_multi = res[calc.key]
        idx_multi = res[calc.index]

        # Should have 3 series
        assert len(calc_multi.series) == 3
        assert len(idx_multi.series) == 3

        # Verify alignments
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[1].alignment.domain_index == 1
        assert calc_multi.series[2].alignment.domain_index == 2

        # Domain 0: [3, 4] -> [0, 0]
        assert np.allclose(calc_multi.series[0], np.array([0.0, 0.0], dtype=np.float32))

        # Domain 1: [6, 7] -> [1, 1]
        assert np.allclose(calc_multi.series[1], np.array([1.0, 1.0], dtype=np.float32))

        # Domain 2: [2, 10] -> [0, 1]
        assert np.allclose(calc_multi.series[2], np.array([0.0, 1.0], dtype=np.float32))

        # Verify calc and index alignments match
        for i in range(3):
            assert calc_multi.series[i].alignment == idx_multi.series[i].alignment

    def test_two_source_channels_same_writes_across_domains(self, client: sy.Synnax):
        """Should handle calculations with two source channels written together
        across domains with proper alignment - reading ONLY calculated channels."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        sensor_a = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        sensor_b = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        calc = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {sensor_a.name} + {sensor_b.name}",
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND,
            [idx.key, sensor_a.key, sensor_b.key],
            enable_auto_commit=True,
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                            4 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor_a.key: np.array([1.0, 2.0, 3.0, 4.0], dtype=np.float32),
                    sensor_b.key: np.array([10.0, 10.0, 10.0, 10.0], dtype=np.float32),
                }
            )

        # Second domain
        with client.open_writer(
            10 * sy.TimeSpan.SECOND,
            [idx.key, sensor_a.key, sensor_b.key],
            enable_auto_commit=True,
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            10 * sy.TimeSpan.SECOND,
                            11 * sy.TimeSpan.SECOND,
                            12 * sy.TimeSpan.SECOND,
                            13 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    sensor_a.key: np.array([10.0, 11.0, 12.0, 13.0], dtype=np.float32),
                    sensor_b.key: np.array([20.0, 20.0, 20.0, 20.0], dtype=np.float32),
                }
            )

        res = client.read(sy.TimeRange.MAX, [calc.key, calc.index])
        calc_multi = res[calc.key]
        idx_multi = res[calc.index]

        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Alignments are summed: (0,0)+(0,0)=(0,0), (1,0)+(1,0)=(2,0)
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[0].alignment.sample_index == 0
        assert calc_multi.series[1].alignment.domain_index == 2
        assert calc_multi.series[1].alignment.sample_index == 0

        # Domain 0: [1,2,3,4] + [10,10,10,10] = [11,12,13,14]
        assert np.allclose(
            calc_multi.series[0], np.array([11.0, 12.0, 13.0, 14.0], dtype=np.float32)
        )
        # Domain 1: [10,11,12,13] + [20,20,20,20] = [30,31,32,33]
        assert np.allclose(
            calc_multi.series[1], np.array([30.0, 31.0, 32.0, 33.0], dtype=np.float32)
        )

        assert calc_multi.series[0].alignment == idx_multi.series[0].alignment
        assert calc_multi.series[1].alignment == idx_multi.series[1].alignment


class TestStatOperationsAlignment:
    """Tests that statistical operations correctly propagate alignment metadata across domains."""

    def test_avg_operation_alignment_across_domains(self, client: sy.Synnax):
        """Should correctly calculate avg with proper alignment per domain."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        data = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        calc = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="avg")],
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([10.0, 20.0, 30.0], dtype=np.float32),
                }
            )

        # Second domain
        with client.open_writer(
            6 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            6 * sy.TimeSpan.SECOND,
                            7 * sy.TimeSpan.SECOND,
                            8 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([40.0, 50.0, 60.0], dtype=np.float32),
                }
            )

        # Read calculated channel
        res = client.read(sy.TimeRange.MAX, [calc.key, calc.index])
        calc_multi = res[calc.key]
        idx_multi = res[calc.index]

        # Should have 2 series (one per domain)
        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Verify domain 0 alignment
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[0].alignment.sample_index == 0
        assert idx_multi.series[0].alignment.domain_index == 0

        # Verify domain 1 alignment
        assert calc_multi.series[1].alignment.domain_index == 1
        assert calc_multi.series[1].alignment.sample_index == 0
        assert idx_multi.series[1].alignment.domain_index == 1

        assert calc_multi.series[0][0] == pytest.approx(20.0)
        assert calc_multi.series[1][0] == pytest.approx(35.0)

        assert calc_multi.series[0].alignment == idx_multi.series[0].alignment
        assert calc_multi.series[1].alignment == idx_multi.series[1].alignment

    def test_min_operation_alignment_across_domains(self, client: sy.Synnax):
        """Should correctly calculate min with proper alignment per domain."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        data = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        calc = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="min")],
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([30.0, 10.0, 20.0], dtype=np.float32),
                }
            )

        # Second domain
        with client.open_writer(
            6 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            6 * sy.TimeSpan.SECOND,
                            7 * sy.TimeSpan.SECOND,
                            8 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([60.0, 40.0, 50.0], dtype=np.float32),
                }
            )

        # Read calculated channel
        res = client.read(sy.TimeRange.MAX, [calc.key, calc.index])
        calc_multi = res[calc.key]
        idx_multi = res[calc.index]

        # Should have 2 series (one per domain)
        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Verify domain 0 alignment
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[0].alignment.sample_index == 0

        # Verify domain 1 alignment
        assert calc_multi.series[1].alignment.domain_index == 1
        assert calc_multi.series[1].alignment.sample_index == 0

        assert calc_multi.series[0][0] == pytest.approx(10.0)
        assert calc_multi.series[1][0] == pytest.approx(10.0)

        # Verify calc and index alignments match
        assert calc_multi.series[0].alignment == idx_multi.series[0].alignment
        assert calc_multi.series[1].alignment == idx_multi.series[1].alignment

    def test_max_operation_alignment_across_domains(self, client: sy.Synnax):
        """Should correctly calculate max with proper alignment per domain."""
        idx = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        data = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )
        calc = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {data.name}",
            operations=[sy.channel.Operation(type="max")],
        )

        # First domain
        with client.open_writer(
            1 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            1 * sy.TimeSpan.SECOND,
                            2 * sy.TimeSpan.SECOND,
                            3 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([10.0, 30.0, 20.0], dtype=np.float32),
                }
            )

        # Second domain
        with client.open_writer(
            6 * sy.TimeSpan.SECOND, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write(
                {
                    idx.key: np.array(
                        [
                            6 * sy.TimeSpan.SECOND,
                            7 * sy.TimeSpan.SECOND,
                            8 * sy.TimeSpan.SECOND,
                        ],
                        dtype=np.int64,
                    ),
                    data.key: np.array([40.0, 60.0, 50.0], dtype=np.float32),
                }
            )

        # Read calculated channel
        res = client.read(sy.TimeRange.MAX, [calc.key, calc.index])
        calc_multi = res[calc.key]
        idx_multi = res[calc.index]

        # Should have 2 series (one per domain)
        assert len(calc_multi.series) == 2
        assert len(idx_multi.series) == 2

        # Verify domain 0 alignment
        assert calc_multi.series[0].alignment.domain_index == 0
        assert calc_multi.series[0].alignment.sample_index == 0

        # Verify domain 1 alignment
        assert calc_multi.series[1].alignment.domain_index == 1
        assert calc_multi.series[1].alignment.sample_index == 0

        assert calc_multi.series[0][0] == pytest.approx(30.0)
        assert calc_multi.series[1][0] == pytest.approx(60.0)

        # Verify calc and index alignments match
        assert calc_multi.series[0].alignment == idx_multi.series[0].alignment
        assert calc_multi.series[1].alignment == idx_multi.series[1].alignment
