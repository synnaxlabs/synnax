#!/usr/bin/env python3

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: Modbus task configuration."""

from examples.modbus.server import ModbusSim
from setup import kill_port, log, run

import synnax as sy

SETUP_VERSION = "0.52"


MODBUS_PORT = 5020


def setup(client: sy.Synnax) -> None:
    kill_port(MODBUS_PORT)
    log("  [task_modbus] Retrieving embedded rack...")
    rack = client.racks.retrieve(name="Node 1 Embedded Driver")
    rack_key = rack.key

    log("  [task_modbus] Creating device...")
    modbus_device = ModbusSim.create_device(rack_key)
    client.devices.create(modbus_device)

    log("  [task_modbus] Creating channels and task config...")
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


run(setup)
