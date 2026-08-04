#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.simulators import PressSimDAQ

from console.case import ConsoleCase
from console.schematic import Valve
from console.schematic.schematic import Schematic
from tests.arc.arc import ArcCase

ARC_CONTINUOUS_PRESS_SOURCE = """
authority 200

start_seq_cmd => main

sequence main {
    stage pressurize {
        1 -> press_vlv_cmd
        wait{100ms} => pressurize
    }
}
"""


class AuthoritySchematicOverride(ArcCase, ConsoleCase):
    """Test that a schematic at authority 255 overrides an Arc program at
    authority 200, and Arc resumes when schematic releases control."""

    arc_source = ARC_CONTINUOUS_PRESS_SOURCE
    arc_name_prefix = "ArcSchematic"
    start_cmd_channel = "start_seq_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ["press_vlv_state", "press_pt", "end_test_cmd"]
    sim_daq_class = PressSimDAQ
    collect_task_status = True

    def setup(self) -> None:
        self._schematic: Schematic | None = None
        self._schematic_controlled = False
        super().setup()
        self.set_manual_timeout(120)

    def verify_sequence_execution(self) -> None:
        # Phase 1: Arc in control - valve should open
        self.wait_for_eq("press_vlv_state", 1)

        self.log("Creating schematic with valve")
        schematic = self.console.project.create_schematic("authority_test_schematic")
        self._schematic = schematic
        schematic.move("left")

        valve = schematic.create_symbol(
            Valve(
                label="press_vlv",
                state_channel="press_vlv_state",
                command_channel="press_vlv_cmd",
            )
        )

        # Phase 2: Schematic override - set authority 255 and acquire control
        self.log("Deselecting valve, setting schematic authority to 255")
        self.console.layout.press_escape()
        schematic.set_authority(255)
        self.log("Acquiring schematic control")
        schematic.acquire_control()
        self._schematic_controlled = True

        self.log("Pressing valve to close it")
        valve.press()

        self.wait_for_eq("press_vlv_state", 0)

        task = self.task_key(self.arc_name)
        assert self.wait_for_task_status(
            task, "Authority held on 1 channel by another writer"
        ), "Arc did not warn when out-ranked by the authority-255 schematic"

        # Phase 3: Release control - Arc should resume
        self.log("Releasing schematic control")
        schematic.release_control()
        self._schematic_controlled = False

        self.wait_for_eq("press_vlv_state", 1)

        assert self.wait_for_task_status_clear(task, "Authority held on"), (
            "Arc control warning did not clear after the schematic released control"
        )

    def teardown(self) -> None:
        if self._schematic_controlled and self._schematic is not None:
            try:
                self._schematic.release_control()
            except Exception as e:
                self.fail(f"Failed to release schematic control: {e}")
        super().teardown()
