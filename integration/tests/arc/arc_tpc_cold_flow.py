#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from examples.simulators import TPCSimDAQ

from tests.arc.arc_case import ArcConsoleCase

ARC_SEQUENCE_SOURCE = """
// Multi-output function that classifies pressure into three categories
func classify_pressure{safe_threshold f32, warning_threshold f32} (pressure f32) (safe_out f32, warning_out f32, critical_out f32) {
    if (pressure < safe_threshold) {
        safe_out = pressure
    } else if (pressure < warning_threshold) {
        warning_out = pressure
    } else {
        critical_out = pressure
    }
}

// Helper function triggered via dataflow - closes all pressurization valves
func close_press{} (trigger f32) {
    gas_booster_iso_cmd = 0
    press_iso_cmd = 0
    ox_press_cmd = 0
    fuel_press_cmd = 0
}

// Vent control - state > 0 closes vents (1=closed, 0=open)
func vent_control{} (state f32) {
    if (state > 0) {
        ox_vent_cmd = 1
        fuel_vent_cmd = 1
    } else {
        ox_vent_cmd = 0
        fuel_vent_cmd = 0
    }
}

// MPV control - state > 0 opens MPVs (1=open, 0=closed)
func mpv_control{} (state f32) {
    if (state > 0) {
        ox_mpv_cmd = 1
        fuel_mpv_cmd = 1
    } else {
        ox_mpv_cmd = 0
        fuel_mpv_cmd = 0
    }
}

// Dataflows: trigger channels activate helper functions
tpc_press_trigger -> close_press{}
tpc_vent_trigger -> vent_control{}
tpc_mpv_trigger -> mpv_control{}

// Status handlers - write strings to pressure_status channel via dataflow
func on_safe{} (value f32) {
    pressure_status = "Safe"
}

func on_warning{} (value f32) {
    pressure_status = "Warning"
}

func on_critical{} (value f32) {
    pressure_status = "Critical"
}

// Dataflow: OX pressure -> classifier -> status handlers
ox_pt_1 -> classify_pressure{safe_threshold=15, warning_threshold=40} -> {
    safe_out: on_safe{},
    warning_out: on_warning{},
    critical_out: on_critical{}
}

start_tpc_cmd => main

sequence main {
    stage precheck {
        1 -> tpc_press_trigger,
        1 -> tpc_vent_trigger,
        0 -> tpc_mpv_trigger,
        1 -> tpc_stage,
        wait{duration=500ms} => press_charge
    }

    stage press_charge {
        1 -> gas_booster_iso_cmd,
        0 -> press_iso_cmd,
        0 -> ox_press_cmd,
        0 -> fuel_press_cmd,
        1 -> tpc_vent_trigger,
        0 -> tpc_mpv_trigger,
        2 -> tpc_stage,
        press_pt_1 > 200 => ox_press
    }

    stage ox_press {
        0 -> gas_booster_iso_cmd,
        1 -> press_iso_cmd,
        1 -> ox_press_cmd,
        0 -> fuel_press_cmd,
        1 -> tpc_vent_trigger,
        0 -> tpc_mpv_trigger,
        3 -> tpc_stage,
        ox_pt_1 > 50 => fuel_press
    }

    stage fuel_press {
        0 -> gas_booster_iso_cmd,
        1 -> press_iso_cmd,
        0 -> ox_press_cmd,
        1 -> fuel_press_cmd,
        1 -> tpc_vent_trigger,
        0 -> tpc_mpv_trigger,
        4 -> tpc_stage,
        fuel_pt_1 > 50 => hold
    }

    stage hold {
        1 -> tpc_press_trigger,
        1 -> tpc_vent_trigger,
        0 -> tpc_mpv_trigger,
        5 -> tpc_stage,
        wait{duration=2s} => fire
    }

    stage fire {
        1 -> tpc_press_trigger,
        1 -> tpc_vent_trigger,
        1 -> tpc_mpv_trigger,
        6 -> tpc_stage,
        wait{duration=1s} => shutdown
    }

    stage shutdown {
        1 -> tpc_press_trigger,
        0 -> tpc_vent_trigger,
        0 -> tpc_mpv_trigger,
        7 -> tpc_stage,
        ox_pt_1 < 5 and fuel_pt_1 < 5 => safe
    }

    stage safe {
        1 -> tpc_press_trigger,
        0 -> tpc_vent_trigger,
        0 -> tpc_mpv_trigger,
        8 -> tpc_stage,
        start_tpc_cmd == 0 => idle
    }

    stage idle {
        1 -> tpc_press_trigger,
        0 -> tpc_vent_trigger,
        0 -> tpc_mpv_trigger,
        0 -> tpc_stage,
        start_tpc_cmd == 1 => precheck
    }
}
"""

PHASE_NAMES = {
    0: "Idle",  # Waiting for start command
    1: "Precheck",  # Close vents, verify sensors
    2: "Press Charge",  # Charge press tank via gas booster
    3: "OX Pressurization",  # Pressurize OX tank
    4: "FUEL Pressurization",  # Pressurize FUEL tank
    5: "Hold",  # Maintain pressure
    6: "Fire",  # Brief simulated firing (open MPVs)
    7: "Shutdown",  # Close MPVs, open vents
    8: "Safe",  # System safe, sequence complete
}


class ArcTPCColdFlow(ArcConsoleCase):
    """Test Arc TPC cold flow sequence with dataflow-based valve control.

    This test simulates a multi-phase rocket engine cold flow test using
    TPCSimDAQ, demonstrating Arc's dataflow-triggered functions and
    multi-output routing.

    Sequence Stages:
    - idle: Waits for start_tpc_cmd == 1 to begin
    - precheck: Closes vents, verifies initial state
    - press_charge: Charges press tank via gas booster (until press_pt_1 > 200)
    - ox_press: Pressurizes OX tank (until ox_pt_1 > 50)
    - fuel_press: Pressurizes FUEL tank (until fuel_pt_1 > 50)
    - hold: Maintains pressure for 2 seconds
    - fire: Opens MPVs for 1 second (simulated firing)
    - shutdown: Vents tanks (until ox_pt_1 < 5 and fuel_pt_1 < 5)
    - safe: Sequence complete, waits for start_tpc_cmd == 0 to return to idle

    Arc Features Tested:
    1. Multi-output functions with routing tables (classify_pressure -> 3 outputs)
    2. Dataflow-triggered helper functions (trigger channels -> valve control)
    3. Conditional valve control via state parameter (vent_control, mpv_control)
    4. String channel writes from status handler functions
    5. Reusable sequence with idle/start loop

    Architecture:
    - tpc_*_trigger -> close_press{}, vent_control{}, mpv_control{}
    - ox_pt_1 -> classify_pressure{} -> on_safe{}, on_warning{}, on_critical{}
    """

    arc_source = ARC_SEQUENCE_SOURCE
    arc_name_prefix = "ArcTPCColdFlow"
    start_cmd_channel = "start_tpc_cmd"
    end_cmd_channel = "end_tpc_test_cmd"
    subscribe_channels = [
        "tpc_stage",
        "ox_pt_1",
        "fuel_pt_1",
        "press_pt_1",
        "pressure_status",
        "end_tpc_test_cmd",
    ]
    sim_daq_class = TPCSimDAQ

    def setup(self) -> None:
        self.client.channels.create(
            name="tpc_stage",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="pressure_status",
            data_type=sy.DataType.STRING,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        # Trigger channels for dataflow-activated helper functions
        # Value passed to trigger becomes the command value
        for trigger_name in ["press_trigger", "vent_trigger", "mpv_trigger"]:
            self.client.channels.create(
                name=f"tpc_{trigger_name}",
                data_type=sy.DataType.FLOAT32,
                virtual=True,
                retrieve_if_name_exists=True,
            )
        super().setup()

    def verify_sequence_execution(self) -> None:
        current_phase = 0
        current_status = ""
        observed_statuses: set[str] = set()

        self.log("Monitoring phase transitions and pressure status...")
        while self.should_continue:
            new_phase = self.read_tlm("tpc_stage")
            new_status = self.read_tlm("pressure_status")

            if new_phase is not None:
                new_phase = int(new_phase)
                if new_phase != current_phase and new_phase > 0:
                    current_phase = new_phase
                    self.log(
                        f"Phase {new_phase}: {PHASE_NAMES.get(new_phase, 'Unknown')}"
                    )

                    if new_phase == 8:
                        self.log("Sequence complete - reached Safe phase")
                        break

            if new_status is not None and str(new_status) != current_status:
                current_status = str(new_status)
                observed_statuses.add(current_status)
                ox_pt = self.read_tlm("ox_pt_1")
                self.log(f"Pressure status: {current_status} (ox_pt_1={ox_pt:.1f})")

            if self.should_stop:
                self.fail(f"Test stopped unexpectedly in phase {current_phase}")
                return

        self.log(f"Observed status values: {observed_statuses}")

        expected_statuses = {"Safe", "Warning", "Critical"}
        missing = expected_statuses - observed_statuses
        if missing:
            self.fail(f"Missing expected status values: {missing}")
            return

        self.log("Multi-output function verified - all status levels observed")
        self.log("TPC cold flow sequence executed successfully")
