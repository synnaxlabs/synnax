#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Unpack

"""Base test case for headless Arc integration tests.

Drives the Arc program + task lifecycle directly through the Synnax Python client,
with no Console UI or Playwright involved.

Deployment note: the Arc runtime reports task status without echoing the triggering
command's key (only ``owner`` / ``details.running``), so the client's blocking
``task.start()`` never resolves and hangs. We send the start command asynchronously and
wait for a status that reports the task running.
"""

from abc import abstractmethod
from collections.abc import Sequence
from dataclasses import dataclass
from uuid import UUID

from pydantic import ValidationError

import synnax as sy
from framework.test_case import Config, TestCase, param
from framework.utils import create_virtual_channel
from synnax.status import Variant
from tests.driver.sim_daq_case import SimDaqCase
from tests.driver.task import run_and_expect_rejection
from x import random_name

START_TIMEOUT_SECONDS = 30


@dataclass
class ArcTaskHandle:
    """A created Arc program paired with the task that executes it."""

    name: str
    arc_key: UUID
    task: sy.task.Task


@dataclass
class ModuleHandle:
    """A created Arc module, imported by its path from an automation."""

    path: str
    arc_key: UUID


@dataclass
class DirectoryHandle:
    """A created directory of Arcs and the import path it prefixes."""

    path: str
    key: UUID


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
    rack: sy.Rack
    arc_name: str
    _arcs: list[ArcTaskHandle]
    _modules: list[ModuleHandle]
    _directories: list[DirectoryHandle]

    def __init__(self, **kwargs: Unpack[Config]) -> None:
        super().__init__(**kwargs)
        self._arcs = []
        self._modules = []
        self._directories = []

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
        self.set_manual_timeout(180)
        create_virtual_channel(
            self.client, self.start_cmd_channel, data_type=sy.DataType.UINT8
        )
        self.subscribe(self.subscribe_channels)
        super().setup()

    def _retrieve_rack(self) -> None:
        rack_key = param(self.params, "rack_key", 0)
        if rack_key:
            self.rack = self.client.racks.retrieve(rack_key)
        else:
            self.rack = self.client.racks.retrieve(embedded=False, limit=1)[0]
        self.log(f"Selecting rack: {self.rack.name} (key: {self.rack.key})")

    def load_arc(
        self,
        source: str,
        *,
        name_prefix: str,
        start: bool = True,
        trigger: str | None = None,
    ) -> str:
        """Create an Arc, bind it to the selected rack, and return its name.

        When ``start`` is True, a start command deploys it and this blocks until the
        runtime reports it running.
        """
        name = f"{name_prefix}_{random_name()}"
        self.log(f"Creating Arc: {name}")
        arc = self.client.arcs.create(name=name, text=sy.Text(raw=source), mode="text")
        task = self.client.arcs.update_task(arc.key, rack=self.rack.key)
        assert task is not None
        self._arcs.append(ArcTaskHandle(name=name, arc_key=arc.key, task=task))
        if start:
            self.start_arc_task(task)
            self.log(f"Arc is running: {name}")
        if trigger:
            self.writer.write(trigger, 1)
        return name

    def create_directory(
        self, name_prefix: str, *, parent: DirectoryHandle | None = None
    ) -> DirectoryHandle:
        """Create a directory of Arcs below parent, or below Arcs, and return it.

        Teardown deletes directories after the modules in them, newest first.
        """
        name = f"{name_prefix}_{random_name()}"
        directories = self.client.arcs.directories
        parent_key = parent.key if parent is not None else directories.root()
        self.log(f"Creating Arc directory: {name}")
        directory = directories.create(name, parent=parent_key)
        path = name if parent is None else f"{parent.path}.{name}"
        handle = DirectoryHandle(path=path, key=directory.key)
        self._directories.append(handle)
        return handle

    def load_module(
        self,
        source: str,
        *,
        name_prefix: str,
        parent: DirectoryHandle | None = None,
    ) -> str:
        """Create an Arc module below parent, or below Arcs, and return its path.

        The name is an identifier, so an automation imports the path as written
        and binds its last segment. Teardown deletes the module after the
        automations that import it.
        """
        name = f"{name_prefix}_{random_name()}"
        self.log(f"Creating Arc module: {name}")
        module = self.client.arcs.create(
            name=name, text=sy.Text(raw=source), mode="text", type="module"
        )
        path = name
        if parent is not None:
            self.client.arcs.directories.move(parent.key, module.resource_id)
            path = f"{parent.path}.{name}"
        self._modules.append(ModuleHandle(path=path, arc_key=module.key))
        return path

    def _module(self, path: str) -> ModuleHandle:
        for module in self._modules:
            if module.path == path:
                return module
        raise KeyError(f"no tracked module at {path!r}")

    def overwrite_module(self, path: str, source: str) -> None:
        """Replace the text of the module at path, which load_module created."""
        self.log(f"Overwriting Arc module: {path}")
        self.client.arcs.set_text(self._module(path).arc_key, source)

    def overwrite_automation(self, name: str, source: str) -> None:
        """Replace the text of the automation named name, which load_arc created."""
        self.log(f"Overwriting Arc: {name}")
        self.client.arcs.set_text(self._arc(name).arc_key, source)

    def wait_for_module_status(
        self, path: str, variant: Variant, timeout: sy.CrudeTimeSpan
    ) -> sy.Status:
        """Fail unless the compile status of the module at path reaches variant in
        time. Returns that status."""
        key = self._module(path).arc_key
        last: sy.Status | None = None

        def reached() -> sy.Status | None:
            nonlocal last
            module = self.client.arcs.retrieve(key=key, include_status=True)
            assert isinstance(module, sy.arc.Module)
            last = module.status
            if last is not None and last.variant == variant:
                return last
            return None

        status = sy.poll(reached, timeout=timeout, interval=0.05)
        if status is None:
            self.fail(f"module {path} status is {last}, not {variant}")
        return status

    def wait_for_config_hash_change(
        self, task_key: sy.task.Key, before: str, timeout: sy.CrudeTimeSpan
    ) -> None:
        """Fail unless the task's stored config hash moves away from before in time."""

        def moved() -> bool | None:
            return self.client.tasks.retrieve(task_key).config_hash != before or None

        if sy.poll(moved, timeout=timeout, interval=0.05) is None:
            self.fail(f"task {task_key} kept config hash {before}")

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
                    if status.owner != task.resource_id:
                        continue
                    if status.variant == "error":
                        raise AssertionError(
                            f"Arc task '{task.name}' failed to start: {status.message}"
                        )
                    if status.details is not None and status.details.running:
                        return
        raise TimeoutError(f"timed out waiting for Arc task '{task.name}' to start")

    def _assert_configure_error(
        self,
        source: str,
        *,
        name_prefix: str,
        substr: str,
        expect: Sequence[str] = (),
    ) -> None:
        """Start an Arc expected to be rejected on deploy and verify the error."""
        name = self.load_arc(source, name_prefix=name_prefix, start=False)
        handle = next(h for h in self._arcs if h.name == name)
        message = run_and_expect_rejection(
            self.client, handle.task, timeout=30 * sy.TimeSpan.SECOND
        )
        if message is None:
            self.fail(f"[{name_prefix}] deploy unexpectedly succeeded")
        assert substr in message, (
            f"[{name_prefix}] expected '{substr}' in error, got: {message}"
        )
        for expected in expect:
            assert expected in message, (
                f"[{name_prefix}] expected '{expected}' in error, got: {message}"
            )
        self.log(f"[{name_prefix}] Got expected error: {message}")

    def _arc(self, name: str) -> ArcTaskHandle:
        for handle in self._arcs:
            if handle.name == name:
                return handle
        raise KeyError(f"no tracked arc named {name!r}")

    def task_key(self, name: str) -> sy.task.Key:
        """Return the task key for a tracked Arc loaded via load_arc."""
        return self._arc(name).task.key

    def remove_arc(self, name: str) -> None:
        """Stop tracking the arc named name so teardown does not delete it.
        Use after deleting an arc through another path (e.g. the Console UI).
        """
        self._arcs = [handle for handle in self._arcs if handle.name != name]

    def run(self) -> None:
        self._retrieve_rack()
        self.arc_name = self.load_arc(
            self.arc_source,
            name_prefix=self.arc_name_prefix,
            trigger=self.start_cmd_channel,
        )
        self.verify_sequence_execution()

    def teardown(self) -> None:
        """Delete all tracked arcs, which deletes their tasks, then stop the simulator.

        Cleanup runs even on failure, and SimDaqCase.teardown always runs so the
        simulator is stopped even when a cleanup step raises.
        """
        try:
            for handle in reversed(self._arcs):
                with self._try_to(f"delete arc {handle.name}"):
                    self.client.arcs.delete(handle.arc_key)
            for module in reversed(self._modules):
                with self._try_to(f"delete module {module.path}"):
                    self.client.arcs.delete(module.arc_key)
            for directory in reversed(self._directories):
                with self._try_to(f"delete directory {directory.path}"):
                    self.client.arcs.directories.delete(directory.key)
            if self.end_cmd_channel and self.sim_daq is not None:
                self.log(f"Signaling simulator to stop via {self.end_cmd_channel}")
                with self._try_to("signal simulator stop"):
                    self.writer.write(self.end_cmd_channel, 1)
        finally:
            super().teardown()

    @abstractmethod
    def verify_sequence_execution(self) -> None:
        """Override to implement test-specific verification logic."""
