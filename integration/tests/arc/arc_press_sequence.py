#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import synnax as sy

from console.case import ConsoleCase

ARC_NAME = f"ArcPressSequence_{int(time.time())}"

ARC_SEQUENCE_SOURCE = """
start_seq_cmd => main

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


class ArcPressSequence(ConsoleCase):
    """Test Arc pressurization sequence execution via Console UI."""

    def setup(self) -> None:
        self.set_manual_timeout(180)
        self.subscribe(
            [
                "press_vlv_state",
                "vent_vlv_state",
                "press_pt",
                "end_test_cmd",
            ]
        )
        super().setup()
        self._create_extra_channels()

    def _create_extra_channels(self) -> None:
        self.client.channels.create(
            name="start_seq_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        self.log("Creating Arc sequence through Console UI")
        self.console.arc.create(ARC_NAME, ARC_SEQUENCE_SOURCE, mode="Text")
        sy.sleep(0.5)

        rack = self.client.racks.retrieve(key=65538)
        rack_name = rack.name

        self.log(f"Selecting rack: {rack_name} (key: {rack.key})")
        self.console.arc.select_rack(rack_name)

        self.log("Configuring Arc task")
        self.console.arc.configure()
        sy.sleep(1.0)

        arc = self.client.arcs.retrieve(name=ARC_NAME)
        self.log(f"Arc saved with key: {arc.key}")

        self.log("Starting Arc task")
        self.console.arc.start()
        self.log(f"Arc is running: {self.console.arc.is_running()}")
        sy.sleep(1.0)

        self.log("Triggering sequence")
        with self.client.open_writer(sy.TimeStamp.now(), "start_seq_cmd") as w:
            w.write("start_seq_cmd", 1)

        self._verify_sequence_execution()

        self.log("Stopping Arc task")
        self.console.arc.stop()
        sy.sleep(0.5)

        self.log("Deleting Arc program")
        self.console.arc.delete(ARC_NAME)

        self.log("Signaling sim_daq to stop")
        with self.client.open_writer(sy.TimeStamp.now(), "end_test_cmd") as w:
            w.write("end_test_cmd", 1)

        self.log(f"Arc sequence on {rack_name} completed")

    def _verify_sequence_execution(self) -> None:
        self.log("Verifying press stage - valve opens...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 1:
                self.log("Press valve opened")
                break
            if self.should_stop:
                self.fail("Press valve should open")
                return

        self.log("Verifying maintain stage - press valve closes...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 0:
                self.log("Maintain stage reached - press valve closed")
                break
            if self.should_stop:
                self.fail("Press valve should close in maintain stage")
                return

        self.log("Verifying vent stage - vent valve opens...")
        while self.should_continue:
            if self.read_tlm("vent_vlv_state") == 1:
                self.log("Vent stage reached - vent valve opened")
                break
            if self.should_stop:
                self.fail("Vent valve should open")
                return

        self.log("Verifying complete stage - vent valve closes...")
        while self.should_continue:
            if self.read_tlm("vent_vlv_state") == 0:
                self.log("Complete stage reached - sequence finished!")
                break
            if self.should_stop:
                self.fail("Vent valve should close in complete stage")
                return
