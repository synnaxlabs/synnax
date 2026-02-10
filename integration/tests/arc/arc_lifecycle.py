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

ARC_LIFECYCLE_SOURCE = """
start_lifecycle_cmd => main

func check_high_pressure(p f32) u8 {
    return p > 25
}

press_pt -> check_high_pressure{} -> stable_for{duration=500ms} -> select{} -> {
    true: set_status{
        status_key="lifecycle_press_alarm",
        name="Lifecycle Press Alarm",
        variant="warning",
        message="Pressure stable above 25 PSI"
    },
    false: set_status{
        status_key="lifecycle_press_normal",
        name="Lifecycle Press Normal",
        variant="warning",
        message="Pressure below 25 PSI"
    }
}

sequence main {
    stage press {
        1 -> press_vlv_cmd,
        press_pt > 30 => maintain
    }

    stage maintain {
        0 -> press_vlv_cmd,
        wait{duration=1s} => vent
    }

    stage vent {
        1 -> vent_vlv_cmd,
        press_pt < 5 => complete
    }

    stage complete {
        0 -> vent_vlv_cmd
    }
}
"""


class ArcLifecycle(ArcConsoleCase):
    """Test Arc lifecycle operations: rename, delete, status, stable_for, select.

    Covers the following rc.md checklist items:
    1. Rename arc automation via context menu (with redeployment warning)
    2. Delete arc automation via context menu (layouts removed from mosaic)
    3. Rename an arc, re-deploy it, ensure new name is displayed
    4. Create a status with the name of the automation when it starts running
    5. stable_for filters values until stable for a specified duration
    6. select routes boolean output to true/false branches
    """

    arc_source = ARC_LIFECYCLE_SOURCE
    arc_name_prefix = "ArcLifecycle"
    start_cmd_channel = "start_lifecycle_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ["press_vlv_state", "press_pt", "end_test_cmd"]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self.new_name = f"ArcRenamed_{get_random_name()}"
        self.client.statuses.set(
            sy.Status(
                key="lifecycle_press_alarm",
                name="Lifecycle Press Alarm",
                variant="disabled",
                message="Initialized",
            )
        )
        super().setup()

    def verify_sequence_execution(self) -> None:
        # --- 1. Verify select true branch: stable_for emits after pressure
        # stays above 25 PSI for 500ms, then select routes to warning status ---
        self.log("Waiting for 'Pressure stable above 25 PSI' (select true branch)")
        if not self.console.notifications.wait_for("Pressure stable above 25 PSI"):
            self.fail("Notification 'Pressure stable above 25 PSI' not found")

        # --- 2. Verify select false branch: initial pressure=0 causes
        # check_high_pressure to return 0, stable for 500ms, then select routes
        # to false.
        self.log("Checking for 'Pressure below 25 PSI' (select false branch)")
        if not self.console.notifications.wait_for("Pressure below 25 PSI"):
            self.fail("Notification 'Pressure below 25 PSI' not found")
        self.console.notifications.close_all()

        # --- 3. Rename while running (triggers redeployment warning) ---
        self.log(f"Renaming Arc from '{self.arc_name}' to '{self.new_name}'")
        self.console.arc.rename(old_name=self.arc_name, new_name=self.new_name)
        self._arc_started = False  # Rename stops the arc

        self.log("Verifying new name in toolbar")
        new_item = self.console.arc.find_item(self.new_name)
        assert new_item is not None, f"Renamed Arc '{self.new_name}' not found"

        old_item = self.console.arc.find_item(self.arc_name)
        assert old_item is None, f"Old name '{self.arc_name}' still present"

        # Update arc_name so parent teardown uses the new name
        self.arc_name = self.new_name

        # --- 4. Re-configure and re-start with new name ---
        self.log("Opening renamed Arc")
        self.console.arc.open(self.new_name)

        self.log("Re-configuring with new name")
        assert self.rack is not None
        self.console.arc.select_rack(self.rack.name)
        self.console.arc.configure()

        self.log("Re-starting with new name")
        self.console.arc.start()
        self._arc_started = True

        # --- 5. Stop, then delete and verify tab removal ---
        self.log("Stopping Arc")
        self.console.arc.stop()
        self._arc_started = False

        self.log("Verifying tab exists before delete")
        tab = self.console.layout.get_tab(self.new_name)
        tab.wait_for(state="visible", timeout=5000)
        self.log("Tab found in mosaic")

        self.log(f"Deleting Arc: {self.new_name}")
        self.console.arc.delete(self.new_name)
        self._arc_created = False

        self.log("Verifying tab removed from mosaic")
        tab = self.console.layout.get_tab(self.new_name)
        tab.wait_for(state="hidden", timeout=5000)
