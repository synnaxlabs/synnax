#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Base test case for headless Arc integration tests.

Drives the Arc program + task lifecycle directly through the Synnax Python client,
with no Console UI or Playwright involved.

Deployment note: the Arc runtime reports task status without echoing the triggering
command's key (only ``details.task`` / ``details.running``), so the client's blocking
``task.start()`` never resolves and hangs. We send the start command asynchronously and
wait for a status that reports the task running.
"""

from abc import abstractmethod
from dataclasses import dataclass
from uuid import UUID

from pydantic import ValidationError

import synnax as sy
from framework.task_status import TaskStatus
from framework.test_case import TestCase
from framework.utils import create_virtual_channel
from tests.driver.sim_daq_case import SimDaqCase
from x import random_name

ARC_TASK_TYPE = sy.arc.Task.TYPE
START_TIMEOUT_SECONDS = 30


@dataclass
class ArcTaskHandle:
    """A created Arc program paired with the task that executes it."""

    name: str
    arc_key: UUID
    task: sy.task.Task


class ArcCase(SimDaqCase, TestCase):
    """Base class for headless Arc integration tests.

    Drives the Arc lifecycle directly through the Synnax Python client (no Console
    UI or Playwright). A test supplies the Arc source and verification logic; the
    base handles rack selection, deployment, and teardown.
    """

    arc_source: str
    arc_name_prefix: str
    start_cmd_channel: str
    end_cmd_channel: str = ""
    subscribe_channels: list[str]
    rack: sy.Rack | None
    arc_name: str
    _arcs: list[ArcTaskHandle]

    # Opt-in: when True, setup records per-task status updates for the duration
    # of the test so wait_for_task_status / task_is_running can read them. This
    # belongs on a generic Task case once one exists.
    collect_task_status: bool = False
    _task_status: TaskStatus | None

    def setup(self) -> None:
        required = [
            "arc_source",
            "arc_name_prefix",
            "start_cmd_channel",
            "subscribe_channels",
        ]
        for attr in required:
            if not hasattr(self, attr):
                raise TypeError(
                    f"{self.__class__.__name__} must define class attribute '{attr}'"
                )
        self.rack = None
        self._arcs = []
        self._task_status = None
        self.set_manual_timeout(180)
        create_virtual_channel(self.client, self.start_cmd_channel, sy.DataType.UINT8)
        self.subscribe(self.subscribe_channels)
        super().setup()
        if self.collect_task_status:
            self._task_status = TaskStatus(self.client)
            self._task_status.open()

    def _retrieve_rack(self) -> None:
        rack_key = self.params.get("rack_key")
        if rack_key:
            self.rack = self.client.racks.retrieve(rack_key)
        else:
            self.rack = self.client.racks.retrieve(embedded=False)
        assert self.rack is not None, "Failed to retrieve rack"
        self.log(f"Selecting rack: {self.rack.name} (key: {self.rack.key})")

    def load_arc(
        self,
        source: str,
        name_prefix: str,
        *,
        start: bool = True,
        trigger: str | None = None,
    ) -> str:
        """Create an Arc, save its task on the selected rack, and return its name.

        The task row is saved against ``self.rack``; when ``start`` is True, a
        start command deploys it and this blocks until the runtime reports it
        running.
        """
        assert self.rack is not None, "Call _retrieve_rack() before load_arc()"
        name = f"{name_prefix}_{random_name()}"
        self.log(f"Creating Arc: {name}")
        arc = self.client.arcs.create(name=name, text=sy.Text(raw=source), mode="text")
        task = sy.task.Task(
            rack=self.rack.key,
            name=name,
            type=ARC_TASK_TYPE,
            config={"arc_key": str(arc.key)},
        )
        self._arcs.append(ArcTaskHandle(name=name, arc_key=arc.key, task=task))
        self.client.tasks.configure(task)
        self.client.ontology.add_children(arc.ontology_id, task.ontology_id)
        if start:
            self.start_arc_task(task)
            self.log(f"Arc is running: {name}")
        if trigger:
            self.writer.write(trigger, 1)
        return name

    def start_arc_task(self, task: sy.task.Task) -> None:
        """Start the task and block until the runtime reports it running.

        Raises AssertionError when the driver rejects the deploy, and
        TimeoutError when no status arrives in time.
        """
        with self.client.open_streamer(["sy_status_set"]) as streamer:
            task.execute_command("start")
            timer = sy.Timer()
            while timer.elapsed() < START_TIMEOUT_SECONDS * sy.TimeSpan.SECOND:
                frame = streamer.read(timeout=float(START_TIMEOUT_SECONDS))
                if frame is None:
                    break
                if "sy_status_set" not in frame:
                    continue
                for raw in frame["sy_status_set"]:
                    try:
                        status = sy.task.Status.model_validate(raw)
                    except ValidationError:
                        continue
                    if status.details is None or status.details.task != task.key:
                        continue
                    if status.variant == "error":
                        raise AssertionError(
                            f"Arc task '{task.name}' failed to start: {status.message}"
                        )
                    if status.details.running:
                        return
        raise TimeoutError(f"timed out waiting for Arc task '{task.name}' to start")

    def task_key(self, name: str) -> sy.task.Key:
        """Return the task key for a tracked Arc loaded via load_arc."""
        for handle in self._arcs:
            if handle.name == name:
                return handle.task.key
        raise KeyError(f"no tracked arc named {name!r}")

    def remove_arc(self, name: str) -> None:
        """Stop tracking the arc named name so teardown does not delete it.
        Use after deleting an arc through another path (e.g. the Console UI).
        """
        self._arcs = [handle for handle in self._arcs if handle.name != name]

    def wait_for_task_status(
        self, task_key: sy.task.Key, text: str, timeout: float = 5.0
    ) -> bool:
        """Return True if a status for task_key containing text has surfaced.

        Requires collect_task_status = True so the base lifecycle records task
        status updates for the duration of the test.
        """
        if self._task_status is None:
            raise RuntimeError(
                "wait_for_task_status requires collect_task_status = True"
            )
        return self._task_status.wait_for(task_key, text, timeout)

    def task_is_running(self, task_key: sy.task.Key) -> bool:
        """Return True if the latest status reports task_key as running.

        Requires collect_task_status = True so the base lifecycle records task
        status updates for the duration of the test.
        """
        if self._task_status is None:
            raise RuntimeError("task_is_running requires collect_task_status = True")
        return self._task_status.is_running(task_key)

    def run(self) -> None:
        self._retrieve_rack()
        self.arc_name = self.load_arc(
            self.arc_source,
            self.arc_name_prefix,
            trigger=self.start_cmd_channel,
        )
        self.verify_sequence_execution()

    def teardown(self) -> None:
        """Delete all tracked arcs and their tasks, then stop the simulator.

        Cleanup runs even on failure, and SimDaqCase.teardown always runs so the
        simulator is stopped even when a cleanup step raises. Tasks are deleted
        explicitly rather than relying on the arc-to-task cascade, since a
        rejected configuration never links the two.
        """
        try:
            for handle in reversed(self._arcs):
                if handle.task.key is not None:
                    with self._try_to(f"delete task {handle.name}"):
                        self.client.tasks.delete(handle.task.key)
                with self._try_to(f"delete arc {handle.name}"):
                    self.client.arcs.delete(handle.arc_key)
            if self.end_cmd_channel and self.sim_daq is not None:
                self.log(f"Signaling simulator to stop via {self.end_cmd_channel}")
                with self._try_to("signal simulator stop"):
                    self.writer.write(self.end_cmd_channel, 1)
            if self._task_status is not None:
                self._task_status.close()
        finally:
            super().teardown()

    @abstractmethod
    def verify_sequence_execution(self) -> None:
        """Override to implement test-specific verification logic."""
        pass
