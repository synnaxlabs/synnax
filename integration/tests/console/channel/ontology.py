#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from framework.utils import get_random_name

METRICS_GROUP = "Metrics"
METRICS_PREFIX = "sy_node_1_metrics"


class ChannelOntology(ConsoleCase):
    """Test channel ontology operations: groups, nesting, and drag-to-move."""

    suffix: str
    channels: list[str]
    group_a: str
    group_b: str

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self.group_a = f"Group A {self.suffix}"
        self.group_b = f"Group B {self.suffix}"
        self._cleanup_groups()
        self._discover_channels()

    def _discover_channels(self) -> None:
        """Discover metrics channel names by expanding the Metrics group."""
        self.console.channels.expand_group(METRICS_GROUP)
        names = self.console.channels.tree.list_names("channel:")
        self.console.channels.hide_channels()
        self.channels = [n for n in names if n.startswith(METRICS_PREFIX)]
        self.log(f"Discovered {len(self.channels)} metrics channels: {self.channels}")
        assert (
            len(self.channels) >= 4
        ), f"Expected at least 4 metrics channels, found {len(self.channels)}"

    def run(self) -> None:
        """Run all channel ontology tests."""
        self.test_create_group()
        self.test_move_to_group()
        self.test_rename_group()
        self.test_create_nested_group()
        self.test_delete_groups()

    def test_create_group(self) -> None:
        """Test creating a group from multiple channels via multi-select."""
        self.log("Testing create group")
        self.console.channels.group(
            names=[self.channels[0], self.channels[1]],
            group_name=self.group_a,
            parent_group=METRICS_GROUP,
        )
        assert self.console.channels.group_exists(
            self.group_a, parent_group=METRICS_GROUP
        ), f"Group '{self.group_a}' should exist after creation"

    def test_move_to_group(self) -> None:
        """Test moving a channel into a group via drag-and-drop."""
        self.log("Testing move channel to group")
        self.console.channels.move_to_group(
            self.channels[2], self.group_a, parent_group=METRICS_GROUP
        )
        # Verify channel is inside the group.
        self.console.channels.expand_group(METRICS_GROUP)
        group = self.console.channels.tree.get_group(self.group_a)
        if not self.console.channels.tree.is_expanded(group):
            self.console.channels.tree.expand(group)
        ch = self.console.channels.tree.find_by_name("channel:", self.channels[2])
        assert (
            ch is not None and ch.is_visible()
        ), f"Channel '{self.channels[2]}' should be inside '{self.group_a}'"
        self.console.channels.hide_channels()

    def test_rename_group(self) -> None:
        """Test renaming a group via context menu."""
        self.log("Testing rename group")
        new_name = f"Renamed Group {self.suffix}"
        self.console.channels.rename_group(
            self.group_a, new_name, parent_group=METRICS_GROUP
        )
        self.group_a = new_name
        assert self.console.channels.group_exists(
            self.group_a, parent_group=METRICS_GROUP
        ), f"Renamed group '{self.group_a}' should exist"

    def test_create_nested_group(self) -> None:
        """Test creating a nested group inside an existing group."""
        self.log("Testing create nested group")
        self.console.channels.expand_group(METRICS_GROUP)
        group_item = self.console.channels.tree.get_group(self.group_a)
        if not self.console.channels.tree.is_expanded(group_item):
            self.console.channels.tree.expand(group_item)

        found = [
            self.console.channels.tree.find_by_name("channel:", n)
            for n in [self.channels[0], self.channels[1]]
        ]
        items = [item for item in found if item is not None]
        assert len(items) == 2, "Expected to find both channels for nested group"
        self.console.channels.tree.group(items, self.group_b)
        self.console.channels.hide_channels()

        # Verify nested group is visible.
        self.console.channels.expand_group(METRICS_GROUP)
        group_item = self.console.channels.tree.get_group(self.group_a)
        if not self.console.channels.tree.is_expanded(group_item):
            self.console.channels.tree.expand(group_item)
        nested = self.console.channels.tree.get_group(self.group_b)
        assert (
            nested.is_visible()
        ), f"Nested group '{self.group_b}' should be visible inside '{self.group_a}'"
        self.console.channels.hide_channels()

    def test_delete_groups(self) -> None:
        """Test deleting groups via context menu."""
        self.log("Testing delete groups")

        # Expand to access nested group_b.
        self.console.channels.expand_group(METRICS_GROUP)
        parent = self.console.channels.tree.get_group(self.group_a)
        if not self.console.channels.tree.is_expanded(parent):
            self.console.channels.tree.expand(parent)

        # Delete nested group_b first (expand so Ungroup appears).
        nested = self.console.channels.tree.get_group(self.group_b)
        if not self.console.channels.tree.is_expanded(nested):
            self.console.channels.tree.expand(nested)
        self.console.channels.tree.delete_group(nested)
        self.console.channels.hide_channels()

        # Delete parent via high-level client method (auto-expands).
        self.console.channels.delete_group(self.group_a, parent_group=METRICS_GROUP)

    def _cleanup_groups(self) -> None:
        """Remove leftover test groups from the channel tree."""
        self.console.channels.show_channels()
        tree = self.console.channels.tree
        for _ in range(10):
            groups = tree.find_by_prefix("group:")
            leftover = [
                g
                for g in groups
                if g.is_visible() and tree.get_text(g) != METRICS_GROUP
            ]
            if not leftover:
                break
            item = leftover[0]
            if not tree.is_expanded(item):
                tree.expand(item)
            tree.delete_group(item)
        self.console.channels.hide_channels()

    def teardown(self) -> None:
        self._cleanup_groups()
        super().teardown()
