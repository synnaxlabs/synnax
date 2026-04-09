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
from tests.arc.arc_case import ArcConsoleCase
from x import random_name

ARC_LIFECYCLE_SOURCE = """

PRESS_HIGH_LIMIT f32 := 25
PRESS_LOW_LIMIT f32 := 5

SOME_CONST_1 f32 := 42.0
SOME_CONST_2 f32 := -49.5
start_lifecycle_cmd => main

func check_high_pressure(p f32) u8 {
    return p > PRESS_HIGH_LIMIT
}

func event_log{msg str} () {
    lifecycle_log = msg
}

press_pt -> check_high_pressure{} -> stable_for{500ms} -> select{} -> {
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

// Functions are deliberately scrambled (1, 3, 2) to test that channel
// accumulation works with both forward and backward references.
func nested_write_1() {
    nested_write_2(press_pt)
}

func nested_write_3(val f32) {
    arc_lifecycle_virt = val
}

func nested_write_2(val f32) {
    nested_write_3(val)
}

interval{100ms} -> nested_write_1{}

sequence main {
    stage press {
        SOME_CONST_1 => const_output,
        1 -> press_vlv_cmd,
        event_log{"pressurizing"},
        press_pt > PRESS_HIGH_LIMIT + 5 => maintain
    }

    stage maintain {
        0 -> press_vlv_cmd,
        wait{1s} => vent
    }

    stage vent {
        SOME_CONST_2 * 2 => const_output,
        1 -> vent_vlv_cmd,
        event_log{"venting"},
        press_pt < PRESS_LOW_LIMIT => complete
    }

    stage complete {
        0 -> vent_vlv_cmd
    }
}

// Regression: stale virtual channel signal must not trigger stage re-entry.
// When yield is first entered, the pre-existing bb_signal_start_cmd value
// should be ignored; only a new write after activation should fire => start.
bb_signal_start_cmd => signal_ctrl

sequence signal_ctrl {
    stage start {
        "start" -> signal_stage_log,
        bb_signal_stop_cmd => stop
    }
    stage stop {
        "stop" -> signal_stage_log,
        wait{250ms} => yield
    }
    stage yield {
        "yield" -> signal_stage_log,
        bb_signal_start_cmd => start
    }
}
"""


class Lifecycle(ArcConsoleCase):
    """Test Arc lifecycle operations: rename, delete, status, stable_for, select.

    Covers the following rc.md checklist items:
    1. Rename arc automation via context menu (with redeployment warning)
    2. Delete arc automation via context menu (layouts removed from mosaic)
    3. Rename an arc, re-deploy it, ensure new name is displayed
    4. Create a status with the name of the automation when it starts running
    5. stable_for filters values until stable for a specified duration
    6. select routes boolean output to true/false branches
    7. Channel writes propagate through function calls (regression for channel
       accumulation bug where transitive channel accesses were lost)
    8. Stale virtual channel signal does not trigger stage re-entry on first
       yield activation (regression for source node watermark bug)
    """

    arc_source = ARC_LIFECYCLE_SOURCE
    arc_name_prefix = "ArcLifecycle"
    start_cmd_channel = "start_lifecycle_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = [
        "press_vlv_state",
        "press_pt",
        "arc_lifecycle_virt",
        "const_output",
        "end_test_cmd",
        "lifecycle_log",
        "signal_stage_log",
    ]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self.new_name = f"ArcRenamed_{random_name()}"
        self.client.channels.create(
            name="arc_lifecycle_virt",
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
            virtual=True,
        )
        self.client.channels.create(
            name="const_output",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="lifecycle_log",
            data_type=sy.DataType.STRING,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="signal_stage_log",
            data_type=sy.DataType.STRING,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="bb_signal_start_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="bb_signal_stop_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )
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
        # --- 0a. Verify global constant as flow source in press stage ---
        self.log("Verifying SOME_CONST_1 (42.0) => const_output during press stage")
        self.wait_for_near("const_output", 42.0, tolerance=0.01, is_virtual=True)

        # --- 0b. Verify transitive channel write through function calls ---
        # nested_write_1() calls nested_write_2() which calls nested_write_3()
        # which writes to arc_lifecycle_virt. This validates that channel
        # accumulation propagates through function calls.
        self.log("Verifying transitive channel write (function call propagation)")
        self.wait_for_gt("arc_lifecycle_virt", 20, is_virtual=True)

        # --- 1. Verify select true branch: stable_for emits after pressure
        # stays above 25 PSI for 500ms, then select routes to warning status ---
        self.log("Waiting for 'Pressure stable above 25 PSI' (select true branch)")
        self.wait_for_eq("lifecycle_log", "pressurizing", is_virtual=True)
        if not self.console.notifications.wait_for("Pressure stable above 25 PSI"):
            self.fail("Notification 'Pressure stable above 25 PSI' not found")

        # --- 2. Verify select false branch: initial pressure=0 causes
        # check_high_pressure to return 0, stable for 500ms, then select routes
        # to false.
        self.log("Checking for 'Pressure below 25 PSI' (select false branch)")
        if not self.console.notifications.wait_for("Pressure below 25 PSI"):
            self.fail("Notification 'Pressure below 25 PSI' not found")
        self.wait_for_eq("lifecycle_log", "venting", is_virtual=True)

        # --- 2a. Verify global constant changed in vent stage ---
        self.log("Verifying SOME_CONST_2 *2 (-99.0) => const_output during vent stage")
        self.wait_for_near("const_output", -99.0, tolerance=0.01, is_virtual=True)

        self.console.notifications.close_all()

        # --- 3. Regression: stale virtual channel must not trigger re-entry ---
        # Trigger signal_ctrl via bb_signal_start_cmd, then stop it. The yield
        # stage's source node must advance its watermark on activation so the
        # pre-existing bb_signal_start_cmd value is not seen as new data.
        self.log("Phase 3: Testing stale virtual channel regression (signal_ctrl)")
        self.writer.write("bb_signal_start_cmd", 1)

        self.wait_for_eq("signal_stage_log", "start", is_virtual=True)
        self.log("signal_ctrl entered start stage")

        self.writer.write("bb_signal_stop_cmd", 1)

        self.wait_for_eq("signal_stage_log", "yield", is_virtual=True)
        self.log("signal_ctrl entered yield stage")

        # Wait then confirm no spurious re-entry from the stale start signal.
        sy.sleep(0.501)  #  > 2 * wait{250ms}
        self.wait_for_eq("signal_stage_log", "yield", is_virtual=True)

        # Confirm a fresh start signal correctly re-enters start.
        self.writer.write("bb_signal_start_cmd", 1)
        self.wait_for_eq("signal_stage_log", "start", is_virtual=True)

        # --- 4. Rename while running (triggers redeployment warning) ---
        self.log(f"Renaming Arc from '{self.arc_name}' to '{self.new_name}'")
        self.console.arc.rename(old_name=self.arc_name, new_name=self.new_name)
        self._arc_started = False  # Rename stops the arc

        self.log("Verifying new name in toolbar")
        self.console.arc.wait_for_item(self.new_name)

        old_item = self.console.arc.find_item(self.arc_name)
        assert old_item is None, f"Old name '{self.arc_name}' still present"

        # Update arc_name so parent teardown uses the new name
        self.arc_name = self.new_name

        # --- 5. Re-configure and re-start with new name ---
        self.log("Opening renamed Arc")
        self.console.arc.open(self.new_name)

        self.log("Re-configuring with new name")
        assert self.rack is not None
        self.console.arc.select_rack(self.rack.name)
        self.console.arc.configure()

        self.log("Re-starting with new name")
        self.console.arc.start()
        self._arc_started = True

        # --- 6. Stop, then delete and verify tab removal ---
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
