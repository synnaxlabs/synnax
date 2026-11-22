#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Start and stop commands on EtherCAT tasks on the HITL station: an auto-start task and
a stop sent while a start fails."""

import re

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
    STGS,
    create_read_task,
    prepare_bus,
)

_START_LIMIT = 45 * sy.TimeSpan.SECOND
"""A bus start gets 75% of the 60 s limit of its command."""
_SLACK = 5 * sy.TimeSpan.SECOND
_STOP_ANSWER = 10 * sy.TimeSpan.SECOND
"""How long the answer of the stop may trail the answer of the start."""
_QUIET = 5 * sy.TimeSpan.SECOND
"""How long a test watches for a status that must not come."""
_BAD_ADDRESS = sy.ethercat.StartupParameter(
    index=0x5FFE, sub_index=0, data_type=sy.DataType.UINT8, value=0
)
"""An object that the 6xSTG does not have, so it aborts the write at once."""
_ABORT = "Object does not exist in the object dictionary"
_ACTIVATING = re.compile(r"Bringing \d+ modules? to OP\.")
"""The description of the start step right before the bus writes the startup
parameters."""


class _Command(TestCase):
    """Sends commands to read tasks of the first 6xSTG."""

    stg: sy.Device
    tasks: list[sy.ethercat.ReadTask]

    def setup(self) -> None:
        self.tasks = []
        self.stg = prepare_bus(self.client)[STGS[0]]

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


class EtherCATAutoStartCommand(_Command):
    """Checks that a start command on an auto-start task starts it once, and that each
    status of the start answers the command."""

    def run(self) -> None:
        tsk = self._create_task("Auto Start", auto_start=True)
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


class EtherCATStopKeepsStartError(_Command):
    """Checks that a stop sent during the try that fails a start keeps the start's
    error, and that the stop answers with that error. The statuses do not tell whether
    the stop met the open start or came after the failure."""

    def run(self) -> None:
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
            watch.wait(
                lambda s: (
                    command_of(s) == start
                    and _ACTIVATING.fullmatch(s.description) is not None
                ),
                timeout=_START_LIMIT,
                expected=f"a step of command {start} that brings the bus to OP",
            )
            stop = tsk._internal.execute_command("stop")
            failed = watch.answer(start, timeout=_START_LIMIT + _SLACK)
            stopped = watch.answer(stop, timeout=_STOP_ANSWER)
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
