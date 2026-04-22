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

S = sy.TimeSpan.SECOND
MS = sy.TimeSpan.MILLISECOND

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

# Source data used in functional verify tests (write -> stream -> assert)
VERIFY_SRC_DATA = np.array([10.0, 20.0, 30.0], dtype=np.float32)

# Passthrough operations (expression = return source)
# (name, operation_type, window_duration, uses_reset_channel)
PASSTHROUGH_EXPR = f"return {CALC_SRC_F32}"
CALC_OP_CHANNELS: list[tuple[str, str, sy.TimeSpan, bool]] = [
    ("mig_calc_op_avg", "avg", sy.TimeSpan(0), False),
    ("mig_calc_op_min", "min", sy.TimeSpan(0), False),
    ("mig_calc_op_max", "max", sy.TimeSpan(0), False),
    ("mig_calc_op_avg_win", "avg", 5 * S, False),
    ("mig_calc_op_min_win", "min", 10 * S, False),
    ("mig_calc_op_max_win", "max", 15 * S, False),
    ("mig_calc_op_avg_rst", "avg", sy.TimeSpan(0), True),
    ("mig_calc_op_min_rst", "min", sy.TimeSpan(0), True),
    ("mig_calc_op_max_rst", "max", sy.TimeSpan(0), True),
    ("mig_calc_op_avg_win_rst", "avg", 5 * S, True),
    ("mig_calc_op_min_win_rst", "min", 10 * S, True),
    ("mig_calc_op_max_win_rst", "max", 15 * S, True),
]

# Expression + operation channels (non-trivial expression with aggregation).
CALC_EXPR = f"return {CALC_SRC_F32} * 2 + 5"
CALC_EXPR_OP_CHANNELS: list[tuple[str, str]] = [
    ("mig_calc_expr_avg", "avg"),
    ("mig_calc_expr_min", "min"),
    ("mig_calc_expr_max", "max"),
]

# Type handling: each entry tests a distinct concern for migration.
# (name, expression, expected data type)
CALC_TYPE_CHANNELS: list[tuple[str, str, sy.DataType]] = [
    # Compound single-source f32 expression
    ("mig_calc_complex", f"return ({CALC_SRC_F32} - 32) * 5 / 9", sy.DataType.FLOAT32),
    # Multi-source expression
    (
        "mig_calc_two_f32",
        f"return {CALC_SRC_F32} + {CALC_SRC_F32_B}",
        sy.DataType.FLOAT32,
    ),
    ("mig_calc_f64_mul", f"return {CALC_SRC_F64} * 3.14159", sy.DataType.FLOAT64),
    ("mig_calc_i64_add", f"return {CALC_SRC_I64} + 100", sy.DataType.INT64),
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

# Windowed test: two calc channels whose output traces cosine and quadratic shapes.
WIN_EPOCH_BASE = 100
WIN_NUM_DOMAINS = 300
WIN_DOMAIN_GAP_S = 0.05
WIN_SAMPLES_PER_DOMAIN = 40
WIN_DT_MS = 1
WIN_WINDOW_S = 0.02
WIN_NOISE_STD = 0.1

WIN_SRC_COS = "mig_win_src_cos"
WIN_SRC_QUAD = "mig_win_src_quad"
WIN_IDX = "mig_win_idx"

WIN_CALC_COS = "mig_win_calc_cos"
WIN_CALC_QUAD = "mig_win_calc_quad"


def _timestamps(start: sy.TimeStamp, count: int, offset: int = 1) -> NpArray:
    return np.array(
        [start + i * MS for i in range(offset, offset + count)],
        dtype=np.int64,
    )


def _win_value(seed: str, d: int) -> float:
    t = d * WIN_DOMAIN_GAP_S
    if seed == "cosine":
        return float(np.cos(2 * np.pi * t / 2.5))
    return float(t**2)


def _win_noisy_data(seed: str, d: int) -> NpArray:
    rng = np.random.default_rng(seed=hash((seed, d)) & 0xFFFFFFFF)
    center = _win_value(seed, d)
    return (center + rng.normal(0, WIN_NOISE_STD, WIN_SAMPLES_PER_DOMAIN)).astype(
        np.float32
    )


class CalcChannelsMigration(TestCase):
    """Base class defining the migration test contract for calculated channels."""

    def run(self) -> None:
        self.test_calc_operations()
        self.test_calc_type_handling()
        self.test_calc_nested()
        self.test_calc_windowed()

    @abstractmethod
    def test_calc_operations(self) -> None: ...

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
            client.channels.create(
                name=name,
                data_type=sy.DataType.FLOAT32,
                expression=PASSTHROUGH_EXPR,
                operations=[
                    sy.channel.Operation(
                        type=op_type,
                        duration=duration,
                        reset_channel=reset.key if uses_reset else 0,
                    )
                ],
                retrieve_if_name_exists=True,
            )

        for name, op_type in CALC_EXPR_OP_CHANNELS:
            client.channels.create(
                name=name,
                data_type=sy.DataType.FLOAT32,
                expression=CALC_EXPR,
                operations=[sy.channel.Operation(type=op_type)],
                retrieve_if_name_exists=True,
            )

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

        start = sy.TimeStamp(WIN_EPOCH_BASE * S)
        with client.open_writer(
            start=start,
            channels=[
                calc_idx.key,
                src_f32.key,
                src_f32_b.key,
                src_f64.key,
                src_i64.key,
            ],
            enable_auto_commit=True,
        ) as writer:
            writer.write(
                {
                    calc_idx.key: _timestamps(start, len(CALC_F32_DATA)),
                    src_f32.key: CALC_F32_DATA,
                    src_f32_b.key: CALC_F32_B_DATA,
                    src_f64.key: CALC_F64_DATA,
                    src_i64.key: CALC_I64_DATA,
                }
            )

    def test_calc_nested(self) -> None:
        self.log("Testing: Create nested calc chain")
        for name, expression in CALC_NESTED_CHANNELS:
            self.client.channels.create(
                name=name,
                expression=expression,
                retrieve_if_name_exists=True,
            )

    def test_calc_windowed(self) -> None:
        self.log("Testing: Create windowed calc channels and write data")
        client = self.client

        idx = client.channels.create(
            name=WIN_IDX,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        src_cos = client.channels.create(
            name=WIN_SRC_COS,
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
            retrieve_if_name_exists=True,
        )
        src_quad = client.channels.create(
            name=WIN_SRC_QUAD,
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
            retrieve_if_name_exists=True,
        )

        for calc_name, src_name in [
            (WIN_CALC_COS, WIN_SRC_COS),
            (WIN_CALC_QUAD, WIN_SRC_QUAD),
        ]:
            client.channels.create(
                name=calc_name,
                data_type=sy.DataType.FLOAT32,
                expression=f"return {src_name}",
                operations=[
                    sy.channel.Operation(
                        type="avg",
                        duration=WIN_WINDOW_S * S,
                    )
                ],
                retrieve_if_name_exists=True,
            )

        base = sy.TimeStamp(WIN_EPOCH_BASE * S) + 10 * S
        write_keys = [idx.key, src_cos.key, src_quad.key]

        for d in range(WIN_NUM_DOMAINS):
            domain_start = base + int(d * WIN_DOMAIN_GAP_S * 1000) * MS
            timestamps = np.array(
                [
                    domain_start + i * WIN_DT_MS * MS
                    for i in range(WIN_SAMPLES_PER_DOMAIN)
                ],
                dtype=np.int64,
            )
            with client.open_writer(
                domain_start,
                write_keys,
                enable_auto_commit=True,
            ) as writer:
                writer.write(
                    {
                        idx.key: timestamps,
                        src_cos.key: _win_noisy_data("cosine", d),
                        src_quad.key: _win_noisy_data("quadratic", d),
                    }
                )


class CalcChannelsVerify(CalcChannelsMigration):
    """Verify calculated channels survive migration."""

    _calc_idx: sy.Channel
    _src: sy.Channel

    def run(self) -> None:
        self._calc_idx = self.client.channels.retrieve(CALC_IDX)
        self._src = self.client.channels.retrieve(CALC_SRC_F32)
        super().run()

    def _stream_and_assert(self, expected: dict[str, float]) -> None:
        """Write VERIFY_SRC_DATA, stream calc results, assert scalar values."""
        channels = {name: self.client.channels.retrieve(name) for name in expected}
        start = sy.TimeStamp.now()
        with self.client.open_streamer(
            [ch.key for ch in channels.values()]
        ) as streamer:
            with self.client.open_writer(
                start, [self._calc_idx.key, self._src.key]
            ) as writer:
                writer.write(
                    {
                        self._calc_idx.key: _timestamps(start, len(VERIFY_SRC_DATA)),
                        self._src.key: VERIFY_SRC_DATA,
                    }
                )
                frame = streamer.read(timeout=5)
                assert frame is not None, "No streamer frame received"
                for name, exp_val in expected.items():
                    actual = float(frame[channels[name].key][0])
                    assert abs(actual - exp_val) < 0.01, (
                        f"{name}: expected {exp_val}, got {actual}"
                    )

    def test_calc_operations(self) -> None:
        self.log("Testing: Calc channel operations metadata")
        reset = self.client.channels.retrieve(CALC_RESET)
        for name, expected_op, expected_dur, uses_reset in CALC_OP_CHANNELS:
            ch = self.client.channels.retrieve(name)
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

        for name, expected_op in CALC_EXPR_OP_CHANNELS:
            ch = self.client.channels.retrieve(name)
            assert ch.expression == CALC_EXPR, f"{name}: expression mismatch"
            assert len(ch.operations) == 1, f"{name}: expected 1 operation"
            assert ch.operations[0].type == expected_op, (
                f"{name}: expected op {expected_op}, got {ch.operations[0].type}"
            )

        self.log("Testing: Calc operations functional (write + stream)")
        self._stream_and_assert(
            {
                "mig_calc_op_avg": 20.0,
                "mig_calc_op_min": 10.0,
                "mig_calc_op_max": 30.0,
            }
        )

        # source=[10,20,30] -> expr=source*2+5 -> [25,45,65]
        self._stream_and_assert(
            {
                "mig_calc_expr_avg": 45.0,
                "mig_calc_expr_min": 25.0,
                "mig_calc_expr_max": 65.0,
            }
        )

        calc_avg_rst = self.client.channels.retrieve("mig_calc_op_avg_rst")
        start = sy.TimeStamp.now()
        with self.client.open_streamer(calc_avg_rst.key) as streamer:
            with self.client.open_writer(
                start, [self._calc_idx.key, self._src.key, reset.key]
            ) as writer:
                writer.write(
                    {
                        self._calc_idx.key: _timestamps(start, 3),
                        self._src.key: VERIFY_SRC_DATA,
                        reset.key: np.array([0, 0, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=5)
                assert frame is not None
                val = float(frame[calc_avg_rst.key][0])
                assert abs(val - 20.0) < 0.01, (
                    f"avg_rst batch1: expected 20.0, got {val}"
                )

                writer.write(
                    {
                        self._calc_idx.key: _timestamps(start, 2, offset=4),
                        self._src.key: np.array([40.0, 50.0], dtype=np.float32),
                        reset.key: np.array([1, 0], dtype=np.uint8),
                    }
                )
                frame = streamer.read(timeout=5)
                assert frame is not None
                val = float(frame[calc_avg_rst.key][0])
                assert abs(val - 45.0) < 0.01, (
                    f"avg_rst after reset: expected 45.0, got {val}"
                )

    def test_calc_type_handling(self) -> None:
        self.log("Testing: Calc type handling metadata")
        for name, expression, expected_type in CALC_TYPE_CHANNELS:
            ch = self.client.channels.retrieve(name)
            assert ch.expression == expression, (
                f"{name}: expression mismatch: {ch.expression!r}"
            )
            assert ch.data_type == expected_type, (
                f"{name}: expected {expected_type}, got {ch.data_type}"
            )

        self.log("Testing: Calc type handling computed values")
        calc_channels = {
            name: self.client.channels.retrieve(name)
            for name, _, _ in CALC_TYPE_CHANNELS
        }

        f32, f32_b = CALC_F32_DATA, CALC_F32_B_DATA
        f64, i64 = CALC_F64_DATA, CALC_I64_DATA
        expected_values: dict[str, NpArray] = {
            "mig_calc_complex": ((f32 - 32) * 5 / 9).astype(np.float32),
            "mig_calc_two_f32": (f32 + f32_b).astype(np.float32),
            "mig_calc_f64_mul": (f64 * 3.14159).astype(np.float64),
            "mig_calc_i64_add": (i64 + 100).astype(np.int64),
        }

        base = sy.TimeStamp(WIN_EPOCH_BASE * S)
        tr = sy.TimeRange(base, base + 10 * S)
        keys = [calc_channels[name].key for name in expected_values]
        frame = self.client.read(tr, keys)

        for name, expected in expected_values.items():
            data = frame[calc_channels[name].key].to_numpy()
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
        for name, expression in CALC_NESTED_CHANNELS:
            ch = self.client.channels.retrieve(name)
            assert ch.expression == expression, (
                f"{name}: expression mismatch: {ch.expression!r}"
            )
            assert ch.data_type == sy.DataType.FLOAT32, (
                f"{name}: expected FLOAT32, got {ch.data_type}"
            )

        # source=[10,20,30] -> L1=*3=[30,60,90] -> L2=+100=[130,160,190] -> L3=/2=[65,80,95]
        l3 = self.client.channels.retrieve(CALC_NESTED_L3)
        start = sy.TimeStamp.now()
        with self.client.open_streamer(l3.key) as streamer:
            with self.client.open_writer(
                start, [self._calc_idx.key, self._src.key]
            ) as writer:
                writer.write(
                    {
                        self._calc_idx.key: _timestamps(start, len(VERIFY_SRC_DATA)),
                        self._src.key: VERIFY_SRC_DATA,
                    }
                )
                frame = streamer.read(timeout=5)
                assert frame is not None, "No streamer frame for nested chain"
                data = frame[l3.key].to_numpy()
                expected = np.array([65.0, 80.0, 95.0], dtype=np.float32)
                assert np.allclose(data, expected, rtol=1e-5), (
                    f"nested L3: expected {expected}, got {data}"
                )

    def test_calc_windowed(self) -> None:
        self.log("Testing: Windowed calc channels")
        base = sy.TimeStamp(WIN_EPOCH_BASE * S) + 10 * S
        end = base + int(WIN_NUM_DOMAINS * WIN_DOMAIN_GAP_S * 1000) * MS
        tr = sy.TimeRange(base, end)

        calc_cos = self.client.channels.retrieve(WIN_CALC_COS)
        calc_quad = self.client.channels.retrieve(WIN_CALC_QUAD)

        frame = self.client.read(
            tr,
            [
                calc_cos.key,
                calc_cos.index,
                calc_quad.key,
            ],
        )
        for seed, calc_ch, max_rmse in [
            ("cosine", calc_cos, 0.2),
            ("quadratic", calc_quad, 0.5),
        ]:
            actual = np.array(
                [float(s[0]) for s in frame[calc_ch.key].series[:WIN_NUM_DOMAINS]]
            )
            expected = np.array([_win_value(seed, d) for d in range(len(actual))])
            rmse = float(np.sqrt(np.mean((actual - expected) ** 2)))
            assert rmse < max_rmse, f"{seed} rmse: {rmse:.4f}"
