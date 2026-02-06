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
from console.schematic import Valve
from console.schematic.schematic import Schematic
from framework.sim_daq_case import SimDaqTestCase
from framework.utils import get_random_name
from tests.arc.arc_case import ArcConsoleCase

ARC_CONTINUOUS_PRESS_SOURCE = """
authority 200

start_seq_cmd => main

sequence main {
    stage pressurize {
        1 -> press_vlv_cmd,
        wait{duration=100ms} => pressurize
    }
}
"""


class ArcAuthoritySchematicOverride(SimDaqTestCase, ConsoleCase):
    """Test that a schematic at authority 255 overrides an Arc program at
    authority 200, and Arc resumes when schematic releases control."""

    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self.arc_name = f"ArcSchematic_{get_random_name()}"
        self._arc_created = False
        self._arc_started = False
        self._schematic_controlled = False
        self.set_manual_timeout(120)

        self.client.channels.create(
            name="start_seq_cmd",
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

        # Create and start Arc program (authority 200, writes 1 to press_vlv_cmd)
        self.log(f"Creating Arc program: {self.arc_name}")
        self.console.arc.create(
            self.arc_name, ARC_CONTINUOUS_PRESS_SOURCE, mode="Text"
        )
        self._arc_created = True
        self.console.arc.select_rack(rack.name)
        self.console.arc.configure()
        self.console.arc.start()
        self._arc_started = True

        self.log("Triggering Arc sequence")
        with self.client.open_writer(sy.TimeStamp.now(), "start_seq_cmd") as w:
            w.write("start_seq_cmd", 1)

        # Phase 1: Arc in control - valve should open
        self.log("Waiting for Arc to control (press_vlv_state == 1)...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 1:
                self.log("Arc in control - valve opened")
                break
        else:
            self.fail("Arc should open valve at authority 200")
            return

        # Create schematic with valve
        self.log("Creating schematic with valve")
        schematic = Schematic(self.console, "authority_test_schematic")
        schematic.move("left")

        valve = schematic.create_symbol(
            Valve(
                label="press_vlv",
                state_channel="press_vlv",
                command_channel="press_vlv",
            )
        )

        # Phase 2: Schematic override - set authority 255 and acquire control
        self.log("Deselecting valve, setting schematic authority to 255")
        self.page.keyboard.press("Escape")
        schematic.set_properties(control_authority=255)
        self.log("Acquiring schematic control")
        schematic.acquire_control()
        self._schematic_controlled = True

        self.log("Pressing valve to close it")
        valve.press()

        self.log("Waiting for schematic override (press_vlv_state == 0)...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 0:
                self.log("Schematic overrides Arc - valve closed")
                break
        else:
            self.fail("Schematic at 255 should override Arc at 200")
            return

        # Phase 3: Release control - Arc should resume
        self.log("Releasing schematic control")
        schematic.release_control()
        self._schematic_controlled = False

        self.log("Waiting for Arc to resume (press_vlv_state == 1)...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 1:
                self.log("Arc resumed control - valve opened again")
                break
        else:
            self.fail("Arc should resume after schematic releases control")
            return

    def teardown(self) -> None:
        if self._schematic_controlled:
            try:
                # Find the schematic pane and release control
                self.page.locator(".react-flow__pane").first.click()
                control_button = (
                    self.page.locator(
                        ".console-controls button.pluto-btn--filled"
                    )
                    .filter(
                        has=self.page.locator("svg.pluto-icon--circle")
                    )
                    .first
                )
                if control_button.count() > 0:
                    control_button.click()
            except Exception as e:
                self.log(f"Failed to release schematic control: {e}")

        if self._arc_started:
            try:
                self.console.arc.open(self.arc_name)
                if self.console.arc.is_running():
                    self.console.arc.stop()
            except Exception as e:
                self.log(f"Failed to stop Arc: {e}")

        if self._arc_created:
            try:
                self.console.arc.delete(self.arc_name)
            except Exception as e:
                self.log(f"Failed to delete Arc: {e}")

        try:
            with self.client.open_writer(sy.TimeStamp.now(), "end_test_cmd") as w:
                w.write("end_test_cmd", 1)
        except Exception as e:
            self.log(f"Failed to signal simulator stop: {e}")

        super().teardown()
