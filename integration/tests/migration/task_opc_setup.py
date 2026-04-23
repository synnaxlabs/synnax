#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: OPC UA task configuration."""

from collections.abc import Callable

import synnax as sy

SETUP_VERSION = "0.52"

TASK_NAME = "mig_opc_read"
IDX_NAME = "mig_opc_idx"
CHANNEL_PREFIX = "mig_opc_float"
NUM_CHANNELS = 2

OPC_ENDPOINT = "opc.tcp://127.0.0.1:4841/freeopcua/server/"
OPC_DEVICE_NAME = "OPC UA Server"


def setup(client: sy.Synnax, log: Callable[[str], None]) -> None:
    log("  [task_opc] Retrieving embedded rack...")
    rack = client.racks.retrieve(name="Node 1 Embedded Driver")
    rack_key = rack.key

    log("  [task_opc] Creating device...")
    opc_device = client.devices.create(
        key="mig-opc-device",
        name=OPC_DEVICE_NAME,
        make="opc",
        model="OPC UA",
        location=OPC_ENDPOINT,
        rack=rack_key,
        configured=True,
        properties={
            "version": "1.0.0",
            "connection": {
                "endpoint": OPC_ENDPOINT,
                "security_mode": "None",
                "security_policy": "None",
            },
            "read": {"indexes": [], "channels": {}},
            "write": {"channels": {}},
        },
    )

    log("  [task_opc] Creating channels and task config...")
    opc_idx = client.channels.create(
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
                index=opc_idx.key,
            )
            for i in range(NUM_CHANNELS)
        ],
        retrieve_if_name_exists=True,
    )
    opc_channels = [
        sy.opcua.ReadChannel(
            channel=int(ch.key),
            node_id=f"NS=2;I={8 + i}",
            data_type="float32",
        )
        for i, ch in enumerate(data_channels)
    ]
    opc_task = sy.opcua.ReadTask(
        name=TASK_NAME,
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


if __name__ == "__main__":
    from setup import run

    run(setup)
