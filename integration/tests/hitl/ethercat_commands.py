#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Start and stop commands on EtherCAT tasks on the HITL station: an auto-start task,
a stop that meets a start, two starts on one network, and a repeated start."""

from pydantic import JsonValue

import synnax as sy
from framework.test_case import TestCase
from tests.driver.task import (
    StatusWatch,
    command_of,
    delete_channels,
    expect_status,
    is_running,
    status_text,
)
from tests.hitl.ethercat import (
    START_TIMEOUT,
    STG_DC,
    STGS,
    create_read_task,
    prepare_bus,
    set_op_mode,
)

_START_LIMIT = 45 * sy.TimeSpan.SECOND
"""A bus start gets 75% of the 60 s limit of its command."""
_SLACK = 5 * sy.TimeSpan.SECOND
_ACK_TIMEOUT = 10 * sy.TimeSpan.SECOND
_RETRY_CHECK = 5 * sy.TimeSpan.SECOND
"""A free-run start that still runs after this retries."""
_STOP_LIMIT = sy.TimeSpan.SECOND
_QUIET = 5 * sy.TimeSpan.SECOND
"""How long a test watches for a status that must not come."""
_BAD_ADDRESS = sy.ethercat.StartupParameter(
    index=0x5FFE, sub_index=0, data_type=sy.DataType.UINT8, value=0
)
"""An object that the 6xSTG does not have, so it aborts the write at once."""
_ABORT = "Object does not exist in the object dictionary"
_STOP_ENDED_START = "A stop ended the start before it finished."
_STOP_CANCELLED_START = "The stop cancelled the start."


def _free_run(stg: sy.Device) -> dict[str, JsonValue]:
    """Returns the choice of the free-run mode of a 6xSTG.

    :raises AssertionError: If the 6xSTG lists no free-run mode.
    """
    modes = stg.properties.get("op_modes")
    if isinstance(modes, list):
        for i, mode in enumerate(modes):
            if isinstance(mode, dict) and mode.get("assign_activate") == 0:
                return {
                    "index": i,
                    "assign_activate": 0,
                    "sync0_cycle": 0,
                    "sync1_cycle": 0,
                }
    raise AssertionError(f"Expected a free-run mode in the 6xSTG modes {modes}")


class EtherCATCommands(TestCase):
    """Sends start and stop commands to EtherCAT tasks and checks the answers."""

    stg: sy.Device
    tasks: list[sy.ethercat.ReadTask]

    def setup(self) -> None:
        self.tasks = []
        self.stg = prepare_bus(self.client)[STGS[0]]

    def run(self) -> None:
        self.test_auto_start_command()
        self.test_stop_keeps_start_error()
        set_op_mode(self.client, self.stg, _free_run(self.stg))
        self.test_two_starts()
        self.test_stop_during_retry()
        self.test_repeat_start()

    def teardown(self) -> None:
        with self._try_to("put back the DC mode of the 6xSTG"):
            set_op_mode(self.client, self.stg, STG_DC)
        for tsk in self.tasks:
            with self._try_to(f"delete task {tsk.name}"):
                self.client.tasks.delete(tsk.key)
        with self._try_to("delete the task channels"):
            delete_channels(
                self.client,
                [ch.channel for tsk in self.tasks for ch in tsk.config.channels],
            )
        super().teardown()

    def test_auto_start_command(self) -> None:
        """Checks that a start command on an auto-start task starts it once, and that
        each status of the start answers the command."""
        self.log("Testing: Start command on an auto-start task")
        tsk = self._create_task("Auto Start Command", auto_start=True)
        with StatusWatch(self.client, [tsk.key]) as watch:
            key = tsk._internal.execute_command("start")
            started = watch.answer(key, timeout=_START_LIMIT + _SLACK)
            watch.collect(_QUIET)
        expect_status(
            started,
            variant="success",
            message="Started task",
            description="",
            running=True,
        )
        if others := [s for s in watch.seen if command_of(s) != key]:
            raise AssertionError(
                f"Expected each status to answer start command {key}, but "
                f"{[status_text(s) for s in others]} did not"
            )
        if len([s for s in watch.seen if s.variant != "loading"]) != 1:
            raise AssertionError(f"Expected one start, but saw {watch.texts()}")
        tsk.stop(timeout=START_TIMEOUT)

    def test_stop_keeps_start_error(self) -> None:
        """Checks that a start that fails on its own while a stop waits keeps its
        error, and that the stop answers with that error."""
        self.log("Testing: Stop during a start that fails")
        tsk = self._create_task(
            "Bad Parameter",
            startup_parameters=[
                sy.ethercat.SlaveStartupParameters(
                    device=self.stg.key, parameters=[_BAD_ADDRESS]
                )
            ],
        )
        with StatusWatch(self.client, [tsk.key]) as watch:
            start = tsk._internal.execute_command("start")
            watch.step(start, timeout=_ACK_TIMEOUT)
            stop = tsk._internal.execute_command("stop")
            failed = watch.answer(start, timeout=_START_LIMIT + _SLACK)
            stopped = watch.answer(stop, timeout=_ACK_TIMEOUT)
        if failed.message == "Stopped task":
            raise AssertionError(
                "Expected the bad startup parameter to fail the start, but the stop "
                "ended it first. The case needs a slower failure."
            )
        if (failed.variant, failed.message, is_running(failed)) != (
            "error",
            "Failed to start task",
            False,
        ) or _ABORT not in failed.description:
            raise AssertionError(
                f'Expected "Failed to start task" with {_ABORT!r}, but got '
                f"{failed.variant}: {status_text(failed)}"
            )
        expect_status(
            stopped,
            variant=failed.variant,
            message=failed.message,
            description=failed.description,
            running=False,
        )

    def test_two_starts(self) -> None:
        """Checks that of two starts sent at once on one network, the start that
        waits for the other fails at its limit and names the other start."""
        self.log("Testing: Two starts on one network")
        tasks = [self._create_task("First Start"), self._create_task("Second Start")]
        expected = (
            f"Another task is still starting on {self.stg.properties['network']}."
        )
        with StatusWatch(self.client, [t.key for t in tasks]) as watch:
            timer = sy.Timer()
            keys = [t._internal.execute_command("start") for t in tasks]
            self._expect_retry(watch, keys)
            answers = [
                watch.answer(k, timeout=_START_LIMIT + _SLACK - timer.elapsed())
                for k in keys
            ]
        waiting = [a for a in answers if a.description == expected]
        if len(waiting) != 1:
            raise AssertionError(
                f"Expected one start to fail with {expected!r}, but the starts "
                f"answered {[status_text(a) for a in answers]}"
            )
        expect_status(
            waiting[0],
            variant="error",
            message="Failed to start task",
            description=expected,
            running=False,
        )

    def test_stop_during_retry(self) -> None:
        """Checks that a stop ends a start that retries within a second."""
        self.log("Testing: Stop during the retries of a start")
        tsk = self._create_task("Stop During Retry")
        with StatusWatch(self.client, [tsk.key]) as watch:
            start = tsk._internal.execute_command("start")
            self._expect_retry(watch, [start])
            stop = tsk._internal.execute_command("stop")
            ended = watch.answer(start, timeout=_STOP_LIMIT)
            stopped = watch.answer(stop, timeout=_ACK_TIMEOUT)
        expect_status(
            ended,
            variant="success",
            message="Stopped task",
            description=_STOP_ENDED_START,
            running=False,
        )
        expect_status(
            stopped,
            variant="success",
            message="Stopped task",
            description=_STOP_CANCELLED_START,
            running=False,
        )

    def test_repeat_start(self) -> None:
        """Checks that a start sent while a start runs joins it, and that each start
        gets one answer, the same for both."""
        self.log("Testing: Repeated start during a start")
        tsk = self._create_task("Repeat Start")
        with StatusWatch(self.client, [tsk.key]) as watch:
            first = tsk._internal.execute_command("start")
            self._expect_retry(watch, [first])
            second = tsk._internal.execute_command("start")
            watch.step(second, timeout=_ACK_TIMEOUT)
            tsk._internal.execute_command("stop")
            for key in (first, second):
                expect_status(
                    watch.answer(key, timeout=_ACK_TIMEOUT),
                    variant="success",
                    message="Stopped task",
                    description=_STOP_ENDED_START,
                    running=False,
                )
            watch.collect(_QUIET)
        answers = [
            s
            for s in watch.seen
            if command_of(s) in (first, second) and s.variant != "loading"
        ]
        if len(answers) != 2:
            raise AssertionError(
                f"Expected one answer for each start, but saw {watch.texts()}"
            )

    def _expect_retry(self, watch: StatusWatch, keys: list[str]) -> None:
        """Checks that each start still runs after _RETRY_CHECK.

        :raises AssertionError: If a start ends, as the config then does not retry.
        """
        for key in keys:
            watch.step(key, timeout=_ACK_TIMEOUT)
        watch.collect(_RETRY_CHECK)
        if ended := [
            s for s in watch.seen if command_of(s) in keys and s.variant != "loading"
        ]:
            raise AssertionError(
                f"Expected the start to retry for {_RETRY_CHECK}, but it ended with "
                f"{[status_text(s) for s in ended]}. A free-run 6xSTG does not retry, "
                "so the start limit tests need another config."
            )

    def _create_task(
        self,
        label: str,
        *,
        auto_start: bool = False,
        startup_parameters: list[sy.ethercat.SlaveStartupParameters] | None = None,
    ) -> sy.ethercat.ReadTask:
        tsk = create_read_task(
            self.client,
            [self.stg],
            name=f"HITL EtherCAT Commands {label}",
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            auto_start=auto_start,
            startup_parameters=startup_parameters,
        )
        self.client.tasks.configure(tsk)
        self.tasks.append(tsk)
        return tsk
