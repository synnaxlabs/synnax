#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test: create calculated channels and verify after upgrade."""

from abc import abstractmethod
from typing import Any

import numpy as np
import synnax as sy

from framework.test_case import TestCase

NpArray = np.ndarray[Any, Any]

# Source channels
CALC_IDX = "mig_calc_idx"
CALC_SRC_F32 = "mig_calc_src_f32"
CALC_SRC_F32_B = "mig_calc_src_f32_b"
CALC_SRC_F64 = "mig_calc_src_f64"
CALC_SRC_I64 = "mig_calc_src_i64"
CALC_RESET = "mig_calc_reset"

# Source data written during setup and used for type-handling verification
CALC_F32_DATA = np.array([10.0, 20.0, 30.0, 50.0, 100.0], dtype=np.float32)
CALC_F32_B_DATA = np.array([5.0, 15.0, 25.0, 35.0, 45.0], dtype=np.float32)
CALC_F64_DATA = np.array([100.0, 200.0, 300.0, 500.0, 1000.0], dtype=np.float64)
CALC_I64_DATA = np.array([1000, 2000, 3000, 5000, 10000], dtype=np.int64)

# Passthrough operations (expression = return source)
# (name, operation_type, window_duration, uses_reset_channel)
PASSTHROUGH_EXPR = f"return {CALC_SRC_F32}"
CALC_OP_CHANNELS: list[tuple[str, str, sy.TimeSpan, bool]] = [
    # Cumulative (no window, no reset)
    ("mig_calc_op_avg", "avg", sy.TimeSpan(0), False),
    ("mig_calc_op_min", "min", sy.TimeSpan(0), False),
    ("mig_calc_op_max", "max", sy.TimeSpan(0), False),
    # Windowed (window, no reset)
    ("mig_calc_op_avg_win", "avg", 5 * sy.TimeSpan.SECOND, False),
    ("mig_calc_op_min_win", "min", 10 * sy.TimeSpan.SECOND, False),
    ("mig_calc_op_max_win", "max", 15 * sy.TimeSpan.SECOND, False),
    # Reset channel only (no window)
    ("mig_calc_op_avg_rst", "avg", sy.TimeSpan(0), True),
    ("mig_calc_op_min_rst", "min", sy.TimeSpan(0), True),
    ("mig_calc_op_max_rst", "max", sy.TimeSpan(0), True),
    # Window + reset channel
    ("mig_calc_op_avg_win_rst", "avg", 5 * sy.TimeSpan.SECOND, True),
    ("mig_calc_op_min_win_rst", "min", 10 * sy.TimeSpan.SECOND, True),
    ("mig_calc_op_max_win_rst", "max", 15 * sy.TimeSpan.SECOND, True),
]

# Expression + operation (non-trivial expression with aggregation)
# (name, operation_type, window_duration)
CALC_EXPR = f"return {CALC_SRC_F32} * 2 + 5"
CALC_EXPR_OP_CHANNELS: list[tuple[str, str, sy.TimeSpan]] = [
    ("mig_calc_expr_avg", "avg", sy.TimeSpan(0)),
    ("mig_calc_expr_min", "min", sy.TimeSpan(0)),
    ("mig_calc_expr_max", "max", sy.TimeSpan(0)),
    ("mig_calc_expr_avg_win", "avg", 5 * sy.TimeSpan.SECOND),
    ("mig_calc_expr_min_win", "min", 10 * sy.TimeSpan.SECOND),
    ("mig_calc_expr_max_win", "max", 15 * sy.TimeSpan.SECOND),
]

# Type handling (various expression patterns across data types)
# (name, expression, expected data type)
CALC_TYPE_CHANNELS: list[tuple[str, str, sy.DataType]] = [
    ("mig_calc_add_lit", f"return {CALC_SRC_F32} + 10", sy.DataType.FLOAT32),
    ("mig_calc_mul_float", f"return {CALC_SRC_F32} * 2.5", sy.DataType.FLOAT32),
    ("mig_calc_power", f"return {CALC_SRC_F32} ^ 2", sy.DataType.FLOAT32),
    ("mig_calc_complex", f"return ({CALC_SRC_F32} - 32) * 5 / 9", sy.DataType.FLOAT32),
    ("mig_calc_two_f32", f"return {CALC_SRC_F32} + {CALC_SRC_F32_B}", sy.DataType.FLOAT32),
    ("mig_calc_f64_mul", f"return {CALC_SRC_F64} * 3.14159", sy.DataType.FLOAT64),
    ("mig_calc_i64_add", f"return {CALC_SRC_I64} + 100", sy.DataType.INT64),
    ("mig_calc_inverse", f"return 10000.0 / {CALC_SRC_F32}", sy.DataType.FLOAT32),
]

# Nested calc chain (each level references the previous calc channel by name)
CALC_NESTED_L1 = "mig_calc_nested_l1"
CALC_NESTED_L2 = "mig_calc_nested_l2"
CALC_NESTED_L3 = "mig_calc_nested_l3"
CALC_NESTED_CHANNELS: list[tuple[str, str]] = [
    (CALC_NESTED_L1, f"return {CALC_SRC_F32} * 3"),
    (CALC_NESTED_L2, f"return {CALC_NESTED_L1} + 100"),
    (CALC_NESTED_L3, f"return {CALC_NESTED_L2} / 2"),
]


class CalcChannelsMigration(TestCase):
    """Base class defining the migration test contract for calculated channels."""

    def run(self) -> None:
        self.test_calc_operations()
        self.test_calc_expr_operations()
        self.test_calc_type_handling()
        self.test_calc_nested()
        # Windowed test runs last — it writes second-interval timestamps that
        # extend the domain into the future.
        self.test_calc_windowed()

    @abstractmethod
    def test_calc_operations(self) -> None: ...

    @abstractmethod
    def test_calc_expr_operations(self) -> None: ...

    @abstractmethod
    def test_calc_type_handling(self) -> None: ...

    @abstractmethod
    def test_calc_nested(self) -> None: ...

    @abstractmethod
    def test_calc_windowed(self) -> None: ...


class CalcChannelsSetup(CalcChannelsMigration):
    """Create calculated channels for migration verification."""

    def test_calc_operations(self) -> None:
        self.log("Testing: Create calc channels with operations")
        client = self.client

        calc_idx = client.channels.create(
            name=CALC_IDX,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name=CALC_SRC_F32,
            data_type=sy.DataType.FLOAT32,
            index=calc_idx.key,
            retrieve_if_name_exists=True,
        )
        reset = client.channels.create(
            name=CALC_RESET,
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        for name, op_type, duration, uses_reset in CALC_OP_CHANNELS:
            op = sy.channel.Operation(
                type=op_type,
                duration=duration,
                reset_channel=reset.key if uses_reset else 0,
            )
            client.channels.create(
                name=name,
                data_type=sy.DataType.FLOAT32,
                expression=PASSTHROUGH_EXPR,
                operations=[op],
                retrieve_if_name_exists=True,
            )
        self.log(f"Created {len(CALC_OP_CHANNELS)} passthrough operation channels")

    def test_calc_expr_operations(self) -> None:
        self.log("Testing: Create calc channels with expression + operation")
        client = self.client
        for name, op_type, duration in CALC_EXPR_OP_CHANNELS:
            op = sy.channel.Operation(type=op_type, duration=duration)
            client.channels.create(
                name=name,
                data_type=sy.DataType.FLOAT32,
                expression=CALC_EXPR,
                operations=[op],
                retrieve_if_name_exists=True,
            )
        self.log(f"Created {len(CALC_EXPR_OP_CHANNELS)} expression+operation channels")

    def test_calc_type_handling(self) -> None:
        self.log("Testing: Create calc channels with typed expressions")
        client = self.client

        calc_idx = client.channels.retrieve(CALC_IDX)
        src_f32 = client.channels.retrieve(CALC_SRC_F32)
        src_f32_b = client.channels.create(
            name=CALC_SRC_F32_B,
            data_type=sy.DataType.FLOAT32,
            index=calc_idx.key,
            retrieve_if_name_exists=True,
        )
        src_f64 = client.channels.create(
            name=CALC_SRC_F64,
            data_type=sy.DataType.FLOAT64,
            index=calc_idx.key,
            retrieve_if_name_exists=True,
        )
        src_i64 = client.channels.create(
            name=CALC_SRC_I64,
            data_type=sy.DataType.INT64,
            index=calc_idx.key,
            retrieve_if_name_exists=True,
        )

        for name, expression, _ in CALC_TYPE_CHANNELS:
            client.channels.create(
                name=name,
                expression=expression,
                retrieve_if_name_exists=True,
            )
        self.log(f"Created {len(CALC_TYPE_CHANNELS)} typed calc channels")

    def test_calc_nested(self) -> None:
        self.log("Testing: Create nested calc chain")
        client = self.client
        # Order matters — each level references the previous
        for name, expression in CALC_NESTED_CHANNELS:
            client.channels.create(
                name=name,
                expression=expression,
                retrieve_if_name_exists=True,
            )
        self.log(f"Created {len(CALC_NESTED_CHANNELS)}-level nested calc chain")

    def test_calc_windowed(self) -> None:
        pass


class CalcChannelsVerify(CalcChannelsMigration):
    """Verify calculated channels survive migration with correct config and output."""

    def test_calc_operations(self) -> None:
        self.log("Testing: Calc channel operations metadata")
        client = self.client

        reset = client.channels.retrieve(CALC_RESET)
        for name, expected_op, expected_dur, uses_reset in CALC_OP_CHANNELS:
            ch = client.channels.retrieve(name)
            assert ch.expression == PASSTHROUGH_EXPR, f"{name}: expression mismatch"
            assert ch.data_type == sy.DataType.FLOAT32, f"{name}: type mismatch"
            assert len(ch.operations) == 1, f"{name}: expected 1 operation"
            assert ch.operations[0].type == expected_op, (
                f"{name}: expected op {expected_op}, got {ch.operations[0].type}"
            )
            assert ch.operations[0].duration == expected_dur, (
                f"{name}: duration mismatch: {ch.operations[0].duration}"
            )
            if uses_reset:
                assert ch.operations[0].reset_channel == reset.key, (
                    f"{name}: reset_channel mismatch"
                )

        # Functional verification: write data and stream results
        self.log("Testing: Calc operations functional (write + stream)")
        calc_idx = client.channels.retrieve(CALC_IDX)
        src = client.channels.retrieve(CALC_SRC_F32)
        calc_avg = client.channels.retrieve("mig_calc_op_avg")
        calc_min = client.channels.retrieve("mig_calc_op_min")
        calc_max = client.channels.retrieve("mig_calc_op_max")

        MS = sy.TimeSpan.MILLISECOND
        start = sy.TimeStamp.now()
        stream_keys = [calc_avg.key, calc_min.key, calc_max.key]
        with client.open_streamer(stream_keys) as streamer:
            with client.open_writer(
                start, [calc_idx.key, src.key]
            ) as writer:
                timestamps = np.array(
                    [start + i * MS for i in range(1, 4)],
                    dtype=np.int64,
                )
                writer.write({
                    calc_idx.key: timestamps,
                    src.key: np.array([10.0, 20.0, 30.0], dtype=np.float32),
                })
                frame = streamer.read(timeout=5)
                assert frame is not None, "No streamer frame received"
                avg_val = float(frame[calc_avg.key][0])
                min_val = float(frame[calc_min.key][0])
                max_val = float(frame[calc_max.key][0])
                assert abs(avg_val - 20.0) < 0.01, f"avg: expected 20.0, got {avg_val}"
                assert abs(min_val - 10.0) < 0.01, f"min: expected 10.0, got {min_val}"
                assert abs(max_val - 30.0) < 0.01, f"max: expected 30.0, got {max_val}"

        # Functional verification: reset channel
        calc_avg_rst = client.channels.retrieve("mig_calc_op_avg_rst")
        start = sy.TimeStamp.now()
        with client.open_streamer(calc_avg_rst.key) as streamer:
            with client.open_writer(
                start, [calc_idx.key, src.key, reset.key]
            ) as writer:
                ts1 = np.array(
                    [start + i * MS for i in range(1, 4)],
                    dtype=np.int64,
                )
                writer.write({
                    calc_idx.key: ts1,
                    src.key: np.array([10.0, 20.0, 30.0], dtype=np.float32),
                    reset.key: np.array([0, 0, 0], dtype=np.uint8),
                })
                frame = streamer.read(timeout=5)
                assert frame is not None
                val = float(frame[calc_avg_rst.key][0])
                assert abs(val - 20.0) < 0.01, f"avg_rst batch1: expected 20.0, got {val}"

                # Second batch with reset signal
                ts2 = np.array(
                    [start + i * MS for i in range(4, 6)],
                    dtype=np.int64,
                )
                writer.write({
                    calc_idx.key: ts2,
                    src.key: np.array([40.0, 50.0], dtype=np.float32),
                    reset.key: np.array([1, 0], dtype=np.uint8),
                })
                frame = streamer.read(timeout=5)
                assert frame is not None
                val = float(frame[calc_avg_rst.key][0])
                assert abs(val - 45.0) < 0.01, f"avg_rst after reset: expected 45.0, got {val}"

    def test_calc_expr_operations(self) -> None:
        self.log("Testing: Expression+operation channel metadata")
        client = self.client

        for name, expected_op, expected_dur in CALC_EXPR_OP_CHANNELS:
            ch = client.channels.retrieve(name)
            assert ch.expression == CALC_EXPR, f"{name}: expression mismatch"
            assert ch.data_type == sy.DataType.FLOAT32, f"{name}: type mismatch"
            assert len(ch.operations) == 1, f"{name}: expected 1 operation"
            assert ch.operations[0].type == expected_op, (
                f"{name}: expected op {expected_op}, got {ch.operations[0].type}"
            )
            assert ch.operations[0].duration == expected_dur, (
                f"{name}: duration mismatch: {ch.operations[0].duration}"
            )

        # Functional verification: expression * 2 + 5 with avg
        self.log("Testing: Expression+operation functional (write + stream)")
        calc_idx = client.channels.retrieve(CALC_IDX)
        src = client.channels.retrieve(CALC_SRC_F32)
        calc_expr_avg = client.channels.retrieve("mig_calc_expr_avg")
        calc_expr_min = client.channels.retrieve("mig_calc_expr_min")
        calc_expr_max = client.channels.retrieve("mig_calc_expr_max")

        MS = sy.TimeSpan.MILLISECOND
        start = sy.TimeStamp.now()
        stream_keys = [calc_expr_avg.key, calc_expr_min.key, calc_expr_max.key]
        with client.open_streamer(stream_keys) as streamer:
            with client.open_writer(
                start, [calc_idx.key, src.key]
            ) as writer:
                timestamps = np.array(
                    [start + i * MS for i in range(1, 4)],
                    dtype=np.int64,
                )
                # source=[10, 20, 30] → expr=source*2+5 → [25, 45, 65]
                writer.write({
                    calc_idx.key: timestamps,
                    src.key: np.array([10.0, 20.0, 30.0], dtype=np.float32),
                })
                frame = streamer.read(timeout=5)
                assert frame is not None, "No streamer frame received"
                avg_val = float(frame[calc_expr_avg.key][0])
                min_val = float(frame[calc_expr_min.key][0])
                max_val = float(frame[calc_expr_max.key][0])
                # avg(25,45,65)=45, min=25, max=65
                assert abs(avg_val - 45.0) < 0.01, f"expr_avg: expected 45.0, got {avg_val}"
                assert abs(min_val - 25.0) < 0.01, f"expr_min: expected 25.0, got {min_val}"
                assert abs(max_val - 65.0) < 0.01, f"expr_max: expected 65.0, got {max_val}"

    def test_calc_type_handling(self) -> None:
        self.log("Testing: Calc type handling metadata")
        client = self.client

        for name, expression, expected_type in CALC_TYPE_CHANNELS:
            ch = client.channels.retrieve(name)
            assert ch.expression == expression, (
                f"{name}: expression mismatch: {ch.expression!r}"
            )
            assert ch.data_type == expected_type, (
                f"{name}: expected {expected_type}, got {ch.data_type}"
            )

        # Write source data and read computed values (calcs are virtual —
        # they evaluate on-demand, so we write then immediately read).
        self.log("Testing: Calc type handling computed values")
        calc_idx = client.channels.retrieve(CALC_IDX)
        src_f32 = client.channels.retrieve(CALC_SRC_F32)
        src_f32_b = client.channels.retrieve(CALC_SRC_F32_B)
        src_f64 = client.channels.retrieve(CALC_SRC_F64)
        src_i64 = client.channels.retrieve(CALC_SRC_I64)

        MS = sy.TimeSpan.MILLISECOND
        sample_count = len(CALC_F32_DATA)
        start = sy.TimeStamp.now()
        timestamps = np.array(
            [start + i * MS for i in range(1, sample_count + 1)],
            dtype=np.int64,
        )
        with client.open_writer(
            start=start,
            channels=[calc_idx.key, src_f32.key, src_f32_b.key, src_f64.key, src_i64.key],
        ) as writer:
            writer.write({
                calc_idx.key: timestamps,
                src_f32.key: CALC_F32_DATA,
                src_f32_b.key: CALC_F32_B_DATA,
                src_f64.key: CALC_F64_DATA,
                src_i64.key: CALC_I64_DATA,
            })

        # Give the calc engine time to evaluate all samples
        sy.sleep(1)

        calc_channels = {
            name: client.channels.retrieve(name) for name, _, _ in CALC_TYPE_CHANNELS
        }

        f32 = CALC_F32_DATA
        f32_b = CALC_F32_B_DATA
        f64 = CALC_F64_DATA
        i64 = CALC_I64_DATA

        expected_values: dict[str, NpArray] = {
            "mig_calc_add_lit": (f32 + 10).astype(np.float32),
            "mig_calc_mul_float": (f32 * 2.5).astype(np.float32),
            "mig_calc_power": (f32**2).astype(np.float32),
            "mig_calc_complex": ((f32 - 32) * 5 / 9).astype(np.float32),
            "mig_calc_two_f32": (f32 + f32_b).astype(np.float32),
            "mig_calc_f64_mul": (f64 * 3.14159).astype(np.float64),
            "mig_calc_i64_add": (i64 + 100).astype(np.int64),
            "mig_calc_inverse": (10000.0 / f32).astype(np.float32),
        }

        tr = sy.TimeRange(start, sy.TimeStamp.now())
        keys = [calc_channels[name].key for name in expected_values]
        frame = client.read(tr, keys)

        for name, expected in expected_values.items():
            ch = calc_channels[name]
            data = frame[ch.key].to_numpy()
            assert len(data) == len(expected), (
                f"{name}: expected {len(expected)} samples, got {len(data)}"
            )
            if expected.dtype in (np.float32, np.float64):
                assert np.allclose(data, expected, rtol=1e-5), (
                    f"{name}: value mismatch: {data} vs {expected}"
                )
            else:
                assert np.array_equal(data, expected), (
                    f"{name}: value mismatch: {data} vs {expected}"
                )

    def test_calc_nested(self) -> None:
        self.log("Testing: Nested calc chain metadata")
        client = self.client

        for name, expression in CALC_NESTED_CHANNELS:
            ch = client.channels.retrieve(name)
            assert ch.expression == expression, (
                f"{name}: expression mismatch: {ch.expression!r}"
            )
            assert ch.data_type == sy.DataType.FLOAT32, (
                f"{name}: expected FLOAT32, got {ch.data_type}"
            )

        # Functional: write source data, stream through 3-level chain
        # source=[10,20,30] → L1=source*3=[30,60,90] → L2=L1+100=[130,160,190]
        # → L3=L2/2=[65,80,95]
        calc_idx = client.channels.retrieve(CALC_IDX)
        src = client.channels.retrieve(CALC_SRC_F32)
        l3 = client.channels.retrieve(CALC_NESTED_L3)

        MS = sy.TimeSpan.MILLISECOND
        start = sy.TimeStamp.now()
        with client.open_streamer(l3.key) as streamer:
            with client.open_writer(
                start, [calc_idx.key, src.key]
            ) as writer:
                timestamps = np.array(
                    [start + i * MS for i in range(1, 4)],
                    dtype=np.int64,
                )
                writer.write({
                    calc_idx.key: timestamps,
                    src.key: np.array([10.0, 20.0, 30.0], dtype=np.float32),
                })
                frame = streamer.read(timeout=5)
                assert frame is not None, "No streamer frame for nested chain"
                data = frame[l3.key].to_numpy()
                expected = np.array([65.0, 80.0, 95.0], dtype=np.float32)
                assert np.allclose(data, expected, rtol=1e-5), (
                    f"nested L3: expected {expected}, got {data}"
                )

    def test_calc_windowed(self) -> None:
        self.log("Testing: Windowed operation functional verification")
        client = self.client
        calc_idx = client.channels.retrieve(CALC_IDX)
        src = client.channels.retrieve(CALC_SRC_F32)
        avg_win = client.channels.retrieve("mig_calc_op_avg_win")
        min_win = client.channels.retrieve("mig_calc_op_min_win")
        max_win = client.channels.retrieve("mig_calc_op_max_win")

        S = sy.TimeSpan.SECOND
        MS = sy.TimeSpan.MILLISECOND
        N = 5

        # Four batches of data, each written at increasing time offsets.
        # The offsets are chosen so that each batch crosses the next window
        # boundary: avg_win resets at 5s, min_win at 10s, max_win at 15s.
        batch1 = np.linspace(50, 10, N, dtype=np.float32)
        batch2 = np.linspace(5, 45, N, dtype=np.float32)
        batch3 = np.linspace(100, 20, N, dtype=np.float32)
        batch4 = np.linspace(3, 19, N, dtype=np.float32)

        start = sy.TimeStamp.now()
        keys = [avg_win.key, min_win.key, max_win.key]
        with client.open_streamer(keys) as streamer:
            with client.open_writer(
                start, [calc_idx.key, src.key]
            ) as writer:

                def write_batch(
                    data: NpArray, offset: sy.TimeSpan
                ) -> sy.Frame:
                    ts = np.array(
                        [start + offset + i * MS for i in range(N)],
                        dtype=np.int64,
                    )
                    writer.write({calc_idx.key: ts, src.key: data})
                    frame = streamer.read(timeout=5)
                    assert frame is not None, "No streamer frame"
                    return frame

                def check(
                    frame: sy.Frame,
                    key: int,
                    expected: float,
                    label: str,
                ) -> None:
                    val = float(frame[key][0])
                    assert abs(val - expected) < 0.1, (
                        f"{label}: expected {expected}, got {val}"
                    )

                # Phase 1: t+1ms — all windows accumulating
                all_so_far = batch1
                f = write_batch(batch1, 1 * MS)
                check(f, avg_win.key, float(np.mean(batch1)), "avg_win phase1")
                check(f, min_win.key, float(np.min(batch1)), "min_win phase1")
                check(f, max_win.key, float(np.max(batch1)), "max_win phase1")

                # Phase 2: t+6s — avg_win resets (>5s), min/max still accumulating
                all_so_far = np.concatenate([all_so_far, batch2])
                f = write_batch(batch2, 6 * S)
                check(f, avg_win.key, float(np.mean(batch2)), "avg_win phase2")
                check(f, min_win.key, float(np.min(all_so_far)), "min_win phase2")
                check(f, max_win.key, float(np.max(all_so_far)), "max_win phase2")

                # Phase 3: t+11s — min_win resets (>10s), max still accumulating
                all_so_far = np.concatenate([all_so_far, batch3])
                f = write_batch(batch3, 11 * S)
                check(f, avg_win.key, float(np.mean(batch3)), "avg_win phase3")
                check(f, min_win.key, float(np.min(batch3)), "min_win phase3")
                check(f, max_win.key, float(np.max(all_so_far)), "max_win phase3")

                # Phase 4: t+16s — max_win resets (>15s),
                # min_win still in its second window (started at t+11s, +10s = t+21s)
                min_since_reset = np.concatenate([batch3, batch4])
                f = write_batch(batch4, 16 * S)
                check(f, avg_win.key, float(np.mean(batch4)), "avg_win phase4")
                check(f, min_win.key, float(np.min(min_since_reset)), "min_win phase4")
                check(f, max_win.key, float(np.max(batch4)), "max_win phase4")
