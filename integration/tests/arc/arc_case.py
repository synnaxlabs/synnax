#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import abstractmethod

from console.case import ConsoleCase
from framework.sim_daq_case import SimDaqTestCase
from framework.utils import get_random_name

import synnax as sy


class ArcConsoleCase(SimDaqTestCase, ConsoleCase):
    """Base class for Arc Console integration tests."""

    arc_source: str
    arc_name_prefix: str
    start_cmd_channel: str
    end_cmd_channel: str
    subscribe_channels: list[str]
    rack: sy.Rack | None
    _arc_created: bool
    _arc_started: bool

    def setup(self) -> None:
        required = [
            "arc_source",
            "arc_name_prefix",
            "start_cmd_channel",
            "end_cmd_channel",
            "subscribe_channels",
        ]
        for attr in required:
            if not hasattr(self, attr):
                raise TypeError(
                    f"{self.__class__.__name__} must define class attribute '{attr}'"
                )
        self.arc_name = f"{self.arc_name_prefix}_{get_random_name()}"
        self.rack = None
        self._arc_created = False
        self._arc_started = False
        self.set_manual_timeout(180)
        self._create_start_cmd_channel()
        self.subscribe(self.subscribe_channels)
        super().setup()

    def _create_start_cmd_channel(self) -> None:
        """Create the virtual command channel that triggers the sequence."""
        self.client.channels.create(
            name=self.start_cmd_channel,
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        self.log(f"Creating Arc program: {self.arc_name}")
        self.console.arc.create(self.arc_name, self.arc_source, mode="Text")
        self._arc_created = True

        rack_key = self.params.get("rack_key")
        if rack_key:
            self.rack = self.client.racks.retrieve(rack_key)
        else:
            self.rack = self.client.racks.retrieve(embedded=False)
        assert self.rack is not None, "Failed to retrieve rack"

        self.log(f"Selecting rack: {self.rack.name} (key: {self.rack.key})")
        self.console.arc.select_rack(self.rack.name)

        self.log("Configuring Arc task")
        self.console.arc.configure()

        arc = self.client.arcs.retrieve(name=self.arc_name)
        self.log(f"Arc saved with key: {arc.key}")

        self.log("Starting Arc task")
        self.console.arc.start()
        self._arc_started = True
        self.log(f"Arc is running: {self.console.arc.is_running()}")

        self.log("Triggering sequence")
        with self.client.open_writer(sy.TimeStamp.now(), self.start_cmd_channel) as w:
            w.write(self.start_cmd_channel, 1)

        self.verify_sequence_execution()

        self.log(f"Arc sequence {arc.name} on {self.rack.name} completed")

    def teardown(self) -> None:
        """Clean up Arc resources. Called even if test fails."""
        if self._arc_started:
            self.log("Stopping Arc task")
            try:
                self.console.arc.stop()
            except Exception as e:
                self.fail(f"Failed to stop Arc task: {e}")

        if self._arc_created:
            self.log("Deleting Arc program")
            try:
                self.console.arc.delete(self.arc_name)
            except Exception as e:
                self.fail(f"Failed to delete Arc program: {e}")

        if hasattr(self, "end_cmd_channel") and self.end_cmd_channel:
            self.log(f"Signaling simulator to stop via {self.end_cmd_channel}")
            try:
                with self.client.open_writer(
                    sy.TimeStamp.now(), self.end_cmd_channel
                ) as w:
                    w.write(self.end_cmd_channel, 1)
            except Exception as e:
                self.fail(f"Failed to signal simulator stop: {e}")

        super().teardown()

    @abstractmethod
    def verify_sequence_execution(self) -> None:
        """Override to implement test-specific verification logic."""
        pass
