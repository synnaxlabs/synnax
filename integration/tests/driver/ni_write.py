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
from tests.driver.task import create_channel, create_index


class NIDigitalWrite(NIDigitalWriteTaskCase):
    """Write digital output on NI device port 0, lines 0 and 1."""

    task_name = "NI Digital Write"
    device_location = "SYMod1"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.ni.DOChan]:
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
