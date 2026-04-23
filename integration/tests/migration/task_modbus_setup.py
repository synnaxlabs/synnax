#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: Modbus task configuration."""

from collections.abc import Callable

import synnax as sy

SETUP_VERSION = "0.52"

MODBUS_PORT = 5020
TASK_NAME = "mig_modbus_read"
IDX_NAME = "mig_modbus_idx"
CHANNEL_PREFIX = "mig_modbus_reg"
NUM_CHANNELS = 2

MODBUS_HOST = "127.0.0.1"
MODBUS_DEVICE_NAME = "Modbus TCP Test Server"


def setup(client: sy.Synnax, log: Callable[[str], None]) -> None:
    log("  [task_modbus] Retrieving embedded rack...")
    rack = client.racks.retrieve(name="Node 1 Embedded Driver")
    rack_key = rack.key

    log("  [task_modbus] Creating device...")
    modbus_device = client.devices.create(
        key="mig-modbus-device",
        name=MODBUS_DEVICE_NAME,
        make="Modbus",
        model="Modbus",
        location=f"{MODBUS_HOST}:{MODBUS_PORT}",
        rack=rack_key,
        configured=False,
        properties={
            "connection": {
                "host": MODBUS_HOST,
                "port": MODBUS_PORT,
                "swap_bytes": False,
                "swap_words": False,
            },
            "read": {"index": 0, "channels": {}},
            "write": {"channels": {}},
        },
    )

    log("  [task_modbus] Creating channels and task config...")
    modbus_idx = client.channels.create(
        name=IDX_NAME,
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )
    data_channels = client.channels.create(
        [
            sy.Channel(
                name=f"{CHANNEL_PREFIX}_{i}",
                data_type=sy.DataType.FLOAT32,
                index=modbus_idx.key,
            )
            for i in range(NUM_CHANNELS)
        ],
        retrieve_if_name_exists=True,
    )
    modbus_channels = [
        sy.modbus.HoldingRegisterInputChan(
            channel=int(ch.key),
            address=i,
            data_type="float32",
        )
        for i, ch in enumerate(data_channels)
    ]
    modbus_task = sy.modbus.ReadTask(
        name=TASK_NAME,
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


if __name__ == "__main__":
    from setup import run

    run(setup)
