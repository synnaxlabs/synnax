#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: NI analog read task configuration.

Skips on non-Windows platforms (no NI-DAQmx simulator available).
"""

from collections.abc import Callable

import synnax as sy

SETUP_VERSION = "0.52"

TASK_NAME = "mig_ni_analog_read"
IDX_NAME = "mig_ni_idx"
CHANNEL_PREFIX = "mig_ni_voltage"
NUM_CHANNELS = 2
DEVICE_LOCATION = "E101Mod4"

import platform


def setup(client: sy.Synnax, log: Callable[[str], None]) -> None:
    if platform.system() != "Windows":
        log("  [task_ni] Skipping — requires Windows")
        return

    log("  [task_ni] Retrieving embedded rack...")
    rack = client.racks.retrieve(name="Node 1 Embedded Driver")
    rack_key = rack.key

    log("  [task_ni] Creating device...")
    ni_device = client.devices.create(
        key="mig-ni-device",
        name="NI Migration Device",
        make="ni",
        model="9205",
        location=DEVICE_LOCATION,
        rack=rack_key,
        properties={"isAnalog": True, "isChassis": False},
    )

    log("  [task_ni] Creating channels and task config...")
    ni_idx = client.channels.create(
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
                index=ni_idx.key,
            )
            for i in range(NUM_CHANNELS)
        ],
        retrieve_if_name_exists=True,
    )
    ni_channels = [
        sy.ni.AIVoltageChan(
            port=i,
            channel=int(ch.key),
            terminal_config="Cfg_Default",
            min_val=-10.0,
            max_val=10.0,
        )
        for i, ch in enumerate(data_channels)
    ]
    ni_task = sy.ni.AnalogReadTask(
        name=TASK_NAME,
        device=ni_device.key,
        sample_rate=50 * sy.Rate.HZ,
        stream_rate=10 * sy.Rate.HZ,
        data_saving=True,
        channels=ni_channels,
    )
    ni_pld = ni_task.to_payload()
    client.tasks.create(
        name=ni_pld.name,
        type=ni_pld.type,
        config=ni_pld.config,
        rack=rack_key,
    )


if __name__ == "__main__":
    from setup import run

    run(setup)
