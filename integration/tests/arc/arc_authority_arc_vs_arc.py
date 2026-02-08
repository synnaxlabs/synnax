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

from console.case import ConsoleCase
from framework.sim_daq_case import SimDaqTestCase
from framework.utils import get_random_name

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


class ArcAuthorityArcVsArc(SimDaqTestCase, ConsoleCase):
    """Test that a higher-authority Arc program wins over a lower one,
    and the lower program resumes when the higher one stops writing."""

    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self.arc_a_name = f"ArcLow_{get_random_name()}"
        self.arc_b_name = f"ArcHigh_{get_random_name()}"
        self._arc_a_created = False
        self._arc_a_started = False
        self._arc_b_created = False
        self._arc_b_started = False
        self.set_manual_timeout(60)

        # Create virtual command channels
        self.client.channels.create(
            name="start_low_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="start_high_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        self.subscribe(["press_vlv_state", "press_pt", "end_test_cmd"])
        super().setup()

    def run(self) -> None:
        rack_key = self.params.get("rack_key")
        if rack_key:
            rack = self.client.racks.retrieve(rack_key)
        else:
            rack = self.client.racks.retrieve(embedded=False)
        assert rack is not None

        # Create and start Arc A (low priority, writes 0)
        self.log(f"Creating Arc A (low priority): {self.arc_a_name}")
        self.console.arc.create(self.arc_a_name, ARC_LOW_PRIORITY_SOURCE, mode="Text")
        self._arc_a_created = True
        self.console.arc.select_rack(rack.name)
        self.console.arc.configure()
        self.console.arc.start()
        self._arc_a_started = True

        self.log("Triggering Arc A sequence")
        with self.client.open_writer(sy.TimeStamp.now(), "start_low_cmd") as w:
            w.write("start_low_cmd", 1)

        # Wait for Arc A to control (valve closed = 0)
        self.log("Waiting for Arc A to control (press_vlv_state == 0)...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 0:
                self.log("Arc A in control - valve closed")
                break
        else:
            self.fail("Arc A should close valve at authority 100")
            return

        # Create and start Arc B (high priority, writes 1)
        self.log(f"Creating Arc B (high priority): {self.arc_b_name}")
        self.console.arc.create(self.arc_b_name, ARC_HIGH_PRIORITY_SOURCE, mode="Text")
        self._arc_b_created = True
        self.log("Selecting rack for Arc B")
        self.console.arc.select_rack(rack.name)
        self.log("Configuring Arc B")
        self.console.arc.configure()
        self.log("Starting Arc B")
        self.console.arc.start()
        self._arc_b_started = True
        self.log(f"Arc B is running: {self.console.arc.is_running()}")

        self.log("Triggering Arc B sequence")
        with self.client.open_writer(sy.TimeStamp.now(), "start_high_cmd") as w:
            w.write("start_high_cmd", 1)

        # Wait for Arc B to override (valve open = 1)
        self.log("Waiting for Arc B to override (press_vlv_state == 1)...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 1:
                self.log("Arc B overrides - valve opened at authority 200")
                break
        else:
            self.fail("Arc B should open valve at authority 200")
            return

        # Wait for Arc B to reach done stage (~5s), Arc A resumes
        self.log("Waiting for Arc B done stage, Arc A should resume...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 0:
                self.log("Arc A resumed control - valve closed again")
                break
        else:
            self.fail("Arc A should resume after Arc B stops writing")
            return

    def teardown(self) -> None:
        if self._arc_b_started and self.console.arc.is_running():
            try:
                self.console.arc.stop()
            except Exception as e:
                self.fail(f"Failed to stop Arc B: {e}")

        if self._arc_b_created:
            try:
                self.console.arc.delete(self.arc_b_name)
            except Exception as e:
                self.fail(f"Failed to delete Arc B: {e}")

        if self._arc_a_started:
            try:
                self.console.arc.open(self.arc_a_name)
                if self.console.arc.is_running():
                    self.console.arc.stop()
            except Exception as e:
                self.fail(f"Failed to stop Arc A: {e}")

        if self._arc_a_created:
            try:
                self.console.arc.delete(self.arc_a_name)
            except Exception as e:
                self.fail(f"Failed to delete Arc A: {e}")

        try:
            with self.client.open_writer(sy.TimeStamp.now(), "end_test_cmd") as w:
                w.write("end_test_cmd", 1)
        except Exception as e:
            self.fail(f"Failed to signal simulator stop: {e}")

        super().teardown()
