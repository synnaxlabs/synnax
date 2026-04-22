#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: OPC UA task configuration."""

import synnax as sy

SETUP_VERSION = "0.52"

OPC_UA_PORT = 4841
TASK_NAME = "mig_opc_read"
IDX_NAME = "mig_opc_idx"
CHANNEL_PREFIX = "mig_opc_float"
NUM_CHANNELS = 2

if __name__ == "__main__":
    from examples.opcua.server import OPCUASim
    from setup import kill_port, log, run

    def setup(client: sy.Synnax) -> None:
        kill_port(OPC_UA_PORT)
        log("  [task_opc] Retrieving embedded rack...")
        rack = client.racks.retrieve(name="Node 1 Embedded Driver")
        rack_key = rack.key

        log("  [task_opc] Creating device...")
        opc_device = OPCUASim.create_device(rack_key)
        client.devices.create(opc_device)

        log("  [task_opc] Creating channels and task config...")
        opc_idx = client.channels.create(
            name=IDX_NAME,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        opc_channels = []
        for i in range(NUM_CHANNELS):
            ch_key = int(
                client.channels.create(
                    name=f"{CHANNEL_PREFIX}_{i}",
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

    run(setup)
