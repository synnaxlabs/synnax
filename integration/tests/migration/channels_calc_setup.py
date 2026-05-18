#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: calculated channels with operations, expressions, and windows."""

from collections.abc import Callable

import numpy as np

import synnax as sy

SETUP_VERSION = "0.49"

CALC_IDX = "mig_calc_idx"
CALC_SRC_F32 = "mig_calc_src_f32"
CALC_SRC_F32_B = "mig_calc_src_f32_b"
CALC_SRC_F64 = "mig_calc_src_f64"
CALC_SRC_I64 = "mig_calc_src_i64"
CALC_RESET = "mig_calc_reset"

PASSTHROUGH_EXPR = f"return {CALC_SRC_F32}"

CALC_OP_CHANNELS: list[tuple[str, str, sy.TimeSpan, bool]] = [
    ("mig_calc_op_avg", "avg", sy.TimeSpan(0), False),
    ("mig_calc_op_min", "min", sy.TimeSpan(0), False),
    ("mig_calc_op_max", "max", sy.TimeSpan(0), False),
    ("mig_calc_op_avg_win", "avg", 5 * sy.TimeSpan.SECOND, False),
    ("mig_calc_op_min_win", "min", 10 * sy.TimeSpan.SECOND, False),
    ("mig_calc_op_max_win", "max", 15 * sy.TimeSpan.SECOND, False),
    ("mig_calc_op_avg_rst", "avg", sy.TimeSpan(0), True),
    ("mig_calc_op_min_rst", "min", sy.TimeSpan(0), True),
    ("mig_calc_op_max_rst", "max", sy.TimeSpan(0), True),
    ("mig_calc_op_avg_win_rst", "avg", 5 * sy.TimeSpan.SECOND, True),
    ("mig_calc_op_min_win_rst", "min", 10 * sy.TimeSpan.SECOND, True),
    ("mig_calc_op_max_win_rst", "max", 15 * sy.TimeSpan.SECOND, True),
]

CALC_EXPR = f"return {CALC_SRC_F32} * 2 + 5"
CALC_EXPR_OP_CHANNELS: list[tuple[str, str]] = [
    ("mig_calc_expr_avg", "avg"),
    ("mig_calc_expr_min", "min"),
    ("mig_calc_expr_max", "max"),
]

CALC_TYPE_CHANNELS: list[tuple[str, str]] = [
    ("mig_calc_complex", f"return ({CALC_SRC_F32} - 32) * 5 / 9"),
    ("mig_calc_two_f32", f"return {CALC_SRC_F32} + {CALC_SRC_F32_B}"),
    ("mig_calc_f64_mul", f"return {CALC_SRC_F64} * 3.14159"),
    ("mig_calc_i64_add", f"return {CALC_SRC_I64} + 100"),
]

CALC_NESTED_CHANNELS: list[tuple[str, str]] = [
    ("mig_calc_nested_l1", f"return {CALC_SRC_F32} * 3"),
    ("mig_calc_nested_l2", "return mig_calc_nested_l1 + 100"),
    ("mig_calc_nested_l3", "return mig_calc_nested_l2 / 2"),
]

CALC_F32_DATA = np.array([10.0, 20.0, 30.0, 50.0, 100.0], dtype=np.float32)
CALC_F32_B_DATA = np.array([5.0, 15.0, 25.0, 35.0, 45.0], dtype=np.float32)
CALC_F64_DATA = np.array([100.0, 200.0, 300.0, 500.0, 1000.0], dtype=np.float64)
CALC_I64_DATA = np.array([1000, 2000, 3000, 5000, 10000], dtype=np.int64)

WIN_IDX = "mig_win_idx"
WIN_SRC_COS = "mig_win_src_cos"
WIN_SRC_QUAD = "mig_win_src_quad"
WIN_NUM_DOMAINS = 300
WIN_DOMAIN_GAP_S = 0.05
WIN_SAMPLES_PER_DOMAIN = 40
WIN_DT_MS = 1
WIN_WINDOW_S = 0.02
WIN_NOISE_STD = 0.1


def _win_value(seed: str, d: int) -> float:
    t = d * WIN_DOMAIN_GAP_S
    if seed == "cosine":
        return float(np.cos(2 * np.pi * t / 2.5))
    return float(t**2)


def setup(client: sy.Synnax, log: Callable[[str], None]) -> None:
    log("  [calc] Creating source channels...")

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

    log("  [calc] Creating operation calc channels...")
    client.channels.create(
        [
            sy.Channel(
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
            )
            for name, op_type, duration, uses_reset in CALC_OP_CHANNELS
        ],
        retrieve_if_name_exists=True,
    )

    client.channels.create(
        [
            sy.Channel(
                name=name,
                data_type=sy.DataType.FLOAT32,
                expression=CALC_EXPR,
                operations=[sy.channel.Operation(type=op_type)],
            )
            for name, op_type in CALC_EXPR_OP_CHANNELS
        ],
        retrieve_if_name_exists=True,
    )

    log("  [calc] Creating typed expression channels and writing data...")
    src_f32 = client.channels.retrieve(CALC_SRC_F32)
    src_f32_b, src_f64, src_i64 = client.channels.create(
        [
            sy.Channel(
                name=CALC_SRC_F32_B,
                data_type=sy.DataType.FLOAT32,
                index=calc_idx.key,
            ),
            sy.Channel(
                name=CALC_SRC_F64, data_type=sy.DataType.FLOAT64, index=calc_idx.key
            ),
            sy.Channel(
                name=CALC_SRC_I64, data_type=sy.DataType.INT64, index=calc_idx.key
            ),
        ],
        retrieve_if_name_exists=True,
    )

    for name, expression in CALC_TYPE_CHANNELS:
        client.channels.create(
            name=name,
            expression=expression,
            retrieve_if_name_exists=True,
        )

    start = sy.TimeStamp(100 * sy.TimeSpan.SECOND)
    timestamps = np.array(
        [start + i * sy.TimeSpan.MILLISECOND for i in range(1, 1 + len(CALC_F32_DATA))],
        dtype=np.int64,
    )
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
                calc_idx.key: timestamps,
                src_f32.key: CALC_F32_DATA,
                src_f32_b.key: CALC_F32_B_DATA,
                src_f64.key: CALC_F64_DATA,
                src_i64.key: CALC_I64_DATA,
            }
        )

    log("  [calc] Creating nested calc chain...")
    for name, expression in CALC_NESTED_CHANNELS:
        client.channels.create(
            name=name,
            expression=expression,
            retrieve_if_name_exists=True,
        )

    log("  [calc] Creating windowed calc channels and writing data...")
    win_idx = client.channels.create(
        name=WIN_IDX,
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )
    src_cos = client.channels.create(
        name=WIN_SRC_COS,
        data_type=sy.DataType.FLOAT32,
        index=win_idx.key,
        retrieve_if_name_exists=True,
    )
    src_quad = client.channels.create(
        name=WIN_SRC_QUAD,
        data_type=sy.DataType.FLOAT32,
        index=win_idx.key,
        retrieve_if_name_exists=True,
    )

    for calc_name, src_name in [
        ("mig_win_calc_cos", WIN_SRC_COS),
        ("mig_win_calc_quad", WIN_SRC_QUAD),
    ]:
        client.channels.create(
            name=calc_name,
            data_type=sy.DataType.FLOAT32,
            expression=f"return {src_name}",
            operations=[
                sy.channel.Operation(
                    type="avg", duration=WIN_WINDOW_S * sy.TimeSpan.SECOND
                )
            ],
            retrieve_if_name_exists=True,
        )

    def _win_noisy_data(seed: str, d: int) -> np.ndarray:
        rng = np.random.default_rng(seed=hash((seed, d)) & 0xFFFFFFFF)
        center = _win_value(seed, d)
        return (center + rng.normal(0, WIN_NOISE_STD, WIN_SAMPLES_PER_DOMAIN)).astype(
            np.float32
        )

    base = sy.TimeStamp(100 * sy.TimeSpan.SECOND) + 10 * sy.TimeSpan.SECOND
    write_keys = [win_idx.key, src_cos.key, src_quad.key]

    for d in range(WIN_NUM_DOMAINS):
        domain_start = base + int(d * WIN_DOMAIN_GAP_S * 1000) * sy.TimeSpan.MILLISECOND
        timestamps = np.array(
            [
                domain_start + i * WIN_DT_MS * sy.TimeSpan.MILLISECOND
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
                    win_idx.key: timestamps,
                    src_cos.key: _win_noisy_data("cosine", d),
                    src_quad.key: _win_noisy_data("quadratic", d),
                }
            )


if __name__ == "__main__":
    from setup import run

    run(setup)
