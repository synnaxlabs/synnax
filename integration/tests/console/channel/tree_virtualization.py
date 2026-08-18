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

CHANNEL_COUNT = 300
# A windowed tree mounts the rows near the viewport plus a small overscan. Anything
# near CHANNEL_COUNT means every row mounted.
MOUNTED_LIMIT = 60


class TreeVirtualization(ConsoleCase):
    """Test that the channel tree windows its rows and stays navigable."""

    names: list[str]
    index: sy.Channel

    def setup(self) -> None:
        super().setup()
        suffix = random_name()
        self.index = create_time_index(self.client, f"tree_virt_time_{suffix}")
        # Zero-padded so the console sort puts them in creation order, contiguously.
        self.names = [f"tree_virt_{suffix}_{i:03d}" for i in range(CHANNEL_COUNT)]
        create_indexed_channels(self.client, self.names, self.index.key)
        self.log(f"Created {CHANNEL_COUNT} channels under prefix tree_virt_{suffix}")

    def run(self) -> None:
        self.console.channels.show_channels()
        self.test_scroll_reaches_the_last_channel()
        self.test_distant_rows_unmount()
        self.test_scrolling_back_remounts()
        self.console.channels.hide_channels()

    def test_scroll_reaches_the_last_channel(self) -> None:
        """The last channel mounts once the tree scrolls to it."""
        self.log("Testing scroll reaches the last channel")
        last = self.names[-1]
        assert self.console.channels.tree.find_by_name("channel:", last) is not None, (
            f"Channel '{last}' should be reachable by scrolling the tree"
        )

    def test_distant_rows_unmount(self) -> None:
        """Rows far above the viewport leave the DOM."""
        self.log("Testing distant rows unmount")
        mounted = self.console.channels.tree.list_names("channel:")
        assert len(mounted) < MOUNTED_LIMIT, (
            f"Expected fewer than {MOUNTED_LIMIT} mounted channel rows out of "
            f"{CHANNEL_COUNT}, found {len(mounted)}"
        )
        first = self.names[0]
        assert first not in mounted, (
            f"Channel '{first}' sits {CHANNEL_COUNT} rows above the viewport and "
            "should not be mounted"
        )

    def test_scrolling_back_remounts(self) -> None:
        """A row that left the DOM comes back when the tree scrolls to it."""
        self.log("Testing scrolling back remounts")
        first = self.names[0]
        assert self.console.channels.tree.find_by_name("channel:", first) is not None, (
            f"Channel '{first}' should remount when the tree scrolls back up"
        )

    def teardown(self) -> None:
        self.client.channels.delete(self.names)
        self.client.channels.delete(self.index.name)
        super().teardown()
