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

    @pytest.mark.focus
    def test_literal_minus_f32_should_succeed(self, client: sy.Synnax):
        """Regression: literal on left side of f32 channel should coerce to f32.

        Previously, `1000 - f32_channel` would create successfully but fail on
        read because the analyzer inferred the return type from the literal (i64)
        instead of the channel (f32). Now the literal correctly coerces.
        """
        ts = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=ts.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        calc = client.channels.create(
            name=random_name(),
            expression=f"return 1000 - {src.name}",
        )
        client.write(
            1 * sy.TimeSpan.SECOND,
            {
                ts.key: [1 * sy.TimeSpan.SECOND, 2 * sy.TimeSpan.SECOND],
                src.key: [10.0, 20.0],
            },
        )
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[calc.key, calc.index],
        )
        assert len(res[calc.key]) == 2

    @pytest.mark.focus
    def test_float_literal_div_f32_should_succeed(self, client: sy.Synnax):
        """Regression: float literal on left side of f32 channel should coerce.

        Previously, `1000.0 / f32_channel` would create successfully but fail
        on read due to return type mismatch. Now the literal coerces to f32.
        """
        ts = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=ts.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        calc = client.channels.create(
            name=random_name(),
            expression=f"return 1000.0 / {src.name}",
        )
        client.write(
            1 * sy.TimeSpan.SECOND,
            {
                ts.key: [1 * sy.TimeSpan.SECOND, 2 * sy.TimeSpan.SECOND],
                src.key: [10.0, 20.0],
            },
        )
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[calc.key, calc.index],
        )
        assert len(res[calc.key]) == 2

    @pytest.mark.focus
    def test_mixed_f32_f64_with_literal_should_reject_at_creation(
        self, client: sy.Synnax
    ):
        """Regression: mixed f32/f64 channels with literal should be caught at creation.

        Previously, `1000.0 - f32_ch + f64_ch` was accepted at creation because
        the literal's FloatConstraint individually accepted both f32 and f64.
        Now the analyzer correctly rejects the f32/f64 mismatch at creation time.
        """
        ts = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        f32_ch = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=ts.key,
                data_type=sy.DataType.FLOAT32,
            ),
        )
        f64_ch = client.channels.create(
            sy.Channel(
                name=random_name(),
                index=ts.key,
                data_type=sy.DataType.FLOAT64,
            ),
        )
        with pytest.raises(Exception, match="type mismatch"):
            client.channels.create(
                name=random_name(),
                expression=f"return 1000.0 - {f32_ch.name} + {f64_ch.name}",
            )


@pytest.mark.framer
@pytest.mark.calculations
class TestCalcChannelStress:
    """Stress test: deep calc chains, multi-domain writes, writer/streamer lifecycle
    chaos, cross-index references, and full numerical verification."""

    @pytest.mark.focus
    def test_deep_chain_multi_domain_writer_streamer_chaos(
        self, client: sy.Synnax
    ):
        S = sy.TimeSpan.SECOND

        # ── TOPOLOGY ──────────────────────────────────────────────────────
        # Two independent index channels to test cross-index behavior
        ts_a = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        ts_b = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )

        # Source channels on index A (f32)
        alpha = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts_a.key, data_type=sy.DataType.FLOAT32
            )
        )
        beta = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts_a.key, data_type=sy.DataType.FLOAT32
            )
        )

        # Source channel on index B (f32) - separate timeline
        gamma = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts_b.key, data_type=sy.DataType.FLOAT32
            )
        )

        # ── CALC CHAIN (4 levels deep) ────────────────────────────────────
        # Level 1: direct operations on source channels
        c_offset = client.channels.create(
            name=random_name(),
            expression=f"return {alpha.name} + 10",
        )
        c_lit_left = client.channels.create(
            name=random_name(),
            expression=f"return 1000 - {alpha.name}",
        )
        c_scaled = client.channels.create(
            name=random_name(),
            expression=f"return 2.5 * {alpha.name}",
        )
        c_product = client.channels.create(
            name=random_name(),
            expression=f"return {alpha.name} * {beta.name}",
        )
        c_power = client.channels.create(
            name=random_name(),
            expression=f"return {alpha.name} ^ 2",
        )
        c_temp_conv = client.channels.create(
            name=random_name(),
            expression=f"return ({alpha.name} - 32) * 5 / 9",
        )
        c_inverse = client.channels.create(
            name=random_name(),
            expression=f"return 10000.0 / {alpha.name}",
        )

        # Calc on the separate index B
        c_gamma_offset = client.channels.create(
            name=random_name(),
            expression=f"return {gamma.name} * 3 + 7",
        )

        # Level 2: calcs referencing calcs
        c_chain2 = client.channels.create(
            name=random_name(),
            expression=f"return {c_offset.name} * 3",
        )
        c_combo2 = client.channels.create(
            name=random_name(),
            expression=f"return {c_scaled.name} + {c_product.name}",
        )

        # Level 3: deeper chain
        c_deep3 = client.channels.create(
            name=random_name(),
            expression=f"return {c_chain2.name} - {c_lit_left.name}",
        )

        # Level 4: deepest - references L3 and L1
        c_mega4 = client.channels.create(
            name=random_name(),
            expression=f"return ({c_deep3.name} + {c_inverse.name}) / 10",
        )

        # Calc referencing the L2 gamma calc (on index B)
        c_gamma_chain = client.channels.create(
            name=random_name(),
            expression=f"return {c_gamma_offset.name} ^ 2",
        )

        # ── EXPECTED VALUE FUNCTIONS ──────────────────────────────────────
        def expect_a(a_vals, b_vals):
            """Return dict of calc_key -> expected numpy array for index A calcs."""
            a = np.array(a_vals, dtype=np.float32)
            b = np.array(b_vals, dtype=np.float32)
            return {
                c_offset.key: a + 10,
                c_lit_left.key: 1000 - a,
                c_scaled.key: 2.5 * a,
                c_product.key: a * b,
                c_power.key: a ** 2,
                c_temp_conv.key: (a - 32) * 5 / 9,
                c_inverse.key: 10000.0 / a,
                c_chain2.key: (a + 10) * 3,
                c_combo2.key: 2.5 * a + a * b,
                c_deep3.key: (a + 10) * 3 - (1000 - a),
                c_mega4.key: (
                    ((a + 10) * 3 - (1000 - a)) + 10000.0 / a
                ) / 10,
            }

        def expect_b(g_vals):
            """Return dict of calc_key -> expected numpy array for index B calcs."""
            g = np.array(g_vals, dtype=np.float32)
            gamma_off = g * 3 + 7
            return {
                c_gamma_offset.key: gamma_off,
                c_gamma_chain.key: gamma_off ** 2,
            }

        def verify_read(res, expected, label=""):
            for key, exp in expected.items():
                actual = np.array(res[key], dtype=np.float32)
                np.testing.assert_allclose(
                    actual,
                    np.array(exp, dtype=np.float32),
                    rtol=1e-4,
                    err_msg=f"{label} key={key}",
                )

        def verify_stream_frame(frame, expected, label=""):
            for key, exp in expected.items():
                if key not in frame.channels:
                    continue
                actual = np.array(frame[key], dtype=np.float32)
                np.testing.assert_allclose(
                    actual,
                    np.array(exp, dtype=np.float32),
                    rtol=1e-4,
                    err_msg=f"stream {label} key={key}",
                )

        all_a_calc_keys = [
            c_offset.key, c_lit_left.key, c_scaled.key, c_product.key,
            c_power.key, c_temp_conv.key, c_inverse.key,
            c_chain2.key, c_combo2.key, c_deep3.key, c_mega4.key,
        ]
        all_b_calc_keys = [c_gamma_offset.key, c_gamma_chain.key]

        # ── PHASE 1: Multi-domain writes, full read verification ──────────
        # Domain 1: 3 samples on index A
        a1, b1 = [10.0, 20.0, 50.0], [2.0, 4.0, 1.0]
        with client.open_writer(
            1 * S, [ts_a.key, alpha.key, beta.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts_a.key: [1 * S, 2 * S, 3 * S],
                alpha.key: a1,
                beta.key: b1,
            })

        # Domain 1 on index B: 2 samples (different count than A)
        g1 = [5.0, 15.0]
        with client.open_writer(
            1 * S, [ts_b.key, gamma.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts_b.key: [1 * S, 2 * S],
                gamma.key: g1,
            })

        # Verify reads after domain 1
        res_a = client.read(sy.TimeRange.MAX, all_a_calc_keys)
        verify_read(res_a, expect_a(a1, b1), "phase1-domain1-A")

        res_b = client.read(sy.TimeRange.MAX, all_b_calc_keys)
        verify_read(res_b, expect_b(g1), "phase1-domain1-B")

        # Domain 2 on index A: 4 samples (different batch size)
        a2, b2 = [100.0, 5.0, 25.0, 40.0], [3.0, 5.0, 2.0, 3.0]
        with client.open_writer(
            10 * S, [ts_a.key, alpha.key, beta.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts_a.key: [10 * S, 11 * S, 12 * S, 13 * S],
                alpha.key: a2,
                beta.key: b2,
            })

        # Domain 2 on index B: 5 samples (more than A domain 2)
        g2 = [1.0, 100.0, 50.0, 8.0, 200.0]
        with client.open_writer(
            10 * S, [ts_b.key, gamma.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts_b.key: [10 * S, 11 * S, 12 * S, 13 * S, 14 * S],
                gamma.key: g2,
            })

        # Full read - should see both domains concatenated
        all_a = a1 + a2
        all_b = b1 + b2
        all_g = g1 + g2
        res_a = client.read(sy.TimeRange.MAX, all_a_calc_keys)
        verify_read(res_a, expect_a(all_a, all_b), "phase1-all-A")
        res_b = client.read(sy.TimeRange.MAX, all_b_calc_keys)
        verify_read(res_b, expect_b(all_g), "phase1-all-B")

        # ── PHASE 2: Streaming with writer lifecycle chaos ────────────────
        # Open streamer on ALL calc channels (both indexes)
        with client.open_streamer(all_a_calc_keys) as streamer_a:
            # Write domain 3 on index A with a fresh writer
            a3, b3 = [8.0, 15.0], [4.0, 1.0]
            with client.open_writer(
                20 * S,
                [ts_a.key, alpha.key, beta.key],
                enable_auto_commit=True,
            ) as w:
                w.write({
                    ts_a.key: [20 * S, 21 * S],
                    alpha.key: a3,
                    beta.key: b3,
                })

            frame = streamer_a.read(timeout=5)
            verify_stream_frame(frame, expect_a(a3, b3), "phase2-domain3")

            # Close writer, open a NEW writer, write again
            # This tests that the streamer survives writer lifecycle changes
            a4, b4 = [200.0, 7.0, 33.0], [2.0, 8.0, 4.0]
            with client.open_writer(
                30 * S,
                [ts_a.key, alpha.key, beta.key],
                enable_auto_commit=True,
            ) as w:
                w.write({
                    ts_a.key: [30 * S, 31 * S, 32 * S],
                    alpha.key: a4,
                    beta.key: b4,
                })

            frame = streamer_a.read(timeout=5)
            verify_stream_frame(frame, expect_a(a4, b4), "phase2-domain4")

            # Rapid-fire: open writer, write 1 sample, close, repeat 3 times
            rapid_a = []
            rapid_b = []
            for i, (av, bv) in enumerate([(42.0, 6.0), (99.0, 1.0), (3.0, 9.0)]):
                t = (40 + i * 10) * S
                with client.open_writer(
                    t,
                    [ts_a.key, alpha.key, beta.key],
                    enable_auto_commit=True,
                ) as w:
                    w.write({
                        ts_a.key: [t],
                        alpha.key: [av],
                        beta.key: [bv],
                    })
                rapid_a.append(av)
                rapid_b.append(bv)

                frame = streamer_a.read(timeout=5)
                verify_stream_frame(
                    frame, expect_a([av], [bv]), f"phase2-rapid-{i}"
                )

        # ── PHASE 3: Close streamer, open new one, keep writing ───────────
        # This tests that a fresh streamer picks up new data correctly
        with client.open_streamer(all_a_calc_keys) as streamer_a2:
            a5, b5 = [11.0, 22.0], [3.0, 6.0]
            with client.open_writer(
                70 * S,
                [ts_a.key, alpha.key, beta.key],
                enable_auto_commit=True,
            ) as w:
                # Write in two separate batches within the same writer
                w.write({
                    ts_a.key: [70 * S],
                    alpha.key: [a5[0]],
                    beta.key: [b5[0]],
                })
                frame = streamer_a2.read(timeout=5)
                verify_stream_frame(
                    frame, expect_a([a5[0]], [b5[0]]), "phase3-batch1"
                )

                w.write({
                    ts_a.key: [71 * S],
                    alpha.key: [a5[1]],
                    beta.key: [b5[1]],
                })
                frame = streamer_a2.read(timeout=5)
                verify_stream_frame(
                    frame, expect_a([a5[1]], [b5[1]]), "phase3-batch2"
                )

        # ── PHASE 4: Concurrent streamers on different indexes ────────────
        with client.open_streamer(all_b_calc_keys) as streamer_b:
            with client.open_streamer(all_a_calc_keys) as streamer_a3:
                # Write to both indexes simultaneously
                a6, b6 = [55.0], [5.0]
                g3 = [77.0]
                with client.open_writer(
                    80 * S,
                    [ts_a.key, alpha.key, beta.key],
                    enable_auto_commit=True,
                ) as wa:
                    wa.write({
                        ts_a.key: [80 * S],
                        alpha.key: a6,
                        beta.key: b6,
                    })

                with client.open_writer(
                    80 * S,
                    [ts_b.key, gamma.key],
                    enable_auto_commit=True,
                ) as wb:
                    wb.write({
                        ts_b.key: [80 * S],
                        gamma.key: g3,
                    })

                frame_a = streamer_a3.read(timeout=5)
                verify_stream_frame(frame_a, expect_a(a6, b6), "phase4-A")

                frame_b = streamer_b.read(timeout=5)
                verify_stream_frame(frame_b, expect_b(g3), "phase4-B")

        # ── PHASE 5: Full read of everything, verify total counts ─────────
        # Collect all alpha/beta values written across all domains
        final_a = a1 + a2 + a3 + a4 + rapid_a + a5 + a6
        final_b = b1 + b2 + b3 + b4 + rapid_b + b5 + b6
        final_g = g1 + g2 + g3

        res_a = client.read(sy.TimeRange.MAX, all_a_calc_keys)
        verify_read(res_a, expect_a(final_a, final_b), "phase5-final-A")
        for key in all_a_calc_keys:
            assert len(res_a[key]) == len(final_a), (
                f"Expected {len(final_a)} samples for calc {key}, "
                f"got {len(res_a[key])}"
            )

        res_b = client.read(sy.TimeRange.MAX, all_b_calc_keys)
        verify_read(res_b, expect_b(final_g), "phase5-final-B")
        for key in all_b_calc_keys:
            assert len(res_b[key]) == len(final_g), (
                f"Expected {len(final_g)} samples for calc {key}, "
                f"got {len(res_b[key])}"
            )

    @pytest.mark.focus
    def test_writer_abort_doesnt_corrupt_calcs(self, client: sy.Synnax):
        """Write domain 1 committed, domain 2 uncommitted (writer closes without
        commit), then verify only domain 1 data is visible through calcs."""
        S = sy.TimeSpan.SECOND
        ts = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        calc = client.channels.create(
            name=random_name(), expression=f"return {src.name} * 2 + 1"
        )

        # Domain 1: committed
        with client.open_writer(
            1 * S, [ts.key, src.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts.key: [1 * S, 2 * S, 3 * S],
                src.key: [10.0, 20.0, 30.0],
            })

        # Domain 2: NOT committed (no auto_commit, no manual commit)
        with client.open_writer(
            10 * S, [ts.key, src.key]
        ) as w:
            w.write({
                ts.key: [10 * S, 11 * S],
                src.key: [999.0, 888.0],
            })
            # Writer closes here without commit - data should be discarded

        # Only domain 1 should be visible
        res = client.read(sy.TimeRange.MAX, [calc.key])
        expected = np.array([10.0, 20.0, 30.0], dtype=np.float32) * 2 + 1
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32), expected, rtol=1e-5
        )
        assert len(res[calc.key]) == 3

    @pytest.mark.focus
    def test_partial_time_range_reads(self, client: sy.Synnax):
        """Write 3 domains, then read partial time ranges that span boundaries."""
        S = sy.TimeSpan.SECOND
        ts = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        calc = client.channels.create(
            name=random_name(), expression=f"return {src.name} + 100"
        )

        # Domain 1: timestamps 1-3s, values 1,2,3
        with client.open_writer(
            1 * S, [ts.key, src.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts.key: [1 * S, 2 * S, 3 * S],
                src.key: [1.0, 2.0, 3.0],
            })

        # Domain 2: timestamps 10-12s, values 10,20,30
        with client.open_writer(
            10 * S, [ts.key, src.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts.key: [10 * S, 11 * S, 12 * S],
                src.key: [10.0, 20.0, 30.0],
            })

        # Domain 3: timestamps 20-22s, values 100,200,300
        with client.open_writer(
            20 * S, [ts.key, src.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts.key: [20 * S, 21 * S, 22 * S],
                src.key: [100.0, 200.0, 300.0],
            })

        # Read only domain 1
        res = client.read(
            sy.TimeRange(1 * S, 4 * S), [calc.key]
        )
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32),
            np.array([101.0, 102.0, 103.0], dtype=np.float32),
            rtol=1e-5,
        )

        # Read spanning domain 1 and 2 (gap in between)
        res = client.read(
            sy.TimeRange(2 * S, 11 * S), [calc.key]
        )
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32),
            np.array([102.0, 103.0, 110.0], dtype=np.float32),
            rtol=1e-5,
        )

        # Read only middle of domain 2
        res = client.read(
            sy.TimeRange(11 * S, 12 * S), [calc.key]
        )
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32),
            np.array([120.0], dtype=np.float32),
            rtol=1e-5,
        )

        # Read spanning all 3 domains
        res = client.read(
            sy.TimeRange(1 * S, 23 * S), [calc.key]
        )
        expected = np.array(
            [1, 2, 3, 10, 20, 30, 100, 200, 300], dtype=np.float32
        ) + 100
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32), expected, rtol=1e-5
        )

    @pytest.mark.focus
    def test_timestamp_alignment_across_domains(self, client: sy.Synnax):
        """Verify that calc channel timestamps exactly match source timestamps
        across multiple domains, and alignment metadata is correct."""
        S = sy.TimeSpan.SECOND
        ts = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        calc = client.channels.create(
            name=random_name(), expression=f"return {src.name} * 3"
        )

        domain_timestamps = [
            [1 * S, 2 * S],
            [10 * S, 11 * S, 12 * S],
            [20 * S],
        ]
        domain_values = [
            [5.0, 10.0],
            [15.0, 20.0, 25.0],
            [30.0],
        ]

        for ts_data, src_data in zip(domain_timestamps, domain_values):
            with client.open_writer(
                ts_data[0], [ts.key, src.key], enable_auto_commit=True
            ) as w:
                w.write({ts.key: ts_data, src.key: src_data})

        res = client.read(sy.TimeRange.MAX, [calc.key, calc.index])
        calc_multi = res[calc.key]
        idx_multi = res[calc.index]

        # Should have 3 series (one per domain)
        assert len(calc_multi.series) == 3
        assert len(idx_multi.series) == 3

        # Verify each domain's alignment matches
        for i in range(3):
            assert calc_multi.series[i].alignment == idx_multi.series[i].alignment
            assert calc_multi.series[i].alignment.domain_index == i
            assert calc_multi.series[i].alignment.sample_index == 0

        # Verify timestamps match exactly
        all_ts = []
        for ts_data in domain_timestamps:
            all_ts.extend(ts_data)
        np.testing.assert_array_equal(
            np.array(idx_multi, dtype=np.int64),
            np.array(all_ts, dtype=np.int64),
        )

        # Verify values
        all_vals = []
        for v in domain_values:
            all_vals.extend(v)
        expected = np.array(all_vals, dtype=np.float32) * 3
        np.testing.assert_allclose(
            np.array(calc_multi, dtype=np.float32), expected, rtol=1e-5
        )

    @pytest.mark.focus
    def test_many_rapid_writer_cycles_with_streamer(self, client: sy.Synnax):
        """Open a streamer, then rapidly cycle through 15 writers, each writing
        1-3 samples. Verify every streamed frame is correct."""
        S = sy.TimeSpan.SECOND
        ts = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        calc = client.channels.create(
            name=random_name(), expression=f"return 500 - {src.name} * 2"
        )

        batches = [
            [1.0],
            [2.0, 3.0],
            [4.0, 5.0, 6.0],
            [7.0],
            [8.0, 9.0],
            [10.0],
            [11.0, 12.0, 13.0],
            [14.0],
            [15.0, 16.0],
            [17.0, 18.0, 19.0],
            [20.0],
            [21.0],
            [22.0, 23.0],
            [24.0, 25.0, 26.0],
            [27.0],
        ]

        all_values = []
        t_cursor = 1

        with client.open_streamer(calc.key) as streamer:
            for batch in batches:
                ts_data = [(t_cursor + j) * S for j in range(len(batch))]
                t_cursor += len(batch) + 5  # gap between domains

                with client.open_writer(
                    ts_data[0],
                    [ts.key, src.key],
                    enable_auto_commit=True,
                ) as w:
                    w.write({
                        ts.key: ts_data,
                        src.key: batch,
                    })

                frame = streamer.read(timeout=5)
                expected = 500 - np.array(batch, dtype=np.float32) * 2
                np.testing.assert_allclose(
                    np.array(frame[calc.key], dtype=np.float32),
                    expected,
                    rtol=1e-5,
                    err_msg=f"batch={batch}",
                )
                all_values.extend(batch)

        # Final full read
        res = client.read(sy.TimeRange.MAX, [calc.key])
        expected_all = 500 - np.array(all_values, dtype=np.float32) * 2
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32),
            expected_all,
            rtol=1e-5,
        )
        assert len(res[calc.key]) == len(all_values)

    @pytest.mark.focus
    def test_writes_after_streamer_close_then_new_streamer(
        self, client: sy.Synnax
    ):
        """Verify data written while no streamer is open is still readable,
        and a new streamer picks up subsequent writes correctly."""
        S = sy.TimeSpan.SECOND
        ts = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        calc = client.channels.create(
            name=random_name(), expression=f"return {src.name} ^ 2 + 1"
        )

        # Phase 1: streamer open, write, verify, close streamer
        with client.open_streamer(calc.key) as s1:
            with client.open_writer(
                1 * S, [ts.key, src.key], enable_auto_commit=True
            ) as w:
                w.write({
                    ts.key: [1 * S, 2 * S],
                    src.key: [3.0, 4.0],
                })
            frame = s1.read(timeout=5)
            expected = np.array([3.0, 4.0], dtype=np.float32) ** 2 + 1
            np.testing.assert_allclose(
                np.array(frame[calc.key], dtype=np.float32), expected, rtol=1e-5
            )

        # Phase 2: NO streamer open, write two more domains
        with client.open_writer(
            10 * S, [ts.key, src.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts.key: [10 * S, 11 * S, 12 * S],
                src.key: [5.0, 6.0, 7.0],
            })

        with client.open_writer(
            20 * S, [ts.key, src.key], enable_auto_commit=True
        ) as w:
            w.write({
                ts.key: [20 * S],
                src.key: [8.0],
            })

        # Verify all data is readable via client.read (no streamer needed)
        res = client.read(sy.TimeRange.MAX, [calc.key])
        all_src = np.array([3, 4, 5, 6, 7, 8], dtype=np.float32)
        expected_all = all_src ** 2 + 1
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32), expected_all, rtol=1e-5
        )

        # Phase 3: open NEW streamer, write more, verify stream works
        with client.open_streamer(calc.key) as s2:
            with client.open_writer(
                30 * S, [ts.key, src.key], enable_auto_commit=True
            ) as w:
                w.write({
                    ts.key: [30 * S, 31 * S],
                    src.key: [9.0, 10.0],
                })
            frame = s2.read(timeout=5)
            expected = np.array([9.0, 10.0], dtype=np.float32) ** 2 + 1
            np.testing.assert_allclose(
                np.array(frame[calc.key], dtype=np.float32), expected, rtol=1e-5
            )

        # Final verification - everything
        res = client.read(sy.TimeRange.MAX, [calc.key])
        all_src = np.array([3, 4, 5, 6, 7, 8, 9, 10], dtype=np.float32)
        expected_final = all_src ** 2 + 1
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32),
            expected_final,
            rtol=1e-5,
        )

    @pytest.mark.focus
    def test_interleaved_streamer_lifecycle(self, client: sy.Synnax):
        """Open and close multiple streamers in overlapping patterns while
        continuously writing through different writers."""
        S = sy.TimeSpan.SECOND
        ts = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        src_a = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        src_b = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        calc_sum = client.channels.create(
            name=random_name(),
            expression=f"return {src_a.name} + {src_b.name}",
        )
        calc_diff = client.channels.create(
            name=random_name(),
            expression=f"return {src_a.name} - {src_b.name}",
        )
        calc_chain = client.channels.create(
            name=random_name(),
            expression=f"return {calc_sum.name} * {calc_diff.name}",
        )

        def write_batch(start_t, a_vals, b_vals):
            ts_data = [(start_t + i) * S for i in range(len(a_vals))]
            with client.open_writer(
                ts_data[0],
                [ts.key, src_a.key, src_b.key],
                enable_auto_commit=True,
            ) as w:
                w.write({
                    ts.key: ts_data,
                    src_a.key: a_vals,
                    src_b.key: b_vals,
                })

        def expect(a_vals, b_vals):
            a = np.array(a_vals, dtype=np.float32)
            b = np.array(b_vals, dtype=np.float32)
            return {
                calc_sum.key: a + b,
                calc_diff.key: a - b,
                calc_chain.key: (a + b) * (a - b),
            }

        def verify_frame(frame, expected, label):
            for key, exp in expected.items():
                if key not in frame.channels:
                    continue
                np.testing.assert_allclose(
                    np.array(frame[key], dtype=np.float32),
                    exp,
                    rtol=1e-4,
                    err_msg=label,
                )

        # Streamer 1: on calc_sum only
        s1 = client.open_streamer([calc_sum.key])

        write_batch(1, [10.0, 20.0], [3.0, 7.0])
        frame = s1.read(timeout=5)
        verify_frame(frame, expect([10.0, 20.0], [3.0, 7.0]), "s1-batch1")

        # Streamer 2: on calc_diff and calc_chain (overlapping with s1)
        s2 = client.open_streamer([calc_diff.key, calc_chain.key])

        write_batch(10, [50.0], [20.0])
        frame1 = s1.read(timeout=5)
        frame2 = s2.read(timeout=5)
        verify_frame(frame1, expect([50.0], [20.0]), "s1-batch2")
        verify_frame(frame2, expect([50.0], [20.0]), "s2-batch2")

        # Close streamer 1, keep streamer 2
        s1.close()

        write_batch(20, [100.0, 200.0, 300.0], [1.0, 2.0, 3.0])
        frame2 = s2.read(timeout=5)
        verify_frame(
            frame2,
            expect([100.0, 200.0, 300.0], [1.0, 2.0, 3.0]),
            "s2-batch3-after-s1-close",
        )

        # Open streamer 3: on ALL calc channels (overlapping with s2)
        s3 = client.open_streamer(
            [calc_sum.key, calc_diff.key, calc_chain.key]
        )

        write_batch(30, [7.0, 8.0], [4.0, 5.0])
        frame2 = s2.read(timeout=5)
        frame3 = s3.read(timeout=5)
        verify_frame(frame2, expect([7.0, 8.0], [4.0, 5.0]), "s2-batch4")
        verify_frame(frame3, expect([7.0, 8.0], [4.0, 5.0]), "s3-batch4")

        # Close streamer 2, keep streamer 3
        s2.close()

        write_batch(40, [1000.0], [1.0])
        frame3 = s3.read(timeout=5)
        verify_frame(frame3, expect([1000.0], [1.0]), "s3-batch5-after-s2-close")

        s3.close()

        # Final read - verify all data across all domains
        all_a = [10, 20, 50, 100, 200, 300, 7, 8, 1000]
        all_b = [3, 7, 20, 1, 2, 3, 4, 5, 1]
        res = client.read(
            sy.TimeRange.MAX,
            [calc_sum.key, calc_diff.key, calc_chain.key],
        )
        exp = expect(all_a, all_b)
        for key in exp:
            np.testing.assert_allclose(
                np.array(res[key], dtype=np.float32),
                exp[key],
                rtol=1e-4,
                err_msg=f"final-read key={key}",
            )
            assert len(res[key]) == len(all_a)

    @pytest.mark.focus
    def test_multiple_write_batches_single_writer_with_streamer(
        self, client: sy.Synnax
    ):
        """Single writer sends many small batches (1-2 samples each) while a
        streamer verifies each batch. Tests that calc pipeline handles high-
        frequency small writes within a single domain."""
        S = sy.TimeSpan.SECOND
        ts = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        c1 = client.channels.create(
            name=random_name(), expression=f"return {src.name} + 1"
        )
        c2 = client.channels.create(
            name=random_name(), expression=f"return {c1.name} * 2"
        )
        c3 = client.channels.create(
            name=random_name(), expression=f"return {c2.name} - 100"
        )

        all_src = []
        with client.open_streamer([c1.key, c2.key, c3.key]) as streamer:
            with client.open_writer(
                1 * S,
                [ts.key, src.key],
                enable_auto_commit=True,
            ) as writer:
                for i in range(20):
                    val = float((i + 1) * 5)
                    writer.write({
                        ts.key: [(1 + i) * S],
                        src.key: [val],
                    })
                    all_src.append(val)

                    frame = streamer.read(timeout=5)
                    s = np.float32(val)
                    e1 = s + 1
                    e2 = e1 * 2
                    e3 = e2 - 100
                    if c1.key in frame.channels:
                        np.testing.assert_allclose(
                            np.array(frame[c1.key], dtype=np.float32),
                            [e1],
                            rtol=1e-5,
                            err_msg=f"c1 batch {i}",
                        )
                    if c3.key in frame.channels:
                        np.testing.assert_allclose(
                            np.array(frame[c3.key], dtype=np.float32),
                            [e3],
                            rtol=1e-5,
                            err_msg=f"c3 batch {i}",
                        )

        # Full read verification
        res = client.read(sy.TimeRange.MAX, [c1.key, c2.key, c3.key])
        src_arr = np.array(all_src, dtype=np.float32)
        np.testing.assert_allclose(
            np.array(res[c1.key], dtype=np.float32), src_arr + 1, rtol=1e-5
        )
        np.testing.assert_allclose(
            np.array(res[c2.key], dtype=np.float32),
            (src_arr + 1) * 2,
            rtol=1e-5,
        )
        np.testing.assert_allclose(
            np.array(res[c3.key], dtype=np.float32),
            (src_arr + 1) * 2 - 100,
            rtol=1e-5,
        )
        assert len(res[c3.key]) == 20

    @pytest.mark.focus
    def test_calc_with_varying_domain_sizes(self, client: sy.Synnax):
        """Write domains of wildly different sizes (1, 100, 2, 50, 1) and verify
        calc correctness across all of them."""
        S = sy.TimeSpan.SECOND
        ts = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        calc = client.channels.create(
            name=random_name(),
            expression=f"return 2.5 * {src.name} - 7",
        )

        domain_sizes = [1, 100, 2, 50, 1]
        all_vals = []
        t_cursor = 1

        for size in domain_sizes:
            vals = [float(t_cursor + j) for j in range(size)]
            ts_data = [(t_cursor + j) * S for j in range(size)]
            with client.open_writer(
                ts_data[0], [ts.key, src.key], enable_auto_commit=True
            ) as w:
                w.write({ts.key: ts_data, src.key: vals})
            all_vals.extend(vals)
            t_cursor += size + 10  # gap

        res = client.read(sy.TimeRange.MAX, [calc.key])
        expected = np.array(all_vals, dtype=np.float32) * 2.5 - 7
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32), expected, rtol=1e-4
        )
        assert len(res[calc.key]) == sum(domain_sizes)

    @pytest.mark.focus
    def test_streamer_survives_writer_error_recovery(self, client: sy.Synnax):
        """Open a streamer, write good data, then open a writer that aborts
        (uncommitted), then open another writer with good data. Streamer should
        still receive correct values from the good writers."""
        S = sy.TimeSpan.SECOND
        ts = client.channels.create(
            name=random_name(), is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        src = client.channels.create(
            sy.Channel(
                name=random_name(), index=ts.key, data_type=sy.DataType.FLOAT32
            )
        )
        calc = client.channels.create(
            name=random_name(), expression=f"return {src.name} * 10"
        )

        with client.open_streamer(calc.key) as streamer:
            # Good write 1
            with client.open_writer(
                1 * S, [ts.key, src.key], enable_auto_commit=True
            ) as w:
                w.write({
                    ts.key: [1 * S, 2 * S],
                    src.key: [5.0, 6.0],
                })
            frame = streamer.read(timeout=5)
            np.testing.assert_allclose(
                np.array(frame[calc.key], dtype=np.float32),
                [50.0, 60.0],
                rtol=1e-5,
            )

            # Aborted write (no auto_commit, no manual commit)
            with client.open_writer(
                10 * S, [ts.key, src.key]
            ) as w:
                w.write({
                    ts.key: [10 * S],
                    src.key: [999.0],
                })
                # Close without commit

            # Good write 2 - streamer should still work
            with client.open_writer(
                20 * S, [ts.key, src.key], enable_auto_commit=True
            ) as w:
                w.write({
                    ts.key: [20 * S, 21 * S],
                    src.key: [7.0, 8.0],
                })
            frame = streamer.read(timeout=5)
            np.testing.assert_allclose(
                np.array(frame[calc.key], dtype=np.float32),
                [70.0, 80.0],
                rtol=1e-5,
            )

        # Final read: only committed data should be visible
        res = client.read(sy.TimeRange.MAX, [calc.key])
        np.testing.assert_allclose(
            np.array(res[calc.key], dtype=np.float32),
            [50.0, 60.0, 70.0, 80.0],
            rtol=1e-5,
        )
        assert len(res[calc.key]) == 4


@pytest.mark.framer
@pytest.mark.calculations
class TestCalcChannelEdgeCases:
    """Exhaustive edge case tests for calculated channel type inference and execution."""

    @pytest.fixture()
    def setup(self, client: sy.Synnax):
        """Create a shared set of source channels across multiple data types."""
        ts = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        channels = {}
        for dt in [
            sy.DataType.FLOAT32,
            sy.DataType.FLOAT64,
            sy.DataType.INT32,
            sy.DataType.INT64,
            sy.DataType.INT8,
            sy.DataType.UINT8,
        ]:
            channels[dt] = client.channels.create(
                sy.Channel(
                    name=random_name(),
                    index=ts.key,
                    data_type=dt,
                ),
            )
        client.write(
            1 * sy.TimeSpan.SECOND,
            {
                ts.key: [
                    1 * sy.TimeSpan.SECOND,
                    2 * sy.TimeSpan.SECOND,
                    3 * sy.TimeSpan.SECOND,
                ],
                channels[sy.DataType.FLOAT32].key: [10.0, 20.0, 30.0],
                channels[sy.DataType.FLOAT64].key: [10.0, 20.0, 30.0],
                channels[sy.DataType.INT32].key: [10, 20, 30],
                channels[sy.DataType.INT64].key: [10, 20, 30],
                channels[sy.DataType.INT8].key: [10, 20, 30],
                channels[sy.DataType.UINT8].key: [10, 20, 30],
            },
        )
        return client, ts, channels

    def _create_and_read(self, client, expression):
        calc = client.channels.create(
            name=random_name(),
            expression=expression,
        )
        res = client.read(
            tr=sy.TimeRange.MAX,
            channels=[calc.key, calc.index],
        )
        assert len(res[calc.key]) == 3
        return res[calc.key]

    def _expect_create_fail(self, client, expression):
        with pytest.raises(Exception):
            client.channels.create(
                name=random_name(),
                expression=expression,
            )

    # --- Literal on left: all operators x all types ---

    @pytest.mark.focus
    def test_int_literal_plus_f32(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 5 + {ch[sy.DataType.FLOAT32].name}")

    @pytest.mark.focus
    def test_int_literal_minus_f64(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 100 - {ch[sy.DataType.FLOAT64].name}")

    @pytest.mark.focus
    def test_int_literal_times_i32(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 3 * {ch[sy.DataType.INT32].name}")

    @pytest.mark.focus
    def test_int_literal_div_f32(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 100 / {ch[sy.DataType.FLOAT32].name}")

    @pytest.mark.focus
    def test_float_literal_plus_f64(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 2.5 + {ch[sy.DataType.FLOAT64].name}")

    @pytest.mark.focus
    def test_float_literal_minus_f32(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 99.9 - {ch[sy.DataType.FLOAT32].name}")

    @pytest.mark.focus
    def test_float_literal_times_f32(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 3.14 * {ch[sy.DataType.FLOAT32].name}")

    @pytest.mark.focus
    def test_float_literal_div_f64(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 1.0 / {ch[sy.DataType.FLOAT64].name}")

    # --- Literal on left with integer channels ---

    @pytest.mark.focus
    def test_int_literal_minus_i64(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 1000 - {ch[sy.DataType.INT64].name}")

    @pytest.mark.focus
    def test_int_literal_div_i32(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 100 / {ch[sy.DataType.INT32].name}")

    @pytest.mark.focus
    def test_int_literal_minus_i8(self, setup):
        client, _, ch = setup
        self._create_and_read(client, f"return 50 - {ch[sy.DataType.INT8].name}")

    # --- Float literal with integer channel should reject ---

    @pytest.mark.focus
    def test_float_literal_plus_i32_rejects(self, setup):
        client, _, ch = setup
        self._expect_create_fail(
            client, f"return 2.5 + {ch[sy.DataType.INT32].name}"
        )

    @pytest.mark.focus
    def test_float_literal_minus_i64_rejects(self, setup):
        client, _, ch = setup
        self._expect_create_fail(
            client, f"return 3.7 - {ch[sy.DataType.INT64].name}"
        )

    # --- Multiple literals, channel in middle ---

    @pytest.mark.focus
    def test_literal_channel_literal_add(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return 5 + {ch[sy.DataType.FLOAT32].name} + 10"
        )

    @pytest.mark.focus
    def test_literal_channel_literal_mixed_ops(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return 2 * {ch[sy.DataType.FLOAT64].name} - 5"
        )

    @pytest.mark.focus
    def test_literal_channel_literal_div_both_sides(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return 100.0 / {ch[sy.DataType.FLOAT32].name} / 2.0"
        )

    # --- Channel used multiple times ---

    @pytest.mark.focus
    def test_channel_squared(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} * {f32}")

    @pytest.mark.focus
    def test_channel_minus_itself(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} - {f32}")

    @pytest.mark.focus
    def test_channel_div_itself(self, setup):
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        self._create_and_read(client, f"return {f64} / {f64}")

    @pytest.mark.focus
    def test_channel_used_three_times(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} * {f32} + {f32}")

    # --- Deeply nested parentheses ---

    @pytest.mark.focus
    def test_deeply_nested_parens(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return (((({f32}))))")

    @pytest.mark.focus
    def test_nested_binary_in_parens(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return ((({f32} + 1)) * 2)")

    @pytest.mark.focus
    def test_nested_parens_literal_left(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return (100 - ({f32} * 2))")

    # --- Power expressions ---

    @pytest.mark.focus
    def test_f32_power_int_literal(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return {ch[sy.DataType.FLOAT32].name} ^ 2"
        )

    @pytest.mark.focus
    def test_f64_power_int_literal(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return {ch[sy.DataType.FLOAT64].name} ^ 3"
        )

    @pytest.mark.focus
    def test_f32_power_float_literal(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return {ch[sy.DataType.FLOAT32].name} ^ 2.5"
        )

    @pytest.mark.focus
    def test_i32_power_int_literal(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return {ch[sy.DataType.INT32].name} ^ 2"
        )

    @pytest.mark.focus
    def test_power_in_complex_expression(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return 2 * {f32} ^ 2 + 1")

    @pytest.mark.focus
    def test_literal_left_times_power(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return 3.14 * {f32} ^ 2")

    # --- Unary negation ---

    @pytest.mark.focus
    def test_negate_channel(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return -{ch[sy.DataType.FLOAT32].name}"
        )

    @pytest.mark.focus
    def test_negate_expression(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return -({f32} + 10)")

    @pytest.mark.focus
    def test_negate_literal_left(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return -(100 - {f32})")

    @pytest.mark.focus
    def test_double_negate(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return -(-{ch[sy.DataType.FLOAT64].name})"
        )

    # --- Chained same operator ---

    @pytest.mark.focus
    def test_chained_addition_three_channels(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} + {f32} + {f32}")

    @pytest.mark.focus
    def test_chained_subtraction(self, setup):
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        self._create_and_read(client, f"return 1000 - {f64} - {f64}")

    @pytest.mark.focus
    def test_chained_multiplication(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} * {f32} * {f32}")

    @pytest.mark.focus
    def test_chained_division(self, setup):
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        self._create_and_read(client, f"return 10000 / {f64} / {f64}")

    # --- Mixed operators (precedence) ---

    @pytest.mark.focus
    def test_add_then_multiply(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} + {f32} * 2")

    @pytest.mark.focus
    def test_multiply_then_add(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} * 2 + {f32}")

    @pytest.mark.focus
    def test_sub_then_div(self, setup):
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        self._create_and_read(client, f"return {f64} - {f64} / 2")

    @pytest.mark.focus
    def test_parens_override_precedence(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return ({f32} + {f32}) * 2")

    @pytest.mark.focus
    def test_literal_left_with_precedence(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return 100 - {f32} * 2")

    @pytest.mark.focus
    def test_literal_left_parens_override(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return (100 - {f32}) * 2")

    # --- Customer-like patterns (3-channel arithmetic with division by literal) ---

    @pytest.mark.focus
    def test_three_f32_channels_mul_div_literal(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(
            client, f"return ({f32} * {f32} * {f32}) / 1000.0"
        )

    @pytest.mark.focus
    def test_three_f32_channels_nested_parens(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(
            client, f"return (({f32} * {f32}) * {f32}) / 1000.0"
        )

    @pytest.mark.focus
    def test_three_f32_channels_right_nested(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(
            client, f"return ({f32} * ({f32} * {f32})) / 1000.0"
        )

    @pytest.mark.focus
    def test_three_f64_channels_mul_div_int_literal(self, setup):
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        self._create_and_read(
            client, f"return ({f64} * {f64} * {f64}) / 1000"
        )

    @pytest.mark.focus
    def test_three_i32_channels_mul_div_literal(self, setup):
        client, _, ch = setup
        i32 = ch[sy.DataType.INT32].name
        self._create_and_read(
            client, f"return ({i32} * {i32} * {i32}) / 100"
        )

    # --- Literal-only subexpressions ---

    @pytest.mark.focus
    def test_literal_subexpr_plus_channel(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return (2 + 3) * {f32}")

    @pytest.mark.focus
    def test_literal_subexpr_div_channel(self, setup):
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        self._create_and_read(client, f"return (100.0 / 5.0) + {f64}")

    @pytest.mark.focus
    def test_channel_div_literal_subexpr(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} / (2 * 5)")

    # --- Interleaved channels and literals ---

    @pytest.mark.focus
    def test_channel_lit_channel_lit(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} * 2 + {f32} * 3")

    @pytest.mark.focus
    def test_lit_channel_lit_channel(self, setup):
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        self._create_and_read(client, f"return 10 + {f64} + 20 + {f64}")

    @pytest.mark.focus
    def test_alternating_ops(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} + 1 - {f32} + 2")

    # --- Type coercion edge cases ---

    @pytest.mark.focus
    def test_exact_integer_float_with_i32(self, setup):
        """5.0 is an exact integer float, should coerce to i32."""
        client, _, ch = setup
        self._create_and_read(
            client, f"return {ch[sy.DataType.INT32].name} + 5.0"
        )

    @pytest.mark.focus
    def test_exact_integer_float_literal_left_i32(self, setup):
        """5.0 on left side with i32 channel."""
        client, _, ch = setup
        self._create_and_read(
            client, f"return 5.0 + {ch[sy.DataType.INT32].name}"
        )

    @pytest.mark.focus
    def test_non_exact_float_with_i32_rejects(self, setup):
        """5.5 is not an exact integer, should reject with i32."""
        client, _, ch = setup
        self._expect_create_fail(
            client, f"return {ch[sy.DataType.INT32].name} + 5.5"
        )

    @pytest.mark.focus
    def test_zero_literal_plus_f32(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return 0 + {ch[sy.DataType.FLOAT32].name}"
        )

    @pytest.mark.focus
    def test_one_literal_times_f32(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return 1 * {ch[sy.DataType.FLOAT32].name}"
        )

    @pytest.mark.focus
    def test_zero_point_zero_plus_f64(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client, f"return 0.0 + {ch[sy.DataType.FLOAT64].name}"
        )

    # --- Mixed type rejection (should all fail at creation) ---

    @pytest.mark.focus
    def test_f32_plus_f64_rejects(self, setup):
        client, _, ch = setup
        self._expect_create_fail(
            client,
            f"return {ch[sy.DataType.FLOAT32].name} + {ch[sy.DataType.FLOAT64].name}",
        )

    @pytest.mark.focus
    def test_i32_plus_f32_rejects(self, setup):
        client, _, ch = setup
        self._expect_create_fail(
            client,
            f"return {ch[sy.DataType.INT32].name} + {ch[sy.DataType.FLOAT32].name}",
        )

    @pytest.mark.focus
    def test_i32_plus_i64_rejects(self, setup):
        client, _, ch = setup
        self._expect_create_fail(
            client,
            f"return {ch[sy.DataType.INT32].name} + {ch[sy.DataType.INT64].name}",
        )

    @pytest.mark.focus
    def test_literal_masks_mixed_types_addition(self, setup):
        """Literal on left with f32 then f64 in addition. Should reject."""
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        f64 = ch[sy.DataType.FLOAT64].name
        self._expect_create_fail(client, f"return 1 + {f32} + {f64}")

    @pytest.mark.focus
    def test_literal_masks_mixed_types_multiplication(self, setup):
        """Literal on left with f32 then f64 in multiplication. Should reject."""
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        f64 = ch[sy.DataType.FLOAT64].name
        self._expect_create_fail(client, f"return 2 * {f32} * {f64}")

    @pytest.mark.focus
    def test_literal_masks_mixed_types_subtraction(self, setup):
        """Literal on left with i32 then i64 in subtraction. Should reject."""
        client, _, ch = setup
        i32 = ch[sy.DataType.INT32].name
        i64 = ch[sy.DataType.INT64].name
        self._expect_create_fail(client, f"return 100 - {i32} - {i64}")

    @pytest.mark.focus
    def test_three_different_types_with_literal(self, setup):
        """All different types. Should reject."""
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        f64 = ch[sy.DataType.FLOAT64].name
        i32 = ch[sy.DataType.INT32].name
        self._expect_create_fail(
            client, f"return 1 + {f32} + {f64} + {i32}"
        )

    # --- Nested calc channels ---

    @pytest.mark.focus
    def test_calc_referencing_calc(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        inner = client.channels.create(
            name=random_name(),
            expression=f"return {f32} * 2",
        )
        self._create_and_read(client, f"return {inner.name} + 10")

    @pytest.mark.focus
    def test_calc_referencing_calc_literal_left(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        inner = client.channels.create(
            name=random_name(),
            expression=f"return {f32} + 1",
        )
        self._create_and_read(client, f"return 100 - {inner.name}")

    @pytest.mark.focus
    def test_calc_plus_source_channel(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        inner = client.channels.create(
            name=random_name(),
            expression=f"return {f32} * 3",
        )
        self._create_and_read(client, f"return {inner.name} + {f32}")

    @pytest.mark.focus
    def test_calc_of_calc_of_calc(self, setup):
        """Three levels of nesting."""
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        lvl1 = client.channels.create(
            name=random_name(), expression=f"return {f64} + 1"
        )
        lvl2 = client.channels.create(
            name=random_name(), expression=f"return {lvl1.name} * 2"
        )
        self._create_and_read(client, f"return {lvl2.name} - 5")

    # --- Large/small literal values ---

    @pytest.mark.focus
    def test_very_large_int_literal(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client,
            f"return {ch[sy.DataType.FLOAT64].name} + 999999999",
        )

    @pytest.mark.focus
    def test_very_small_float_literal(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client,
            f"return {ch[sy.DataType.FLOAT64].name} * 0.0001",
        )

    @pytest.mark.focus
    def test_negative_literal(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client,
            f"return {ch[sy.DataType.FLOAT32].name} + -5",
        )

    @pytest.mark.focus
    def test_large_literal_left_div(self, setup):
        client, _, ch = setup
        self._create_and_read(
            client,
            f"return 1000000 / {ch[sy.DataType.FLOAT32].name}",
        )

    # --- Expressions with only same-type channels (no literals) ---

    @pytest.mark.focus
    def test_two_f32_add(self, setup):
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} + {f32}")

    @pytest.mark.focus
    def test_two_i32_multiply(self, setup):
        client, _, ch = setup
        i32 = ch[sy.DataType.INT32].name
        self._create_and_read(client, f"return {i32} * {i32}")

    @pytest.mark.focus
    def test_two_f64_subtract(self, setup):
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        self._create_and_read(client, f"return {f64} - {f64}")

    # --- Complex real-world patterns ---

    @pytest.mark.focus
    def test_temperature_conversion_pattern(self, setup):
        """F to C: (temp - 32) * 5 / 9"""
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return ({f32} - 32) * 5 / 9")

    @pytest.mark.focus
    def test_percentage_pattern(self, setup):
        """(value / total) * 100"""
        client, _, ch = setup
        f64 = ch[sy.DataType.FLOAT64].name
        self._create_and_read(client, f"return ({f64} / 1000) * 100")

    @pytest.mark.focus
    def test_moving_average_like_pattern(self, setup):
        """(a + a + a) / 3"""
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return ({f32} + {f32} + {f32}) / 3")

    @pytest.mark.focus
    def test_polynomial_pattern(self, setup):
        """a^2 + 2*a + 1"""
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return {f32} ^ 2 + 2 * {f32} + 1")

    @pytest.mark.focus
    def test_inverse_pattern(self, setup):
        """1 / channel"""
        client, _, ch = setup
        self._create_and_read(
            client, f"return 1 / {ch[sy.DataType.FLOAT64].name}"
        )

    @pytest.mark.focus
    def test_scaling_offset_pattern(self, setup):
        """mx + b: 2.5 * channel + 100"""
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(client, f"return 2.5 * {f32} + 100")

    @pytest.mark.focus
    def test_difference_over_sum(self, setup):
        """(a - b) / (a + b) with same channel"""
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(
            client, f"return ({f32} - 5) / ({f32} + 5)"
        )

    @pytest.mark.focus
    def test_literal_sandwich(self, setup):
        """lit op channel op lit op channel op lit"""
        client, _, ch = setup
        f32 = ch[sy.DataType.FLOAT32].name
        self._create_and_read(
            client, f"return 1 + {f32} + 2 + {f32} + 3"
        )
