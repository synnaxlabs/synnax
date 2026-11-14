#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A 32xDO of the HITL station in DewesoftX mode, and its switch to Synnax. Each test
writes the mode in RAM, and never the save object."""

from datetime import UTC

from pydantic import JsonValue

import synnax as sy
from framework.test_case import TestCase
from tests.driver.task import assert_start_rejected, delete_channels
from tests.hitl.ethercat import (
    DEWESOFTX_MODE,
    SLAVES,
    START_TIMEOUT,
    STGS,
    SWITCHED,
    SYNNAX_MODE,
    create_write_task,
    prepare_bus,
    scanned_mode,
    write_mode,
)

NAME = SLAVES[SWITCHED - 1]
LEFT_OUT = (
    f"Failed to start task: Slave {SWITCHED} ({NAME}) is left out of the network: it "
    "refused SAFE-OP: Invalid DC SYNC configuration. Fix the device, then stop every "
    "task on the network to try it again."
)
SYNC_ADVICE = (
    'To fix "Invalid DC SYNC configuration", right-click the device, select '
    "Configure, and pick another sync mode."
)
SWITCH_ADVICE = (
    "The device is set up for DewesoftX. In Configure, switch the device to Synnax."
)
SCAN_TYPE = "ethercat_scan"
_STATUS_TIMEOUT = 30 * sy.TimeSpan.SECOND


class EtherCATSwitch(TestCase):
    """Checks the start error of a 32xDO in DewesoftX mode, with and without a picked
    sync mode, then its switch to Synnax, then the warning when it goes back to
    DewesoftX mode. Leaves the 32xDO in Synnax mode."""

    channels: list[int]

    def setup(self) -> None:
        self.channels = []
        slaves = prepare_bus(self.client)
        self.stg = slaves[STGS[0]]
        self.slave = slaves[SWITCHED]

    def run(self) -> None:
        write_mode(self.client, self.stg, self.slave, DEWESOFTX_MODE)
        self.test_start_gives_switch_advice()
        self.test_picked_sync_mode_gives_sync_advice_first()
        self.test_switch()
        self.test_warns_after_going_back()

    def teardown(self) -> None:
        with self._try_to("clear the sync mode of the 32xDO"):
            props = self.client.devices.retrieve(key=self.slave.key).properties
            if "op_mode" in props:
                self._pick_op_mode(None)
        with self._try_to("put the 32xDO back in Synnax mode"):
            if scanned_mode(self.client, self.slave) != SYNNAX_MODE:
                write_mode(self.client, self.stg, self.slave, SYNNAX_MODE)
        with self._try_to("delete the task channels"):
            delete_channels(self.client, self.channels)
        super().teardown()

    def test_start_gives_switch_advice(self) -> None:
        """Checks that the start error tells the user to switch the device, and not to
        pick another sync mode."""
        self.log("Testing: Switch advice at the start")
        self._expect_start_error(f"{LEFT_OUT} {SWITCH_ADVICE}")

    def test_picked_sync_mode_gives_sync_advice_first(self) -> None:
        """Checks that a picked sync mode puts the sync advice before the switch
        advice, since either can be the cause."""
        self.log("Testing: Sync advice before switch advice")
        op_modes = self.slave.properties.get("op_modes")
        assert isinstance(op_modes, list)
        dc = next(
            (
                i
                for i, m in enumerate(op_modes)
                if isinstance(m, dict) and m.get("assign_activate") not in (None, 0)
            ),
            None,
        )
        if dc is None:
            self.fail(f"Expected slave {SWITCHED} to list a DC sync mode")
        mode = op_modes[dc]
        assert isinstance(mode, dict)
        self._pick_op_mode({"index": dc, "assign_activate": mode["assign_activate"]})
        try:
            self._expect_start_error(f"{LEFT_OUT} {SYNC_ADVICE} {SWITCH_ADVICE}")
        finally:
            self._pick_op_mode(None)

    def test_switch(self) -> None:
        """Checks that the switch reads back Synnax mode, and asks for no power
        restart, as the entries of the 32xDO are the same in both modes."""
        self.log("Testing: Switch to Synnax")
        scan = self.client.tasks.retrieve(
            type=SCAN_TYPE, rack=self.slave.rack, one=True
        )
        status = scan.execute_command_sync(
            "set_standalone", args={"device": self.slave.key}, timeout=START_TIMEOUT
        )
        expected = (
            "success",
            f"Switched {self.slave.name} to Synnax only",
            f"The device in slot {SWITCHED} runs Synnax only now.",
        )
        got = (status.variant, status.message, status.description)
        if got != expected:
            self.fail(f"Expected {expected}, but got {got}")
        read = sy.poll(
            lambda: (
                True if scanned_mode(self.client, self.slave) == SYNNAX_MODE else None
            ),
            timeout=_STATUS_TIMEOUT,
            interval=0.5,
        )
        if read is None:
            self.fail(f"Expected the scan to read Synnax mode from slot {SWITCHED}")

    def test_warns_after_going_back(self) -> None:
        """Checks that the scan warns when the 32xDO goes back to DewesoftX mode after
        the switch."""
        self.log("Testing: Warning after going back to DewesoftX mode")
        write_mode(self.client, self.stg, self.slave, DEWESOFTX_MODE)
        at = self.client.devices.retrieve(key=self.slave.key).properties.get(
            "switched_at"
        )
        if not isinstance(at, (int, float)):
            self.fail(f"Expected slot {SWITCHED} to hold the time of its switch")
        iso = sy.TimeStamp(int(at)).datetime(UTC).isoformat()
        expected = (
            "warning",
            "Set up for DewesoftX again",
            f"The device was switched to Synnax at {iso[:10]} {iso[11:16]} UTC, and "
            "then went back to DewesoftX mode. Switch it to Synnax again in "
            "Configure.",
        )
        got: tuple[str, str, str] | None = None

        def warned() -> bool | None:
            nonlocal got
            status = self.client.devices.retrieve(
                key=self.slave.key, include_status=True
            ).status
            if status is None:
                return None
            got = (status.variant, status.message, status.description)
            return True if got == expected else None

        if sy.poll(warned, timeout=_STATUS_TIMEOUT, interval=0.5) is None:
            self.fail(f"Expected {expected}, but got {got}")

    def _expect_start_error(self, expected: str) -> None:
        tsk = create_write_task(
            self.client,
            self.slave,
            name="HITL EtherCAT Switch",
            state_rate=25 * sy.Rate.HZ,
        )
        self.channels += [
            k for ch in tsk.config.channels for k in (ch.cmd_channel, ch.state_channel)
        ]
        text = assert_start_rejected(
            self.client, tsk, f"a start on slot {SWITCHED}", timeout=START_TIMEOUT
        )
        if text != expected:
            self.fail(f"Expected {expected!r}, but got {text!r}")

    def _pick_op_mode(self, op_mode: dict[str, JsonValue] | None) -> None:
        """Sets the sync mode that the user picks for the 32xDO, or clears it."""
        action = (
            sy.device.remove(
                sy.device.RemovePayload(path=["properties"], key="op_mode")
            )
            if op_mode is None
            else sy.device.set(
                sy.device.SetPayload(path=["properties", "op_mode"], value=op_mode)
            )
        )
        self.client.devices.dispatch(self.slave.key, action)
