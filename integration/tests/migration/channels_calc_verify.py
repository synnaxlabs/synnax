#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration verify: confirm calculated channels survived migration."""

from typing import Any

import numpy as np

import synnax as sy
from framework.test_case import TestCase
from tests.migration.channels_calc_setup import (
    CALC_EXPR,
    CALC_EXPR_OP_CHANNELS,
    CALC_F32_B_DATA,
    CALC_F32_DATA,
    CALC_F64_DATA,
    CALC_I64_DATA,
    CALC_IDX,
    CALC_NESTED_CHANNELS,
    CALC_OP_CHANNELS,
    CALC_RESET,
    CALC_SRC_F32,
    CALC_TYPE_CHANNELS,
    MS,
    PASSTHROUGH_EXPR,
    WIN_DOMAIN_GAP_S,
    WIN_NUM_DOMAINS,
    S,
    _win_value,
)

NpArray = np.ndarray[Any, Any]

VERIFY_SRC_DATA = np.array([10.0, 20.0, 30.0], dtype=np.float32)

CALC_TYPE_CHANNELS_TYPED: list[tuple[str, str, sy.DataType]] = [
    (name, expr, dt)
    for (name, expr), dt in zip(
        CALC_TYPE_CHANNELS,
        [
            sy.DataType.FLOAT32,
            sy.DataType.FLOAT32,
            sy.DataType.FLOAT64,
            sy.DataType.INT64,
        ],
    )
]

WIN_EPOCH_BASE = 100
WIN_SRC_COS = "mig_win_src_cos"
WIN_SRC_QUAD = "mig_win_src_quad"
WIN_CALC_COS = "mig_win_calc_cos"
WIN_CALC_QUAD = "mig_win_calc_quad"

CALC_NESTED_L1 = CALC_NESTED_CHANNELS[0][0]
CALC_NESTED_L2 = CALC_NESTED_CHANNELS[1][0]
CALC_NESTED_L3 = CALC_NESTED_CHANNELS[2][0]


def _timestamps(start: sy.TimeStamp, count: int, offset: int = 1) -> NpArray:
    return np.array(
        [start + i * MS for i in range(offset, offset + count)],
        dtype=np.int64,
    )


class CalcChannelsVerify(TestCase):
    """Verify calculated channels survive migration."""

    _calc_idx: sy.Channel
    _src: sy.Channel

    def run(self) -> None:
        self._calc_idx = self.client.channels.retrieve(CALC_IDX)
        self._src = self.client.channels.retrieve(CALC_SRC_F32)
        self.test_calc_operations()
        self.test_calc_type_handling()
        self.test_calc_nested()
        self.test_calc_windowed()

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
        for name, expression, expected_type in CALC_TYPE_CHANNELS_TYPED:
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
            for name, _, _ in CALC_TYPE_CHANNELS_TYPED
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
            [calc_cos.key, calc_cos.index, calc_quad.key],
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
