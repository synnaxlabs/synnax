#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from examples.simulators import PressSimDAQ

from framework.utils import get_random_name
from tests.arc.arc_case import ArcConsoleCase

ARC_LOW_PRIORITY_SOURCE = """
authority 100

start_low_cmd => main

sequence main {
    stage active {
        0 -> press_vlv_cmd,
        wait{duration=100ms} => active
    }
}
"""

ARC_HIGH_PRIORITY_SOURCE = """
authority 200

start_high_cmd => main

sequence main {
    stage active {
        1 -> press_vlv_cmd,
        wait{duration=100ms} => active_hold
    }
    stage active_hold {
        1 -> press_vlv_cmd,
        wait{duration=5s} => done
    }
    stage done {
        set_authority{value=0}
    }
}
"""


class ArcAuthorityArcVsArc(ArcConsoleCase):
    """Test that a higher-authority Arc program wins over a lower one,
    and the lower program resumes when the higher one stops writing."""

    arc_source = ARC_LOW_PRIORITY_SOURCE
    arc_name_prefix = "ArcLow"
    start_cmd_channel = "start_low_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ["press_vlv_state", "press_pt", "end_test_cmd"]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self.arc_b_name = f"ArcHigh_{get_random_name()}"
        self._arc_b_created = False
        self._arc_b_started = False
        self.client.channels.create(
            name="start_high_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        # Arc A is already running (handled by ArcConsoleCase.run())
        self.log("Waiting for Arc A to control (press_vlv_state == 0)...")
        self.wait_for_eq("press_vlv_state", 0)

        # Create and start Arc B (high priority, writes 1)
        self.log(f"Creating Arc B (high priority): {self.arc_b_name}")
        self.console.arc.create(self.arc_b_name, ARC_HIGH_PRIORITY_SOURCE, mode="Text")
        self._arc_b_created = True
        assert self.rack is not None
        self.log("Selecting rack for Arc B")
        self.console.arc.select_rack(self.rack.name)
        self.log("Configuring Arc B")
        self.console.arc.configure()
        self.log("Starting Arc B")
        self.console.arc.start()
        self._arc_b_started = True

        self.log("Triggering Arc B sequence")
        with self.client.open_writer(sy.TimeStamp.now(), "start_high_cmd") as w:
            w.write("start_high_cmd", 1)

        # Wait for Arc B to override (valve open = 1)
        self.log("Waiting for Arc B to override (press_vlv_state == 1)...")
        self.wait_for_eq("press_vlv_state", 1)

        # Wait for Arc B to reach done stage (~5s), Arc A resumes
        self.log("Waiting for Arc B done stage, Arc A should resume...")
        self.wait_for_eq("press_vlv_state", 0, timeout=10)

    def teardown(self) -> None:
        if self._arc_b_started and self.console.arc.is_running():
            self.log("Stopping Arc B")
            try:
                self.console.arc.stop()
            except Exception as e:
                self.fail(f"Failed to stop Arc B: {e}")

        if self._arc_b_created:
            self.log(f"Deleting Arc B: {self.arc_b_name}")
            try:
                self.console.arc.delete(self.arc_b_name)
            except Exception as e:
                self.fail(f"Failed to delete Arc B: {e}")

        super().teardown()
