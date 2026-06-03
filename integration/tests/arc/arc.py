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
command's key (only ``details.task`` / ``details.running``; see
``core/pkg/service/arc/runtime/task.go``), so the client's blocking ``task.start()``
never resolves and hangs. We deploy with ``auto_start`` instead -- the runtime starts
the task during configuration and emits the success status ``tasks.configure`` waits on.
"""

from abc import abstractmethod
from dataclasses import dataclass
from uuid import UUID

import synnax as sy
from framework.test_case import TestCase
from framework.utils import create_virtual_channel
from tests.driver.sim_daq_case import SimDaqCase
from x import random_name

ARC_TASK_TYPE = sy.arc.Task.TYPE
CONFIGURE_TIMEOUT_SECONDS = 30


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
    _arcs: list[ArcTaskHandle]

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
        self.set_manual_timeout(180)
        create_virtual_channel(self.client, self.start_cmd_channel, sy.DataType.UINT8)
        self.subscribe(self.subscribe_channels)
        super().setup()

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
        """Create an Arc, deploy its task on the selected rack, and return its name.

        The task is configured on ``self.rack`` and, when ``start`` is True,
        auto-started during configuration.
        """
        assert self.rack is not None, "Call _retrieve_rack() before load_arc()"
        name = f"{name_prefix}_{random_name()}"
        self.log(f"Creating Arc: {name}")
        arc = self.client.arcs.create(name=name, text=sy.Text(raw=source), mode="text")
        task = sy.task.Task(
            rack=self.rack.key,
            name=name,
            type=ARC_TASK_TYPE,
            config={"arc_key": str(arc.key), "auto_start": start},
        )
        self.client.tasks.configure(task, timeout=CONFIGURE_TIMEOUT_SECONDS)
        self.client.ontology.add_children(arc.ontology_id, task.ontology_id)
        if start:
            self.log(f"Arc is running: {name}")
        if trigger:
            self.writer.write(trigger, 1)
        self._arcs.append(ArcTaskHandle(name=name, arc_key=arc.key, task=task))
        return name

    def run(self) -> None:
        self._retrieve_rack()
        self.load_arc(
            self.arc_source,
            self.arc_name_prefix,
            trigger=self.start_cmd_channel,
        )
        self.verify_sequence_execution()

    def teardown(self) -> None:
        """Delete all tracked arcs (cascades to their tasks). Runs even on failure."""
        for handle in reversed(self._arcs):
            try:
                self.client.arcs.delete(handle.arc_key)
            except Exception as e:
                self.fail(f"Failed to delete {handle.name}: {e}")

        if self.end_cmd_channel and self.sim_daq is not None:
            self.log(f"Signaling simulator to stop via {self.end_cmd_channel}")
            try:
                self.writer.write(self.end_cmd_channel, 1)
            except Exception as e:
                self.fail(f"Failed to signal simulator stop: {e}")

        super().teardown()

    @abstractmethod
    def verify_sequence_execution(self) -> None:
        """Override to implement test-specific verification logic."""
        pass
