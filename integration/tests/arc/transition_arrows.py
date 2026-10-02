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
from tests.arc.arc import ArcCase

ARC_TRANSITION_ARROWS_SOURCE = """
authority 200

sequence main {
    stage armed {
        1 -> ta_state
        ta_abort_cmd == 1 => abort
    }
}

sequence abort {
    2 -> ta_state
}

ta_start_cmd == 1 => main
"""


class TransitionArrows(ArcCase):
    """A false condition must not start a top-level sequence through `=>`.

    1. Write ``ta_start_cmd = 0``: main must not start.
    2. Write ``ta_start_cmd = 1``: main starts and writes ``ta_state = 1``.
    3. Write ``ta_abort_cmd = 0``: abort must not start.
    4. Write ``ta_abort_cmd = 1``: abort starts and writes ``ta_state = 2``.
    """

    arc_source = ARC_TRANSITION_ARROWS_SOURCE
    arc_name_prefix = "ArcTransitionArrows"
    start_cmd_channel = "ta_start_cmd"
    subscribe_channels = ["ta_state"]

    def setup(self) -> None:
        create_virtual_channel(self.client, "ta_abort_cmd", sy.DataType.UINT8)
        create_virtual_channel(self.client, "ta_state", sy.DataType.UINT8)
        super().setup()
        self.set_manual_timeout(60)

    def run(self) -> None:
        self._retrieve_rack()
        self.arc_name = self.load_arc(self.arc_source, self.arc_name_prefix)
        self.verify_sequence_execution()

    def verify_sequence_execution(self) -> None:
        self.log("Writing ta_start_cmd=0; main must not start")
        self.writer.write("ta_start_cmd", 0)
        sy.sleep(2.0)
        state = self.read_tlm("ta_state")
        if state is not None:
            self.fail(f"ta_state={state} after ta_start_cmd=0; main started on false")
            return

        self.log("Writing ta_start_cmd=1; main must start")
        self.writer.write("ta_start_cmd", 1)
        self.wait_for_eq("ta_state", 1, timeout=5 * sy.TimeSpan.SECOND)

        self.log("Writing ta_abort_cmd=0; abort must not start")
        self.writer.write("ta_abort_cmd", 0)
        sy.sleep(2.0)
        state = self.read_tlm("ta_state")
        if state != 1:
            self.fail(f"ta_state={state} after ta_abort_cmd=0; abort started on false")
            return

        self.log("Writing ta_abort_cmd=1; abort must start")
        self.writer.write("ta_abort_cmd", 1)
        self.wait_for_eq("ta_state", 2, timeout=5 * sy.TimeSpan.SECOND)
