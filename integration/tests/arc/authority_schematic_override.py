#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.simulators import PressSimDAQ

import synnax as sy
from console.case import ConsoleCase
from console.schematic import Valve
from console.schematic.schematic import Schematic
from framework.utils import create_virtual_channel
from tests.arc.arc import ArcCase

ARC_PER_CHANNEL_PRESS_SOURCE = """
import control

authority 0

start_seq_cmd => main

sequence main {
    stage pressurize {
        control.set_authority{value=210, channel=press_vlv_cmd}
        1 -> press_vlv_cmd
        seq_close_cmd => depressurize
        wait{100ms} => pressurize
    }
    stage depressurize {
        control.set_authority{value=210, channel=press_vlv_cmd}
        0 -> press_vlv_cmd
        wait{100ms} => depressurize
    }
}
"""


class AuthoritySchematicOverride(ArcCase, ConsoleCase):
    """Test schematic-versus-Arc control in both directions. A schematic at
    authority 255 overrides the Arc's per-channel claim at 210, and the Arc
    resumes when the schematic releases. A schematic holding control at the
    default authority 1 must NOT block the Arc's per-channel claim, since the
    claim must cover the command channel's index, which the schematic holds."""

    arc_source = ARC_PER_CHANNEL_PRESS_SOURCE
    arc_name_prefix = "ArcSchematic"
    start_cmd_channel = "start_seq_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ["press_vlv_state", "press_pt", "end_test_cmd"]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self._schematic: Schematic | None = None
        self._schematic_controlled = False
        create_virtual_channel(self.client, "seq_close_cmd", sy.DataType.UINT8)
        super().setup()
        self.set_manual_timeout(120)

    def verify_sequence_execution(self) -> None:
        # Phase 1: Arc in control - valve should open
        self.wait_for_eq("press_vlv_state", 1)

        self.log("Creating schematic with valve")
        schematic = self.console.pages.create(Schematic, "authority_test_schematic")
        self._schematic = schematic
        schematic.move("left")

        valve = schematic.create_symbol(
            Valve(
                label="press_vlv",
                state_channel="press_vlv_state",
                command_channel="press_vlv_cmd",
            )
        )
        self.console.layout.press_escape()

        # Phase 2: Acquire at the untouched default authority 1. The Arc's
        # per-channel claim at 210 must keep commanding the valve while the
        # schematic holds control, including the command channel's index, which
        # the schematic holds too.
        self.log("Acquiring schematic control at the default authority")
        schematic.acquire_control()
        self._schematic_controlled = True

        self.log("Commanding sequence to close valve while schematic holds")
        self.writer.write("seq_close_cmd", 1)
        self.wait_for_eq("press_vlv_state", 0, timeout=10)
        self.log("Arc commanded the valve without a schematic release")

        # Phase 3: Releasing at authority 1 must not disturb the sequence.
        self.log("Releasing schematic control")
        schematic.release_control()
        self._schematic_controlled = False
        self.wait_for_eq("press_vlv_state", 0, timeout=5)

        # Phase 4: Schematic override - set authority 255 and acquire control.
        # The Control Authority input only attaches in edit mode.
        self.log("Setting schematic authority to 255")
        schematic.enable_edit()
        self.console.layout.show_visualization_toolbar()
        schematic.set_authority(255)
        self.log("Acquiring schematic control")
        schematic.acquire_control()
        self._schematic_controlled = True

        self.log("Pressing valve to open it")
        valve.press()

        self.wait_for_eq("press_vlv_state", 1)

        # Phase 5: Release control - Arc should resume closing the valve
        self.log("Releasing schematic control")
        schematic.release_control()
        self._schematic_controlled = False

        self.wait_for_eq("press_vlv_state", 0)

    def teardown(self) -> None:
        if self._schematic_controlled and self._schematic is not None:
            try:
                self._schematic.release_control()
            except Exception as e:
                self.fail(f"Failed to release schematic control: {e}")
        super().teardown()
