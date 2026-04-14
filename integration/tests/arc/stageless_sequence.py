#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from tests.arc.arc_case import ArcConsoleCase

ARC_STAGELESS_SOURCE = """
authority 200

sequence main {
    1 -> stageless_valve
    wait{2s}
    0 -> stageless_valve
}

stageless_start_cmd => main
"""


class StagelessSequence(ArcConsoleCase):
    """A bare sequence body with no stages.

    Validates that the parser accepts no-comma-separated items in a sequence,
    that writes execute as immediately-truthy steps that advance, and that a
    bare ``wait{}`` blocks progression until the duration elapses.

    Sequence:
      1. Write ``stageless_valve = 1`` (immediate)
      2. ``wait{2s}`` blocks for ~2 seconds
      3. Write ``stageless_valve = 0``
    """

    arc_source = ARC_STAGELESS_SOURCE
    arc_name_prefix = "ArcStagelessSequence"
    start_cmd_channel = "stageless_start_cmd"
    subscribe_channels = [
        "stageless_valve",
    ]

    def setup(self) -> None:
        idx = self.client.channels.create(
            name="stageless_valve_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="stageless_valve",
            data_type=sy.DataType.UINT8,
            index=idx.key,
            retrieve_if_name_exists=True,
        )
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for stageless_valve=1 (first write of sequence)...")
        self.wait_for_eq(
            "stageless_valve", 1, timeout=5 * sy.TimeSpan.SECOND
        )
        self.log("First write observed")

        self.log("Waiting for stageless_valve=0 (after wait{2s})...")
        self.wait_for_eq(
            "stageless_valve", 0, timeout=8 * sy.TimeSpan.SECOND
        )
        self.log(
            "Second write observed — sequence advanced past wait gate "
            "and executed final write"
        )
