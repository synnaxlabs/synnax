#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Test channel rename synchronization across UI elements.

Verifies that renaming a channel properly synchronizes across:
- Resources Toolbar (channel list)
- Line Plot Visualization Toolbar
- Log Visualization Toolbar
- Schematic Visualization Toolbar
- Table Visualization Toolbar

Note: Task Configuration Dialog is excluded as it requires hardware devices.
"""

import synnax as sy

from console.case import ConsoleCase
from console.log import Log
from console.plot import Plot
from console.schematic.schematic import Schematic
from console.schematic.value import Value
from console.table import Table
from framework.utils import get_random_name


class RenameSynchronization(ConsoleCase):
    """Test channel rename synchronization across UI elements."""

    suffix: str
    index_name: str
    data_name: str
    new_name: str

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self.index_name = f"sync_idx_{self.suffix}"
        self.data_name = f"sync_data_{self.suffix}"
        self.new_name = f"renamed_sync_{self.suffix}"

    def teardown(self) -> None:
        self.console.channels.delete([self.new_name, self.index_name])
        super().teardown()

    def run(self) -> None:
        """Run channel rename synchronization tests."""
        console = self.console
        client = self.client

        self.log("Creating test channels")
        console.channels.create(name=self.index_name, is_index=True)
        console.channels.create(
            name=self.data_name,
            data_type=sy.DataType.FLOAT32,
            index=self.index_name,
        )

        self.log("Setting up Line Plot with channel")
        plot = Plot(client, console, f"Sync Test Plot {self.suffix}")
        plot.add_channels("Y1", [self.data_name])

        self.log("Setting up Log with channel")
        log_page = Log(client, console, f"Sync Test Log {self.suffix}")
        log_page.set_channel(self.data_name)

        self.log("Setting up Schematic with channel")
        schematic = Schematic(client, console, f"Sync Test Schematic {self.suffix}")
        value_symbol = Value(label=self.data_name, channel_name=self.data_name)
        schematic.create_symbol(value_symbol)

        self.log("Setting up Table with channel")
        table = Table(client, console, f"Sync Test Table {self.suffix}")
        table.set_cell_channel(self.data_name)

        self.log("Renaming channel")
        console.channels.rename(names=self.data_name, new_names=self.new_name)

        self.log("Verifying sync in Resources Toolbar")
        console.channels.show_channels()
        new_item = console.channels._find_channel_item(self.new_name)
        assert (
            new_item is not None
        ), f"Renamed channel {self.new_name} should appear in Resources Toolbar"
        old_item = console.channels._find_channel_item(self.data_name)
        assert (
            old_item is None
        ), f"Old channel name {self.data_name} should not appear after rename"
        console.channels.hide_channels()

        ch = self.client.channels.retrieve(self.new_name)
        assert (
            ch.name == self.new_name
        ), f"Server should have channel with name {self.new_name}"

        self.log("Verifying sync in Line Plot Toolbar")
        assert plot.has_channel(
            "Y1", self.new_name
        ), f"Renamed channel {self.new_name} should appear in Line Plot toolbar"

        self.log("Verifying sync in Log Toolbar")
        assert log_page.has_channel(
            self.new_name
        ), f"Renamed channel {self.new_name} should appear in Log toolbar"

        self.log("Verifying sync in Schematic Toolbar")
        console.layout.get_tab(f"Sync Test Schematic {self.suffix}").click()
        props = value_symbol.get_properties()
        assert props["channel"] == self.new_name, (
            f"Value symbol should have renamed channel {self.new_name}, "
            f"got {props['channel']}"
        )

        self.log("Verifying sync in Table Toolbar")
        assert table.has_channel(
            self.new_name
        ), f"Renamed channel {self.new_name} should appear in Table toolbar"

        plot.close()
        log_page.close()
        schematic.close()
        table.close()
