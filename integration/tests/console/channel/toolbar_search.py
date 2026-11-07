#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from console.case import ConsoleCase
from framework.utils import create_indexed_channels, create_time_index
from x import random_name


class ToolbarSearch(ConsoleCase):
    """Test the search in the channels pane: layouts, rename, and close."""

    token: str
    group_name: str
    group_key: sy.group.Key
    names: list[str]
    other: str
    index: sy.Channel

    def setup(self) -> None:
        super().setup()
        self.token = random_name().lower()
        self.group_name = f"Search group {random_name()}"
        self.names = [f"{self.token}_flow_{i}" for i in range(3)]
        self.other = f"plain_{random_name().lower()}"
        self.index = create_time_index(self.client, f"search_time_{random_name()}")
        created = create_indexed_channels(
            self.client, [*self.names, self.other], index_key=self.index.key
        )
        channels_group = sy.group.ontology_id(self.client.channels.retrieve_group().key)
        grp = self.client.groups.create(channels_group, self.group_name)
        self.group_key = grp.key
        self.client.ontology.move_children(
            channels_group,
            sy.group.ontology_id(grp.key),
            *[ch.ontology_id for ch in created],
        )

    def run(self) -> None:
        self.test_flat_results()
        self.test_groups_layout()
        self.test_rename_result()
        self.test_close_restores_tree()
        self.console.channels.hide_channels()

    def test_flat_results(self) -> None:
        """Each match shows with the name of its group. Other channels do not."""
        self.log("Testing flat results")
        channels = self.console.channels
        channels.search(self.token)
        for name in self.names:
            item = channels.tree.wait_for_name("channel:", name)
            label = item.locator(".console-tree-item__group").inner_text()
            assert label == self.group_name, (
                f"Result '{name}' should show its group '{self.group_name}', "
                f"found '{label}'"
            )
        assert channels.tree.find_by_name("channel:", self.other) is None, (
            f"Channel '{self.other}' does not match '{self.token}'"
        )

    def test_groups_layout(self) -> None:
        """The groups layout shows each match inside its group."""
        self.log("Testing groups layout")
        channels = self.console.channels
        channels.toggle_search_groups()
        group = channels.tree.wait_for_name("group:", self.group_name)
        item = channels.tree.wait_for_name("channel:", self.names[0])
        group_level = int(group.get_attribute("aria-level") or 0)
        item_level = int(item.get_attribute("aria-level") or 0)
        assert item_level == group_level + 1, (
            f"Result '{self.names[0]}' should sit one level under its group"
        )
        assert item.locator(".console-tree-item__group").count() == 0, (
            "The groups layout should not repeat the group name on a result"
        )
        channels.toggle_search_groups()

    def test_rename_result(self) -> None:
        """A result renames the channel, as in the tree."""
        self.log("Testing rename from a result")
        channels = self.console.channels
        old = self.names[1]
        new = f"{self.token}_renamed"
        channels.tree.rename(channels.tree.wait_for_name("channel:", old), new)
        channels.tree.wait_for_name("channel:", new)
        self.names[1] = new
        assert self.client.channels.retrieve(new).name == new

    def test_close_restores_tree(self) -> None:
        """Escape closes the search and brings the tree back."""
        self.log("Testing close restores the tree")
        channels = self.console.channels
        channels.close_search()
        group = channels.tree.wait_for_name("group:", self.group_name)
        assert group.is_visible(), f"Group '{self.group_name}' should show again"
        assert channels.tree.find_by_name("channel:", self.names[0]) is None, (
            "The tree should show the group collapsed, not the search results"
        )

    def teardown(self) -> None:
        self.client.channels.delete([*self.names, self.other, self.index.name])
        self.client.groups.delete([self.group_key])
        super().teardown()
