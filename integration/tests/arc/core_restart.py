#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import dataclasses
import socket

import synnax as sy
from framework import cores, drivers
from framework.test_case import TestCase
from framework.utils import create_virtual_channels

START = "core_restart_start_cmd"
PRESSURE = "core_restart_pressure"
HELD = "core_restart_held"
VENTED = "core_restart_vented"

ARC_SOURCE = f"""
import time

sequence main {{
    stage hold {{
        1 -> {HELD}
        time.interval(100ms) -> ({PRESSURE} < 0.003) => next
    }}
    stage vented {{
        1 -> {VENTED}
    }}
}}
{START} => main
"""

HIGH = 1.0
LOW = 0.0
CORE_UP_TIMEOUT = 60 * sy.TimeSpan.SECOND
STATUS_TIMEOUT = 30 * sy.TimeSpan.SECOND
VENT_TIMEOUT = 30 * sy.TimeSpan.SECOND


def _free_port() -> int:
    with socket.socket() as s:
        s.bind(("localhost", 0))
        return int(s.getsockname()[1])


class CoreRestart(TestCase):
    """A running Arc keeps its stage while its Core restarts under a standalone
    Driver, and moves on once the Core is back, with no new start command
    (SY-6065).

    The test runs a Core and a Driver of its own, so the shared Core is never
    restarted."""

    _core: cores.Process | None = None
    _driver: drivers.Process | None = None

    def setup(self) -> None:
        core_binary = self.options.core_binary
        driver_binary = self.options.driver_binary
        if core_binary is None or driver_binary is None:
            self.skip("needs --core-binary and --driver-binary")
        self.set_manual_timeout(240)
        connection = dataclasses.replace(
            self.synnax_connection,
            server_address="localhost",
            port=_free_port(),
            secure=False,
        )
        self._core = cores.Process(
            binary=core_binary,
            core=cores.Core(
                connection=connection, log_path=self.results_dir / "core.log"
            ),
            persisted=True,
            driver_disabled=True,
        )
        self._core.wait_ready()
        self._driver = drivers.Process(
            binary=driver_binary,
            connection=connection,
            log_path=self.results_dir / "driver.log",
        )
        self._scratch = connection.create_client()
        self._rack = self._wait_for_driver(since=sy.TimeStamp(0))
        create_virtual_channels(
            self._scratch,
            [
                (START, sy.DataType.UINT8),
                (PRESSURE, sy.DataType.FLOAT64),
                (HELD, sy.DataType.UINT8),
                (VENTED, sy.DataType.UINT8),
            ],
        )

    def _wait_for_driver(self, since: sy.TimeStamp) -> sy.Rack:
        """Return the rack of the standalone Driver once it reports itself running
        after since."""

        def running() -> sy.Rack | None:
            racks = self._scratch.racks.retrieve(embedded=False, include_status=True)
            for rack in racks:
                status = rack.status
                if (
                    status is not None
                    and status.details is not None
                    and status.details.running
                    and status.time > since
                ):
                    return rack
            return None

        rack = sy.poll(running, timeout=CORE_UP_TIMEOUT, interval=0.2)
        if rack is None:
            self.fail(f"the Driver did not report itself running after {since}")
        return rack

    def _status_of(self, key: sy.task.Key) -> sy.task.Status | None:
        return self._scratch.tasks.retrieve(key, include_status=True).status

    def _expect_running(self, key: sy.task.Key, since: sy.TimeStamp) -> None:
        """Fail unless a running status for the task is written after since. The
        Core keeps the status from before its restart, so an older one proves
        nothing, and can write a stopped one of its own before the Driver's."""
        last: sy.task.Status | None = None

        def fresh() -> bool | None:
            nonlocal last
            last = self._status_of(key)
            if last is None or last.time <= since or last.details is None:
                return None
            return True if last.details.running else None

        if sy.poll(fresh, timeout=STATUS_TIMEOUT, interval=0.1) is None:
            self.fail(f"no running task status after the restart: {last}")

    def _start(self, task: sy.task.Task) -> None:
        task.execute_command("start")

        def running() -> bool | None:
            status = self._status_of(task.key)
            if status is not None and status.variant == "error":
                self.fail(f"the Arc task failed to start: {status.message}")
            if status is not None and status.details and status.details.running:
                return True
            return None

        if sy.poll(running, timeout=STATUS_TIMEOUT, interval=0.1) is None:
            self.fail("the Arc task did not start")

    def run(self) -> None:
        assert self._core is not None
        arc = self._scratch.arcs.create(
            name="core_restart", text=sy.Text(raw=ARC_SOURCE), mode="text"
        )
        task = self._scratch.arcs.update_task(arc.key, rack=self._rack.key)
        assert task is not None
        self._start(task)

        self.log("Entering the hold stage with the pressure high")
        with self._scratch.open_streamer([HELD, VENTED]) as streamer:
            with self._scratch.open_writer(
                start=sy.TimeStamp.now(),
                channels=[START, PRESSURE],
                enable_auto_commit=True,
            ) as w:
                # The Arc reads a channel with no value as 0, which would leave the
                # stage at once.
                w.write({PRESSURE: HIGH})
                w.write({START: 1})
                self._read_until(streamer, HELD, STATUS_TIMEOUT)

        self.log("Restarting the Core")
        stopped = sy.TimeStamp.now()
        self._core.restart()
        restarted = sy.TimeStamp.now()
        self._scratch = self._core.core.connection.create_client()
        self._wait_for_driver(since=restarted)
        self.log("The Driver reconnected")
        self._expect_running(task.key, since=stopped)

        self.log("Dropping the pressure, which only the hold stage reads")
        with self._scratch.open_streamer([HELD, VENTED]) as streamer:
            with self._scratch.open_writer(
                start=sy.TimeStamp.now(),
                channels=[PRESSURE],
                enable_auto_commit=True,
            ) as w:
                timer = sy.Timer()
                while timer.elapsed() < VENT_TIMEOUT:
                    # The Arc can stream the pressure again only after a write, so
                    # the test writes it on each pass, as a sensor would.
                    w.write({PRESSURE: LOW})
                    frame = streamer.read(timeout=0.2)
                    if frame is None:
                        continue
                    if HELD in frame:
                        self.fail("the Arc entered the hold stage again")
                    if VENTED in frame:
                        break
                else:
                    self.fail(
                        "the Arc did not leave the hold stage: "
                        f"{self._status_of(task.key)}"
                    )
        self._expect_running(task.key, since=stopped)

    def _read_until(
        self, streamer: sy.Streamer, channel: str, timeout: sy.TimeSpan
    ) -> None:
        timer = sy.Timer()
        while timer.elapsed() < timeout:
            frame = streamer.read(timeout=0.2)
            if frame is not None and channel in frame:
                return
        self.fail(f"no value on {channel} in {timeout}")

    def teardown(self) -> None:
        try:
            if self._driver is not None:
                with self._try_to("stop the Driver"):
                    self._driver.stop()
            if self._core is not None:
                with self._try_to("stop the Core"):
                    cores.stop([self._core])
        finally:
            super().teardown()
