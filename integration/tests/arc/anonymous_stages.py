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

ARC_ANONYMOUS_STAGES_SOURCE = """
authority 200

sequence main {
    stage {
        1 -> anon_cmd,
        anon_signal > 5 => next,
    }
    stage {
        0 -> anon_cmd,
    }
}

anon_start_cmd => main
"""


class AnonymousStages(ArcConsoleCase):
    """Stages declared without identifiers participate in a sequence by
    position rather than by name.

    The sequence has two anonymous stages. The first writes
    ``anon_cmd = 1`` and waits for ``anon_signal > 5`` to fire ``=> next``,
    transitioning to the second anonymous stage which writes
    ``anon_cmd = 0``. Validates the parser's optional-identifier rule and
    that the runtime can address steps by position.
    """

    arc_source = ARC_ANONYMOUS_STAGES_SOURCE
    arc_name_prefix = "ArcAnonymousStages"
    start_cmd_channel = "anon_start_cmd"
    subscribe_channels = [
        "anon_cmd",
    ]

    def setup(self) -> None:
        create_virtual_channel(self.client, "anon_cmd", sy.DataType.UINT8)
        create_virtual_channel(self.client, "anon_signal", sy.DataType.FLOAT32)
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for anon_cmd=1 (first anonymous stage entered)...")
        self.wait_for_eq(
            "anon_cmd", 1, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )

        self.log("Driving anon_signal=2 (below transition threshold)")
        self.writer.write("anon_signal", 2.0)
        sy.sleep(1.0)

        self.log("Driving anon_signal=10 (above transition threshold)")
        self.writer.write("anon_signal", 10.0)
        self.log("Waiting for anon_cmd=0 (second anonymous stage entered)...")
        self.wait_for_eq(
            "anon_cmd", 0, timeout=5 * sy.TimeSpan.SECOND, is_virtual=True
        )
        self.log("Sequence transitioned through both anonymous stages")
