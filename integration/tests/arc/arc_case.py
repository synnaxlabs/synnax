#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import abstractmethod
from dataclasses import dataclass

import synnax as sy
from console.case import ConsoleCase
from framework.models import STATUS
from framework.utils import create_virtual_channel
from tests.driver.sim_daq_case import SimDaqCase
from x import random_name


@dataclass
class _ArcHandle:
    name: str
    started: bool


class ArcConsoleCase(SimDaqCase, ConsoleCase):
    """Base class for Arc Console integration tests."""

    arc_source: str
    arc_name_prefix: str
    start_cmd_channel: str
    end_cmd_channel: str = ""
    subscribe_channels: list[str]
    rack: sy.Rack | None
    _arcs: list[_ArcHandle]

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
        self._create_start_cmd_channel()
        self.subscribe(self.subscribe_channels)
        super().setup()

    def _create_start_cmd_channel(self) -> None:
        """Create the virtual command channel that triggers the sequence."""
        create_virtual_channel(self.client, self.start_cmd_channel, sy.DataType.UINT8)

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
        configure: bool = True,
    ) -> str:
        """Create, configure, and optionally start an arc. Auto-cleaned in teardown."""
        assert self.rack is not None, "Call _retrieve_rack() before load_arc()"
        name = f"{name_prefix}_{random_name()}"
        self.log(f"Creating Arc: {name}")
        self.console.arc.create(name, source, mode="Text")
        self.console.arc.select_rack(self.rack.name)
        if configure:
            try:
                self.console.arc.configure()
            except Exception as e:
                status = self.console.arc.get_status()
                self.fail(f"Configure failed. Status: '{status}', Exception: {e}")
        if start:
            try:
                self.console.arc.start()
            except Exception as e:
                status = self.console.arc.get_status()
                self.fail(f"Start failed. Status: '{status}', Exception: {e}")
            self.log(f"Arc is running: {self.console.arc.is_running()}")
        if trigger:
            self.writer.write(trigger, 1)
        self._arcs.append(_ArcHandle(name=name, started=start))
        return name

    def stop_arc(self, name: str) -> None:
        """Stop a running arc by name."""
        self.console.arc.open(name)
        if self.console.arc.is_running():
            self.console.arc.stop()
        for arc in self._arcs:
            if arc.name == name:
                arc.started = False
                break

    def rename_arc(self, old_name: str, new_name: str) -> None:
        """Rename an arc and update tracking."""
        self.console.arc.rename(old_name=old_name, new_name=new_name)
        for arc in self._arcs:
            if arc.name == old_name:
                arc.name = new_name
                arc.started = False  # Rename stops the arc
                break

    def remove_arc(self, name: str) -> None:
        """Remove an arc from tracking (after manual delete)."""
        self._arcs = [a for a in self._arcs if a.name != name]

    def run(self) -> None:
        self._retrieve_rack()
        self.arc_name = self.load_arc(
            self.arc_source,
            self.arc_name_prefix,
            trigger=self.start_cmd_channel,
        )
        self.verify_sequence_execution()

    def teardown(self) -> None:
        """Clean up all tracked arcs. Called even if test fails."""
        if self._status == STATUS.FAILED:
            self.console.screenshot(f"failure_{self.name}")

        for arc in reversed(self._arcs):
            if arc.started:
                try:
                    self.console.arc.open(arc.name)
                    if self.console.arc.is_running():
                        self.console.arc.stop()
                except Exception as e:
                    self.log(f"Failed to stop {arc.name}: {e}")
            try:
                self.console.arc.delete(arc.name)
            except Exception as e:
                self.log(f"Failed to delete {arc.name}: {e}")

        if self.end_cmd_channel and self.sim_daq is not None:
            self.log(f"Signaling simulator to stop via {self.end_cmd_channel}")
            try:
                self.writer.write(self.end_cmd_channel, 1)
            except Exception as e:
                self.log(f"Failed to signal simulator stop: {e}")

        super().teardown()

    @abstractmethod
    def verify_sequence_execution(self) -> None:
        """Override to implement test-specific verification logic."""
        pass
