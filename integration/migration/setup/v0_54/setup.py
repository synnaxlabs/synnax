#!/usr/bin/env python3

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""v0.54 migration setup script.

Standalone script that creates test resources against a running Synnax Core.
Uses only synnax + stdlib. Once committed, this file is never modified.

Usage:
    python setup.py
"""

import sys
from datetime import datetime

import numpy as np

import synnax as sy
from examples.modbus.server import ModbusSim
from examples.opcua.server import OPCUASim

HOST = "localhost"
PORT = 9090
USERNAME = "synnax"
PASSWORD = "seldon"

S = sy.TimeSpan.SECOND
MS = sy.TimeSpan.MILLISECOND


def log(msg: str) -> None:
    ts = datetime.now().strftime("%H:%M:%S.%f")[:-4]
    print(f"{ts} | {msg}")


def setup_channels(client: sy.Synnax) -> None:
    """Create typed data channels and write known sample data."""
    log("  [channels] Creating index and data channels...")

    idx = client.channels.create(
        name="mig_channels_idx",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )

    F32 = np.finfo(np.float32)
    F64 = np.finfo(np.float64)

    channels_spec: list[tuple[str, sy.DataType, np.ndarray]] = [
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

    data_channels = []
    for name, data_type, _ in channels_spec:
        ch = client.channels.create(
            name=name,
            data_type=data_type,
            index=idx.key,
            retrieve_if_name_exists=True,
        )
        data_channels.append(ch)

    log("  [channels] Writing sample data...")
    sample_count = len(channels_spec[0][2])
    start = sy.TimeStamp(200 * S)
    timestamps = np.array(
        [start + i * S for i in range(sample_count)],
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
        for ch, (_, _, expected) in zip(data_channels, channels_spec):
            payload[ch.key] = expected
        writer.write(payload)


def setup_calc_channels(client: sy.Synnax) -> None:
    """Create calculated channels with operations, expressions, and windowed data."""
    log("  [calc] Creating source channels...")

    CALC_IDX = "mig_calc_idx"
    CALC_SRC_F32 = "mig_calc_src_f32"
    CALC_SRC_F32_B = "mig_calc_src_f32_b"
    CALC_SRC_F64 = "mig_calc_src_f64"
    CALC_SRC_I64 = "mig_calc_src_i64"
    CALC_RESET = "mig_calc_reset"

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

    log("  [calc] Creating operation calc channels...")
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

    CALC_EXPR = f"return {CALC_SRC_F32} * 2 + 5"
    CALC_EXPR_OP_CHANNELS: list[tuple[str, str]] = [
        ("mig_calc_expr_avg", "avg"),
        ("mig_calc_expr_min", "min"),
        ("mig_calc_expr_max", "max"),
    ]
    for name, op_type in CALC_EXPR_OP_CHANNELS:
        client.channels.create(
            name=name,
            data_type=sy.DataType.FLOAT32,
            expression=CALC_EXPR,
            operations=[sy.channel.Operation(type=op_type)],
            retrieve_if_name_exists=True,
        )

    log("  [calc] Creating typed expression channels and writing data...")
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

    CALC_TYPE_CHANNELS: list[tuple[str, str]] = [
        ("mig_calc_complex", f"return ({CALC_SRC_F32} - 32) * 5 / 9"),
        ("mig_calc_two_f32", f"return {CALC_SRC_F32} + {CALC_SRC_F32_B}"),
        ("mig_calc_f64_mul", f"return {CALC_SRC_F64} * 3.14159"),
        ("mig_calc_i64_add", f"return {CALC_SRC_I64} + 100"),
    ]
    for name, expression in CALC_TYPE_CHANNELS:
        client.channels.create(
            name=name,
            expression=expression,
            retrieve_if_name_exists=True,
        )

    CALC_F32_DATA = np.array([10.0, 20.0, 30.0, 50.0, 100.0], dtype=np.float32)
    CALC_F32_B_DATA = np.array([5.0, 15.0, 25.0, 35.0, 45.0], dtype=np.float32)
    CALC_F64_DATA = np.array([100.0, 200.0, 300.0, 500.0, 1000.0], dtype=np.float64)
    CALC_I64_DATA = np.array([1000, 2000, 3000, 5000, 10000], dtype=np.int64)

    start = sy.TimeStamp(100 * S)
    timestamps = np.array(
        [start + i * MS for i in range(1, 1 + len(CALC_F32_DATA))],
        dtype=np.int64,
    )
    with client.open_writer(
        start=start,
        channels=[calc_idx.key, src_f32.key, src_f32_b.key, src_f64.key, src_i64.key],
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
    CALC_NESTED_CHANNELS: list[tuple[str, str]] = [
        ("mig_calc_nested_l1", f"return {CALC_SRC_F32} * 3"),
        ("mig_calc_nested_l2", "return mig_calc_nested_l1 + 100"),
        ("mig_calc_nested_l3", "return mig_calc_nested_l2 / 2"),
    ]
    for name, expression in CALC_NESTED_CHANNELS:
        client.channels.create(
            name=name,
            expression=expression,
            retrieve_if_name_exists=True,
        )

    log("  [calc] Creating windowed calc channels and writing data...")
    WIN_IDX = "mig_win_idx"
    WIN_SRC_COS = "mig_win_src_cos"
    WIN_SRC_QUAD = "mig_win_src_quad"
    WIN_NUM_DOMAINS = 300
    WIN_DOMAIN_GAP_S = 0.05
    WIN_SAMPLES_PER_DOMAIN = 40
    WIN_DT_MS = 1
    WIN_WINDOW_S = 0.02
    WIN_NOISE_STD = 0.1

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
            operations=[sy.channel.Operation(type="avg", duration=WIN_WINDOW_S * S)],
            retrieve_if_name_exists=True,
        )

    def _win_value(seed: str, d: int) -> float:
        t = d * WIN_DOMAIN_GAP_S
        if seed == "cosine":
            return float(np.cos(2 * np.pi * t / 2.5))
        return float(t**2)

    def _win_noisy_data(seed: str, d: int) -> np.ndarray:
        rng = np.random.default_rng(seed=hash((seed, d)) & 0xFFFFFFFF)
        center = _win_value(seed, d)
        return (center + rng.normal(0, WIN_NOISE_STD, WIN_SAMPLES_PER_DOMAIN)).astype(
            np.float32
        )

    base = sy.TimeStamp(100 * S) + 10 * S
    write_keys = [win_idx.key, src_cos.key, src_quad.key]

    for d in range(WIN_NUM_DOMAINS):
        domain_start = base + int(d * WIN_DOMAIN_GAP_S * 1000) * MS
        timestamps = np.array(
            [domain_start + i * WIN_DT_MS * MS for i in range(WIN_SAMPLES_PER_DOMAIN)],
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


def setup_rbac(client: sy.Synnax) -> None:
    """Create custom role, users, and assign roles."""
    log("  [rbac] Creating custom role...")

    CUSTOM_ROLE_NAME = "mig_rbac_role"
    PASSWORD = "mig_rbac_pass123"

    role = client.access.roles.create(
        sy.Role(
            name=CUSTOM_ROLE_NAME,
            description="Custom role for migration testing",
        )
    )

    users_spec = [
        ("mig_rbac_custom", "MigCustom", "RbacUser", CUSTOM_ROLE_NAME),
        ("mig_rbac_operator", "MigOperator", "RbacUser", "Operator"),
        ("mig_rbac_viewer", "MigViewer", "RbacUser", "Viewer"),
    ]

    log("  [rbac] Creating users...")
    for username, first_name, last_name, _ in users_spec:
        client.users.create(
            username=username,
            password=PASSWORD,
            first_name=first_name,
            last_name=last_name,
        )

    log("  [rbac] Assigning roles...")
    internal = client.access.roles.retrieve(internal=True)
    builtin_by_name = {r.name: r for r in internal}

    for username, _, _, role_name in users_spec:
        user = client.users.retrieve(username=username)
        if role_name == CUSTOM_ROLE_NAME:
            role_key = role.key
        else:
            role_key = builtin_by_name[role_name].key
        client.access.roles.assign(user=user.key, role=role_key)


def setup_tasks(client: sy.Synnax) -> None:
    """Create OPC UA, Modbus, and NI task configurations.

    Starts simulators for OPC UA and Modbus so that tasks.configure can
    validate the connection, matching the integration test pattern.
    NI tasks use tasks.create since there is no NI simulator on macOS/Linux.
    """
    log("  [tasks] Retrieving embedded rack...")
    rack = client.racks.retrieve(name="Node 1 Embedded Driver")
    rack_key = rack.key

    log("  [tasks] Creating devices...")

    opc_device = OPCUASim.create_device(rack_key)
    client.devices.create(opc_device)

    modbus_device = ModbusSim.create_device(rack_key)
    client.devices.create(modbus_device)

    # ni_device = client.devices.create(
    #     key="mig-ni-device",
    #     name="NI Migration Device",
    #     make="ni",
    #     model="9205",
    #     location="E101Mod4",
    #     rack=rack_key,
    #     properties={"isAnalog": True, "isChassis": False},
    # )

    log("  [tasks] Creating OPC UA task channels and config...")
    opc_idx = client.channels.create(
        name="mig_opc_idx",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )
    opc_channels = []
    for i in range(2):
        ch_key = int(
            client.channels.create(
                name=f"mig_opc_float_{i}",
                data_type=sy.DataType.FLOAT32,
                index=opc_idx.key,
                retrieve_if_name_exists=True,
            ).key
        )
        opc_channels.append(
            sy.opcua.ReadChannel(
                channel=ch_key,
                node_id=f"NS=2;I={8 + i}",
                data_type="float32",
            )
        )
    opc_task = sy.opcua.ReadTask(
        name="mig_opc_read",
        device=opc_device.key,
        sample_rate=50 * sy.Rate.HZ,
        stream_rate=10 * sy.Rate.HZ,
        data_saving=True,
        channels=opc_channels,
    )
    opc_pld = opc_task.to_payload()
    client.tasks.create(
        name=opc_pld.name,
        type=opc_pld.type,
        config=opc_pld.config,
        rack=rack_key,
    )

    log("  [tasks] Creating Modbus task channels and config...")
    modbus_idx = client.channels.create(
        name="mig_modbus_idx",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )
    modbus_channels = []
    for i in range(2):
        ch_key = int(
            client.channels.create(
                name=f"mig_modbus_reg_{i}",
                data_type=sy.DataType.FLOAT32,
                index=modbus_idx.key,
                retrieve_if_name_exists=True,
            ).key
        )
        modbus_channels.append(
            sy.modbus.HoldingRegisterInputChan(
                channel=ch_key,
                address=i,
                data_type="float32",
            )
        )
    modbus_task = sy.modbus.ReadTask(
        name="mig_modbus_read",
        device=modbus_device.key,
        sample_rate=50 * sy.Rate.HZ,
        stream_rate=10 * sy.Rate.HZ,
        data_saving=True,
        channels=modbus_channels,
    )
    modbus_pld = modbus_task.to_payload()
    client.tasks.create(
        name=modbus_pld.name,
        type=modbus_pld.type,
        config=modbus_pld.config,
        rack=rack_key,
    )

    # log("  [tasks] Creating NI task channels and config...")
    # ni_idx = client.channels.create(
    #     name="mig_ni_idx",
    #     data_type=sy.DataType.TIMESTAMP,
    #     is_index=True,
    #     retrieve_if_name_exists=True,
    # )
    # ni_channels = []
    # for i in range(2):
    #     ch_key = int(
    #         client.channels.create(
    #             name=f"mig_ni_voltage_{i}",
    #             data_type=sy.DataType.FLOAT32,
    #             index=ni_idx.key,
    #             retrieve_if_name_exists=True,
    #         ).key
    #     )
    #     ni_channels.append(
    #         sy.ni.AIVoltageChan(
    #             port=i,
    #             channel=ch_key,
    #             terminal_config="Cfg_Default",
    #             min_val=-10.0,
    #             max_val=10.0,
    #         )
    #     )
    # ni_task = sy.ni.AnalogReadTask(
    #     name="mig_ni_analog_read",
    #     device=ni_device.key,
    #     sample_rate=50 * sy.Rate.HZ,
    #     stream_rate=10 * sy.Rate.HZ,
    #     data_saving=True,
    #     channels=ni_channels,
    # )
    # client.tasks.create(ni_task, rack=rack_key)



def main() -> None:
    log(f"Connecting to Synnax at {HOST}:{PORT}...")
    client = sy.Synnax(
        host=HOST,
        port=PORT,
        username=USERNAME,
        password=PASSWORD,
    )

    steps = [
        ("Channels", setup_channels),
        ("Calculated Channels", setup_calc_channels),
        ("RBAC", setup_rbac),
        ("Tasks", setup_tasks),
    ]

    for name, func in steps:
        log(f"--- {name} ---")
        try:
            func(client)
        except Exception as e:
            log(f"FAILED: {name}: {e}", file=sys.stderr)
            sys.exit(1)


if __name__ == "__main__":
    main()
