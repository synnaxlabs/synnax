#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI write task integration tests."""

import synnax as sy

from tests.driver.ni_task import NIDigitalWriteTaskCase
from tests.driver.task import (
    create_channel,
    create_index,
    send_and_verify_commands,
    _assert_no_task_errors,
)


def _do_channels(
    client: sy.Synnax, devices: dict[str, sy.Device]
) -> list[sy.ni.DOChan]:
    """Create two digital output channels on port 0, lines 0 and 1."""
    cmd_idx = create_index(client, "ni_do_cmd_time")
    state_idx = create_index(client, "ni_do_state_time")
    return [
        sy.ni.DOChan(
            cmd_channel=create_channel(
                client,
                name=f"ni_do_cmd_{i}",
                data_type=sy.DataType.UINT8,
                index=cmd_idx.key,
            ),
            state_channel=create_channel(
                client,
                name=f"ni_do_state_{i}",
                data_type=sy.DataType.UINT8,
                index=state_idx.key,
            ),
            port=0,
            line=i,
        )
        for i in range(2)
    ]


class NIDigitalWrite(NIDigitalWriteTaskCase):
    """Write valid digital output (0/1) on NI device port 0, lines 0 and 1."""

    task_name = "NI Digital Write"
    device_locations = ["SYMod1"]
    command_values = [[1, 0], [0, 1]]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.DOChan]:
        return _do_channels(client, devices)


class NIDigitalWriteInvalidData(NIDigitalWriteTaskCase):
    """Write invalid digital data (42) and verify the driver reports an error."""

    task_name = "NI Digital Write Invalid"
    device_locations = ["SYMod1"]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.DOChan]:
        return _do_channels(client, devices)

    def run(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Send invalid digital values (42)")
        with self.tsk.run():
            cmd_keys = self._channel_keys(self.tsk)
            send_and_verify_commands(
                self.client,
                cmd_keys=cmd_keys,
                writer_name=f"{self.task_name}_test_writer",
                task_name=self.tsk.name,
                command_values=[[42, 42], [100, 100]],
            )
            try:
                _assert_no_task_errors(
                    self.client, self.tsk.key, task_name=self.tsk.name
                )
            except AssertionError:
                self.log("Driver correctly rejected invalid digital data")
                return
        self.fail("Driver did not report an error for invalid digital data (42)")
