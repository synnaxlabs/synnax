#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

ARC_WRITE_CASCADING_SOURCE = """
authority 200

sequence main {
    1 -> cascade_a
    1 -> cascade_b
    1 -> cascade_c
    wait{1s}
    0 -> cascade_a
    0 -> cascade_b
    0 -> cascade_c
}

cascade_start_cmd => main
"""


class WriteCascading(ArcConsoleCase):
    """Consecutive writes in a stageless sequence cascade through on a single
    tick.

    Each write step is immediately truthy, so the sequence advances past it
    on the same scheduler cycle. Three consecutive writes therefore land
    together before the runtime moves on to the wait gate. After the gate
    elapses, the next three writes also cascade through together to flip
    all three channels back to 0.
    """

    arc_source = ARC_WRITE_CASCADING_SOURCE
    arc_name_prefix = "ArcWriteCascading"
    start_cmd_channel = "cascade_start_cmd"
    subscribe_channels = [
        "cascade_a",
        "cascade_b",
        "cascade_c",
    ]

    def setup(self) -> None:
        create_virtual_channel(self.client, "cascade_a", sy.DataType.UINT8)
        create_virtual_channel(self.client, "cascade_b", sy.DataType.UINT8)
        create_virtual_channel(self.client, "cascade_c", sy.DataType.UINT8)
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for all three channels to read 1 (initial cascade)...")
        self.wait_for_eq(
            "cascade_a", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.wait_for_eq(
            "cascade_b", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.wait_for_eq(
            "cascade_c", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.log("All three channels at 1; sequence cascaded through writes")

        self.log("Waiting for all three channels to read 0 (cascade after wait)...")
        self.wait_for_eq(
            "cascade_a", 0, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.wait_for_eq(
            "cascade_b", 0, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.wait_for_eq(
            "cascade_c", 0, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.log(
            "All three channels at 0; second cascade fired together after wait"
        )
