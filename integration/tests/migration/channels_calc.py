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


def invert_moving_avg(y: NpArray, window_samples: int) -> NpArray:
    """Back-calculate input x such that moving_avg(x, window) ≈ y.

    Uses the identity: x[n] = K*(y[n] - y[n-1]) + x[n-K]
    derived by differentiating y(t) = (1/W) * ∫[t-W,t] x(τ)dτ.

    x = input samples, y = desired moving average output,
    K = window size in samples, W = window duration, τ = integration variable.
    """
    n_samples = len(y)
    k = window_samples
    x = np.zeros(n_samples, dtype=np.float64)
    x[0] = float(y[0])
    for i in range(1, min(k, n_samples)):
        x[i] = (i + 1) * float(y[i]) - i * float(y[i - 1])
    for i in range(k, n_samples):
        x[i] = k * (float(y[i]) - float(y[i - 1])) + x[i - k]
    return x.astype(np.float32)


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
        self.log("Testing: Windowed avg — multi-domain verification")
        client = self.client
        calc_idx = client.channels.retrieve(CALC_IDX)
        src = client.channels.retrieve(CALC_SRC_F32)
        avg_win = client.channels.retrieve("mig_calc_op_avg_win")

        S = sy.TimeSpan.SECOND
        MS = sy.TimeSpan.MILLISECOND

        # Write multiple domains with known data using seed functions.
        # Each domain is a separate writer (enable_auto_commit so it
        # finalizes immediately). The avg operation produces one aggregated
        # value per domain on historical read.
        RATE = 50  # Hz
        SAMPLES_PER_DOMAIN = 100  # 2 seconds of data per domain
        NUM_DOMAINS = 10
        dt_ms = int(1000 / RATE)  # 20ms per sample

        # Seed functions applied to domain index to vary data across domains
        def cosine_domain(d: int) -> NpArray:
            phase = 2 * np.pi * d / NUM_DOMAINS
            return (np.cos(phase) * 50 + 50).astype(np.float32) + np.linspace(
                0, 10, SAMPLES_PER_DOMAIN, dtype=np.float32
            )

        def linear_domain(d: int) -> NpArray:
            base = d * 10.0
            return np.linspace(base, base + 20, SAMPLES_PER_DOMAIN, dtype=np.float32)

        def quadratic_domain(d: int) -> NpArray:
            t = np.linspace(0, 1, SAMPLES_PER_DOMAIN, dtype=np.float32)
            return ((d + 1) * t**2 * 100).astype(np.float32)

        seeds = [
            ("cosine", cosine_domain),
            ("linear", linear_domain),
            ("quadratic", quadratic_domain),
        ]

        start = sy.TimeStamp.now()
        # Track per-seed domain data for debug plot and assertions
        all_inputs: list[list[NpArray]] = [[] for _ in seeds]
        all_expected: list[list[float]] = [[] for _ in seeds]
        domain_starts: list[sy.TimeStamp] = []

        for d in range(NUM_DOMAINS):
            domain_start = start + d * 3 * S  # 3s gap between domains
            domain_starts.append(domain_start)
            timestamps = np.array(
                [domain_start + i * dt_ms * MS for i in range(SAMPLES_PER_DOMAIN)],
                dtype=np.int64,
            )

            # Cycle through seeds — each domain uses one seed
            seed_idx = d % len(seeds)
            seed_name, seed_fn = seeds[seed_idx]
            data = seed_fn(d)

            with client.open_writer(
                domain_start,
                [calc_idx.key, src.key],
                enable_auto_commit=True,
            ) as writer:
                writer.write({calc_idx.key: timestamps, src.key: data})

            all_inputs[seed_idx].append(data)
            all_expected[seed_idx].append(float(np.mean(data)))
            self.log(f"Domain {d} ({seed_name}): {len(data)} samples, "
                     f"mean={np.mean(data):.2f}")

        sy.sleep(2)

        # Read calc channel output — includes calc index for proper frame access
        end = start + NUM_DOMAINS * 3 * S
        tr = sy.TimeRange(start, end)
        frame = client.read(tr, [avg_win.key, avg_win.index])
        calc_multi = frame[avg_win.key]
        self.log(f"Read {len(calc_multi.series)} series from avg_win")

        # Extract one value per series (domain)
        calc_values = [float(s[0]) for s in calc_multi.series]
        self.log(f"Calc values: {calc_values}")

        # Debug plot: input distributions per domain vs calc output
        try:
            import matplotlib
            matplotlib.use("Agg")
            import matplotlib.pyplot as plt

            fig, axes = plt.subplots(2, 1, figsize=(14, 8))

            # Top: input data per domain (box-like summary)
            ax = axes[0]
            colors = ["tab:blue", "tab:green", "tab:red"]
            for d in range(NUM_DOMAINS):
                seed_idx = d % len(seeds)
                seed_name = seeds[seed_idx][0]
                data = seeds[seed_idx][1](d)
                t_domain = np.linspace(d * 3, d * 3 + 2, len(data))
                ax.plot(t_domain, data, color=colors[seed_idx],
                        alpha=0.4, linewidth=0.5)
                ax.axvline(d * 3, color="gray", alpha=0.2, linewidth=0.5)
            ax.set_title("Input data per domain (colored by seed)")
            ax.set_ylabel("Value")
            ax.grid(True, alpha=0.3)
            # Manual legend
            for i, (name, _) in enumerate(seeds):
                ax.plot([], [], color=colors[i], label=name)
            ax.legend()

            # Bottom: expected avg vs actual calc output per domain
            ax = axes[1]
            domain_times = [d * 3 + 1 for d in range(NUM_DOMAINS)]
            expected_flat = [float(np.mean(seeds[d % len(seeds)][1](d)))
                            for d in range(NUM_DOMAINS)]
            ax.plot(domain_times, expected_flat, "o--", color="tab:green",
                    label="Expected avg", markersize=8)
            if len(calc_values) == NUM_DOMAINS:
                ax.plot(domain_times, calc_values, "s-", color="tab:orange",
                        label="Calc channel output", markersize=6)
            else:
                ax.plot(range(len(calc_values)), calc_values, "s-",
                        color="tab:orange",
                        label=f"Calc output ({len(calc_values)} values)")
            ax.set_title("Expected vs actual avg per domain")
            ax.set_xlabel("Time (s)")
            ax.set_ylabel("Average value")
            ax.legend()
            ax.grid(True, alpha=0.3)

            plt.tight_layout()
            plot_path = "/Users/nicoalba/Desktop/calc_windowed_debug.png"
            plt.savefig(plot_path, dpi=150)
            plt.close()
            self.log(f"Debug plot saved to {plot_path}")
        except Exception as e:
            self.log(f"Could not save debug plot: {e}")

        # Assert: one value per domain, matching the expected mean
        assert len(calc_values) == NUM_DOMAINS, (
            f"Expected {NUM_DOMAINS} domain values, got {len(calc_values)}"
        )
        for d in range(NUM_DOMAINS):
            seed_idx = d % len(seeds)
            seed_name = seeds[seed_idx][0]
            expected = float(np.mean(seeds[seed_idx][1](d)))
            actual = calc_values[d]
            assert abs(actual - expected) < 0.5, (
                f"Domain {d} ({seed_name}): expected {expected:.2f}, "
                f"got {actual:.2f}"
            )
