#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: typed data channels with known sample data."""

from collections.abc import Callable

import numpy as np

import synnax as sy

SETUP_VERSION = "0.49"

IDX_NAME = "mig_channels_idx"

F32 = np.finfo(np.float32)
F64 = np.finfo(np.float64)

DATA_CHANNELS: list[tuple[str, sy.DataType, np.ndarray]] = [
    (
        "mig_ch_float32",
        sy.DataType.FLOAT32,
        np.array(
            [
                0.0,
                -0.0,
                1.0,
                -1.0,
                F32.max,
                F32.min,
                F32.tiny,
                -F32.tiny,
                F32.eps,
                3.1415927,
                -2.7182818,
                0.000031416,
                -9.80665,
                1.23456e20,
                -7.891011e-12,
                4.56789e37,
                -1.17549e-38,
            ],
            dtype=np.float32,
        ),
    ),
    (
        "mig_ch_float64",
        sy.DataType.FLOAT64,
        np.array(
            [
                0.0,
                -0.0,
                1.0,
                -1.0,
                F64.max,
                F64.min,
                F64.tiny,
                -F64.tiny,
                F64.eps,
                3.141592653589793,
                -2.718281828459045,
                0.00003141592653589793,
                1.2345678901234567e150,
                -9.876543210987654e-150,
                1.7976931348623155e308,
                -2.2250738585072014e-308,
                -9.80665,
            ],
            dtype=np.float64,
        ),
    ),
    (
        "mig_ch_int8",
        sy.DataType.INT8,
        np.array(
            [
                -128,
                -73,
                -50,
                -25,
                -1,
                0,
                1,
                25,
                42,
                50,
                73,
                99,
                100,
                110,
                120,
                126,
                127,
            ],
            dtype=np.int8,
        ),
    ),
    (
        "mig_ch_int16",
        sy.DataType.INT16,
        np.array(
            [
                -32768,
                -12345,
                -5000,
                -500,
                -1,
                0,
                1,
                500,
                5000,
                9999,
                12345,
                20000,
                25000,
                30000,
                31000,
                32000,
                32767,
            ],
            dtype=np.int16,
        ),
    ),
    (
        "mig_ch_int32",
        sy.DataType.INT32,
        np.array(
            [
                -2147483648,
                -123456789,
                -1000000,
                -1000,
                -1,
                0,
                1,
                1000,
                1000000,
                123456789,
                500000000,
                987654321,
                1000000000,
                1500000000,
                1900000000,
                2000000000,
                2147483647,
            ],
            dtype=np.int32,
        ),
    ),
    (
        "mig_ch_int64",
        sy.DataType.INT64,
        np.array(
            [
                np.iinfo(np.int64).min,
                -1234567890123456789,
                -999999999999,
                -1000000,
                -1,
                0,
                1,
                1000000,
                999999999999,
                1234567890123456789,
                2000000000000000000,
                3000000000000000000,
                4000000000000000000,
                5000000000000000000,
                6000000000000000000,
                7223372036854775807,
                np.iinfo(np.int64).max,
            ],
            dtype=np.int64,
        ),
    ),
    (
        "mig_ch_uint8",
        sy.DataType.UINT8,
        np.array(
            [
                0,
                1,
                10,
                25,
                50,
                73,
                100,
                128,
                150,
                175,
                199,
                200,
                220,
                240,
                250,
                254,
                255,
            ],
            dtype=np.uint8,
        ),
    ),
    (
        "mig_ch_uint16",
        sy.DataType.UINT16,
        np.array(
            [
                0,
                1,
                100,
                500,
                1000,
                5000,
                12345,
                20000,
                32768,
                40000,
                50000,
                54321,
                60000,
                63000,
                64000,
                65534,
                65535,
            ],
            dtype=np.uint16,
        ),
    ),
    (
        "mig_ch_uint32",
        sy.DataType.UINT32,
        np.array(
            [
                0,
                1,
                1000,
                100000,
                1000000,
                123456789,
                500000000,
                1000000000,
                2000000000,
                2147483648,
                3000000000,
                3141592653,
                3500000000,
                4000000000,
                4200000000,
                4294967294,
                4294967295,
            ],
            dtype=np.uint32,
        ),
    ),
    (
        "mig_ch_uint64",
        sy.DataType.UINT64,
        np.array(
            [
                0,
                1,
                1000000,
                1000000000,
                1234567890123456789,
                2**32,
                2**40,
                2**48,
                2**56,
                2**63,
                10000000000000000000,
                12000000000000000000,
                14000000000000000000,
                16000000000000000000,
                9876543210987654321,
                18000000000000000000,
                np.iinfo(np.uint64).max,
            ],
            dtype=np.uint64,
        ),
    ),
]


def setup(client: sy.Synnax, log: Callable[[str], None]) -> None:
    log("  [channels] Creating index and data channels...")

    idx = client.channels.create(
        name=IDX_NAME,
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )

    data_channels = client.channels.create(
        [
            sy.Channel(name=name, data_type=data_type, index=idx.key)
            for name, data_type, _ in DATA_CHANNELS
        ],
        retrieve_if_name_exists=True,
    )

    log("  [channels] Writing sample data...")
    sample_count = len(DATA_CHANNELS[0][2])
    start = sy.TimeStamp(200 * sy.TimeSpan.SECOND)
    timestamps = np.array(
        [start + i * sy.TimeSpan.SECOND for i in range(sample_count)],
        dtype=np.int64,
    )
    channel_keys = [idx.key] + [ch.key for ch in data_channels]
    with client.open_writer(
        start=start,
        channels=channel_keys,
        name="mig_channels_writer",
        enable_auto_commit=True,
    ) as writer:
        payload: dict[int, np.ndarray] = {idx.key: timestamps}
        for ch, (_, _, expected) in zip(data_channels, DATA_CHANNELS):
            payload[ch.key] = expected
        writer.write(payload)


if __name__ == "__main__":
    from setup import run

    run(setup)
