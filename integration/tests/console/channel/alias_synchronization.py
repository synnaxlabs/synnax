#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Test channel alias synchronization across UI elements.

Verifies that setting and removing an alias for a channel properly synchronizes across:
- Resources Toolbar (channel list)
- Line Plot Visualization Toolbar

Note: Log, Schematic, and Table toolbars are excluded as aliases do not sync there.
Note: Task Configuration Dialog is excluded as it requires hardware devices.
"""

import uuid

import synnax as sy

from console.case import ConsoleCase
from console.plot import Plot


class AliasSynchronization(ConsoleCase):
    """Test channel alias synchronization across UI elements."""

    # SY-3584: Verify alias for Logs Schematics, Tables

    prefix: str
    range_name: str
    index_name: str
    data_name: str
    alias_name: str

    def setup(self) -> None:
        super().setup()
        self.prefix = str(uuid.uuid4())[:6]
        self.range_name = f"alias_sync_range_{self.prefix}"
        self.index_name = f"alias_sync_idx_{self.prefix}"
        self.data_name = f"alias_sync_data_{self.prefix}"
        self.alias_name = f"AliasSync_{self.prefix}"

    def teardown(self) -> None:
        self.console.channels.delete([self.data_name, self.index_name])
        self.console.ranges.open_explorer()
        self.console.ranges.delete_from_explorer(self.range_name)
        super().teardown()

    def run(self) -> None:
        """Run channel alias synchronization tests."""
        console = self.console
        client = self.client

        self.log("Creating range and setting it active")
        console.ranges.create(self.range_name, persisted=True)
        console.ranges.open_explorer()
        console.ranges.favorite_from_explorer(self.range_name)
        console.ranges.show_toolbar()
        console.ranges.set_active(self.range_name)

        self.log("Creating test channels")
        console.channels.create(name=self.index_name, is_index=True)
        console.channels.create(
            name=self.data_name,
            data_type=sy.DataType.FLOAT32,
            index=self.index_name,
        )

        self.log("Setting up Line Plot with channel")
        plot = Plot(client, console, f"Alias Test Plot {self.prefix}")
        plot.add_channels("Y1", [self.data_name])

        self.log("Setting alias for channel")
        console.channels.set_alias(self.data_name, self.alias_name)

        self.log("Verifying sync in Resources Toolbar")
        console.channels.show_channels()
        alias_item = console.channels._find_channel_item(self.alias_name)
        assert (
            alias_item is not None
        ), f"Alias {self.alias_name} should appear in Resources Toolbar"
        console.channels.hide_channels()

        rng = client.ranges.retrieve(name=self.range_name)
        data_ch = client.channels.retrieve(self.data_name)
        scoped_ch = rng[self.alias_name]
        assert (
            scoped_ch.key == data_ch.key
        ), f"Alias should resolve to channel key {data_ch.key}"

        self.log("Verifying sync in Line Plot Toolbar")
        assert plot.has_channel(
            "Y1", self.alias_name
        ), f"Alias {self.alias_name} should appear in Line Plot toolbar"

        self.log("Removing alias for channel")
        console.channels.clear_alias(self.alias_name)

        self.log("Verifying alias removal in Resources Toolbar")
        console.channels.show_channels()
        alias_item_after = console.channels._find_channel_item(
            self.alias_name, retry_with_refresh=False
        )
        assert (
            alias_item_after is None
        ), f"Alias {self.alias_name} should no longer appear in Resources Toolbar"
        original_item = console.channels._find_channel_item(self.data_name)
        assert (
            original_item is not None
        ), f"Original channel {self.data_name} should appear in Resources Toolbar"
        console.channels.hide_channels()

        self.log("Verifying alias removal in Line Plot Toolbar")
        assert not plot.has_channel(
            "Y1", self.alias_name
        ), f"Alias {self.alias_name} should no longer appear in Line Plot toolbar"
        assert plot.has_channel(
            "Y1", self.data_name
        ), f"Original channel {self.data_name} should appear in Line Plot toolbar"

        plot.close()
