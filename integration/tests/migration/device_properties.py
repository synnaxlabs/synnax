#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""SY-4047: Device properties migration tests.

Reproduces the bug where clicking Configure on an NI task fails with a ZodError
because ``propertiesZ`` rejects incomplete device properties stored by older versions.

Sequence:
    1. ``DevicePropertiesSetup`` creates an NI analog read task on sim hardware,
       then strips the device properties to simulate pre-v0.54 stored data.
    2. Core/Console is upgraded to the target version.
    3. ``DevicePropertiesConsoleVerify`` opens the task in the Console, clicks
       Configure, and verifies the play button appears (configure succeeded).
    4. ``DevicePropertiesVerify`` retrieves the task via Python client, runs it,
       and verifies data is produced using ``assert_sample_count``.
"""

import synnax as sy
from tests.driver.ni_task import NIAnalogReadTaskCase
from tests.driver.task import create_channel, create_index
from tests.migration.task import (
    ReadTaskConsoleVerify,
    ReadTaskMigration,
    ReadTaskMigrationSetup,
    ReadTaskMigrationVerify,
)

TASK_NAME = "mig_sy4047_ni_read"
IDX_NAME = "mig_sy4047_idx"
CHANNEL_PREFIX = "mig_sy4047_voltage"
NUM_CHANNELS = 2
DEVICE_LOCATION = "E101Mod4"  # NI 9205 sim


class _Base(ReadTaskMigration, NIAnalogReadTaskCase):
    """Shared base for SY-4047 device properties migration tests."""

    task_name = TASK_NAME
    device_locations = [DEVICE_LOCATION]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIVoltageChan]:
        idx = create_index(client, IDX_NAME)
        return [
            sy.ni.AIVoltageChan(
                port=i,
                channel=create_channel(
                    client,
                    name=f"{CHANNEL_PREFIX}_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                min_val=-10.0,
                max_val=10.0,
            )
            for i in range(NUM_CHANNELS)
        ]


class DevicePropertiesSetup(ReadTaskMigrationSetup, _Base):
    """Create an NI analog read task, verify it runs, then strip device properties.

    After normal setup (create + configure + start/stop), overwrites the device
    properties to remove ``counterInput`` and make ``analogOutput`` partial —
    simulating what a pre-v0.54 device would store.
    """

    def run(self) -> None:
        super().run()
        client = self.client
        dev = client.devices.retrieve(name=self.device_name)
        self.log(f"Stripping properties for '{dev.name}' (key={dev.key})")
        self.log(f"  Original keys: {list(dev.properties.keys())}")

        stripped = dict(dev.properties)
        stripped.pop("counterInput", None)
        stripped.pop("counter_input", None)
        for key in ("analogOutput", "analog_output"):
            if key in stripped and isinstance(stripped[key], dict):
                port_key = "portCount" if key == "analogOutput" else "port_count"
                stripped[key] = {port_key: stripped[key].get(port_key, 0)}

        dev.properties = stripped
        client.devices.create(dev)

        dev = client.devices.retrieve(key=dev.key)
        self.log(f"  Stripped keys: {list(dev.properties.keys())}")
        has_counter = (
            "counterInput" in dev.properties or "counter_input" in dev.properties
        )
        assert not has_counter, "counterInput should have been stripped"
        self.log("Device properties stripped successfully")


class DevicePropertiesVerify(ReadTaskMigrationVerify, _Base):
    """Retrieve the task, verify config survived, and run to verify data."""

    task_type = "ni_analog_read"
    task_class = sy.ni.AnalogReadTask
    channel_prefix = CHANNEL_PREFIX
    num_channels = NUM_CHANNELS


class DevicePropertiesConsoleVerify(ReadTaskConsoleVerify):
    """Verify the task form renders and Configure works in the Console."""

    task_name = TASK_NAME
    expected_channels = [f"{CHANNEL_PREFIX}_{i}" for i in range(NUM_CHANNELS)]
    requires_platform = "windows"
