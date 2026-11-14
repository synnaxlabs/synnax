#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""EtherCAT tasks across a crash of the Driver on the HITL station. Each test kills the
embedded Driver of the Core under test, as a crash does, and the Core starts it again.
A kill stops the tasks of every integration on the Driver, so the case runs in an
isolated sequence."""

import re

import psutil

import synnax as sy
from framework.test_case import TestCase
from tests.driver.task import StatusWatch, delete_channels, expect_status, is_running
from tests.hitl.ethercat import START_TIMEOUT, STGS, create_read_task, prepare_bus

_DRIVER = re.compile(r"driver-[0-9a-f]{16}(\.exe)?")
"""The name of the embedded Driver process, which the Core writes from its own copy."""
_SCAN_TYPES = frozenset(
    {
        "ethercat_scan",
        "http_scan",
        "labjack_scan",
        "modbus_scan",
        "ni_scanner",
        "opc_scan",
    }
)
_EXIT_TIMEOUT = 10 * sy.TimeSpan.SECOND
_ACK_TIMEOUT = 10 * sy.TimeSpan.SECOND
_RESTART_TIMEOUT = sy.TimeSpan.MINUTE
"""The Core starts the Driver 2 s after it exits, and the Driver then deploys each
task."""
_QUIET = 5 * sy.TimeSpan.SECOND
"""How long a test watches for a status that must not come."""
_BOOT = 10 * sy.TimeSpan.SECOND
"""How long the Driver takes to deploy its tasks after its first status."""
_LOST_START_LIMIT = 12 * sy.TimeSpan.SECOND
"""A start sent while the Driver restarts must end in about 10 s, either way."""


class EtherCATDriverRestarts(TestCase):
    """Kills the Driver while a task runs, while it starts, with an auto-start task,
    and before a start, and checks the statuses that the Driver writes when it is
    back."""

    slaves: dict[int, sy.Device]
    tasks: list[sy.ethercat.ReadTask]

    def setup(self) -> None:
        self.tasks = []
        self.slaves = prepare_bus(self.client)

    def run(self) -> None:
        self.test_crash_while_running()
        self.test_crash_while_starting()
        self.test_start_during_restart()
        self.test_boot_scan_tasks()
        # Runs last: its task starts again at each later boot of the Driver.
        self.test_boot_with_auto_start()

    def teardown(self) -> None:
        for tsk in self.tasks:
            with self._try_to(f"delete task {tsk.name}"):
                self.client.tasks.delete(tsk.key)
        with self._try_to("delete the task channels"):
            delete_channels(
                self.client,
                [ch.channel for tsk in self.tasks for ch in tsk.config.channels],
            )
        super().teardown()

    def test_crash_while_running(self) -> None:
        """Checks that a task that ran when the Driver crashed shows as stopped, and
        that the Driver does not start it again."""
        self.log("Testing: Crash while a task runs")
        tsk = self._create_task("Running")
        tsk.start(timeout=START_TIMEOUT)
        with StatusWatch(self.client, [tsk.key]) as watch:
            self._wait_for_driver(self._kill_driver())
            stopped = watch.wait(
                lambda s: not is_running(s),
                timeout=_RESTART_TIMEOUT,
                expected="a status that the task does not run",
            )
            expect_status(
                stopped,
                variant="warning",
                message="Task stopped",
                description="The Driver restarted while the task was running. Start "
                "the task again.",
                running=False,
            )
            if started := [s for s in watch.collect(_QUIET) if is_running(s)]:
                raise AssertionError(
                    f"Expected the Driver to leave the task stopped, but it wrote "
                    f"{[s.message for s in started]}"
                )
        tsk.start(timeout=START_TIMEOUT)
        tsk.stop(timeout=START_TIMEOUT)

    def test_crash_while_starting(self) -> None:
        """Checks that a start that the crash cut off fails, and that the failure
        answers the start command, so the client stops waiting."""
        self.log("Testing: Crash while a task starts")
        tsk = self._create_task("Starting")
        with StatusWatch(self.client, [tsk.key]) as watch:
            key = tsk._internal.execute_command("start")
            watch.step(key, timeout=_ACK_TIMEOUT)
            self._wait_for_driver(self._kill_driver())
            expect_status(
                watch.answer(key, timeout=_RESTART_TIMEOUT),
                variant="error",
                message="Failed to start task",
                description="The Driver restarted while the task was starting. Start "
                "the task again.",
                running=False,
            )

    def test_boot_with_auto_start(self) -> None:
        """Checks that an auto-start task ends on its own start after a crash, and is
        never shown as stopped."""
        self.log("Testing: Crash with an auto-start task")
        tsk = self._create_task("Auto Start", auto_start=True)
        tsk.start(timeout=START_TIMEOUT)
        with StatusWatch(self.client, [tsk.key]) as watch:
            self._wait_for_driver(self._kill_driver())
            started = watch.wait(
                lambda s: s.variant != "loading",
                timeout=_RESTART_TIMEOUT,
                expected="a final status of the start",
            )
            expect_status(
                started,
                variant="success",
                message="Started task",
                description="",
                running=True,
            )
            watch.collect(_QUIET)
            if any(s.message == "Task stopped" for s in watch.seen):
                raise AssertionError(
                    f'Expected no "Task stopped" for an auto-start task, but saw '
                    f"{watch.texts()}"
                )
        tsk.stop(timeout=START_TIMEOUT)

    def test_boot_scan_tasks(self) -> None:
        """Checks that no scan task shows as stopped after a crash."""
        self.log("Testing: Crash with the scan tasks")
        rack = self.client.racks.retrieve_embedded_rack()
        scans = [
            t for t in self.client.tasks.list(rack=rack.key) if t.type in _SCAN_TYPES
        ]
        ethercat = next((t for t in scans if t.type == "ethercat_scan"), None)
        if ethercat is None:
            raise AssertionError(
                f"Expected an EtherCAT scan task on rack {rack.name}, but found "
                f"{[t.type for t in scans]}"
            )
        with StatusWatch(self.client, [t.key for t in scans]) as watch:
            self._wait_for_driver(self._kill_driver())
            owner = sy.task.resource_id(ethercat.key)
            watch.wait(
                lambda s: s.owner == owner,
                timeout=_RESTART_TIMEOUT,
                expected="a status of the EtherCAT scan task",
            )
            watch.collect(_BOOT)
            if any(s.message == "Task stopped" for s in watch.seen):
                raise AssertionError(
                    f'Expected no "Task stopped" for a scan task, but saw '
                    f"{watch.texts()}"
                )

    def test_start_during_restart(self) -> None:
        """Checks that a start sent while the Driver restarts ends in about 10 s, with
        an answer or an error, and does not wait for the full timeout."""
        self.log("Testing: Start while the Driver restarts")
        tsk = self._create_task("Lost Start")
        old = self._kill_driver()
        timer = sy.Timer()
        outcome = "started"
        try:
            tsk.start(timeout=START_TIMEOUT)
        except (RuntimeError, TimeoutError) as e:
            outcome = f"{type(e).__name__}: {e}"
        elapsed = timer.elapsed()
        self.log(f"The start ended after {elapsed}: {outcome}")
        self._wait_for_driver(old)
        if elapsed > _LOST_START_LIMIT:
            raise AssertionError(
                f"Expected the start to end within {_LOST_START_LIMIT}, but it ended "
                f"after {elapsed}: {outcome}"
            )
        if outcome == "started":
            tsk.stop(timeout=START_TIMEOUT)

    def _create_task(
        self, label: str, *, auto_start: bool = False
    ) -> sy.ethercat.ReadTask:
        tsk = create_read_task(
            self.client,
            [self.slaves[pos] for pos in STGS],
            name=f"HITL EtherCAT Restart {label}",
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            auto_start=auto_start,
        )
        self.client.tasks.configure(tsk)
        self.tasks.append(tsk)
        return tsk

    def _drivers(self) -> list[psutil.Process]:
        """Returns the Driver processes whose Core listens on the port under test, so a
        kill never reaches another Core's Driver."""
        port = self.synnax_connection.port
        drivers: list[psutil.Process] = []
        for p in psutil.process_iter(["name"]):
            if not _DRIVER.fullmatch(p.info["name"] or ""):
                continue
            try:
                core = p.parent()
                if core is not None and any(
                    c.status == psutil.CONN_LISTEN and c.laddr.port == port
                    for c in core.net_connections(kind="tcp")
                ):
                    drivers.append(p)
            except psutil.Error:
                continue
        return drivers

    def _kill_driver(self) -> int:
        """Kills the embedded Driver and returns its process ID once it exits.

        :raises AssertionError: If the Core under test does not run one Driver.
        """
        drivers = self._drivers()
        if len(drivers) != 1:
            raise AssertionError(
                f"Expected one embedded Driver, but found {[p.pid for p in drivers]}"
            )
        driver = drivers[0]
        driver.kill()
        driver.wait(timeout=_EXIT_TIMEOUT.seconds)
        return driver.pid

    def _wait_for_driver(self, old: int) -> None:
        """Returns once a Driver other than the process old runs.

        :raises TimeoutError: If the Core does not start the Driver again in time.
        """
        started = sy.poll(
            lambda: next((p for p in self._drivers() if p.pid != old), None),
            timeout=_RESTART_TIMEOUT,
            interval=0.2,
        )
        if started is None:
            raise TimeoutError(
                f"Expected the Core to start the Driver within {_RESTART_TIMEOUT}"
            )
