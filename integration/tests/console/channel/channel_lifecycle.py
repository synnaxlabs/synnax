#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import synnax as sy
from console.case import ConsoleCase
from console.plot import Plot
from framework.utils import assert_link_format
from x import random_name


class ChannelLifecycle(ConsoleCase):
    """Test channel lifecycle operations."""

    suffix: str
    shared_index: str
    shared_data: str

    def setup(self) -> None:
        super().setup()
        self.suffix = random_name()
        self._create_shared_channels()

    def _create_shared_channels(self) -> None:
        """Create shared index + data channel for read-only tests."""
        self.shared_index = f"shared_idx_{self.suffix}"
        self.shared_data = f"shared_data_{self.suffix}"

        self.client.channels.create(
            name=self.shared_index,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        self.console.channels.wait_for_channels(self.shared_index, timeout=5.0)
        self.console.channels.create(
            name=self.shared_data,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )

    def teardown(self) -> None:
        with self._try_to("delete channels"):
            self.console.channels.delete([self.shared_data, self.shared_index])
        super().teardown()

    def run(self) -> None:
        """Run all channel operation tests."""

        # Create Channels Modal
        self.test_create_multiple_channels()

        # Resources Toolbar
        self.test_open_channel_plot()

        ## Context Menu
        self.test_rename_channel()
        self.test_alias_operations()
        self.test_delete_channel()
        self.test_copy_link()

        # Search and Command Palette
        self.test_open_channel_plot_by_name()
        self.test_open_create_channel_modal()
        self.test_open_create_calculated_channel_modal()

    def test_create_multiple_channels(self) -> None:
        """Test creating multiple channels using the 'Create more' checkbox."""
        self.log("Testing create multiple channels with 'Create more'")

        console = self.console
        client = self.client

        suffix = random_name()

        all_data_types = [
            sy.DataType.FLOAT64,
            sy.DataType.FLOAT32,
            sy.DataType.INT64,
            sy.DataType.INT32,
            sy.DataType.INT16,
            sy.DataType.INT8,
            sy.DataType.UINT64,
            sy.DataType.UINT32,
            sy.DataType.UINT16,
            sy.DataType.UINT8,
            sy.DataType.UUID,
        ]

        sample_size = max(1, len(all_data_types) // 4)
        data_types = random.sample(all_data_types, sample_size)

        index_name = f"multi_idx_{suffix}"
        channels: list[dict[str, str | int | bool]] = [
            {"name": index_name, "is_index": True}
        ]
        for dt in data_types:
            channels.append(
                {
                    "name": f"{str(dt)}_ch_{suffix}",
                    "data_type": dt,
                    "index": index_name,
                }
            )

        created = console.channels.create_with_create_more(channels)

        expected_count = 1 + len(data_types)
        assert len(created) == expected_count, (
            f"Expected {expected_count} channels created, got {len(created)}"
        )
        self.log(f"Created channels: {created}")

        for ch_config in channels:
            ch_name = str(ch_config["name"])
            assert console.channels.exists(ch_name), f"Channel {ch_name} should exist"

            ch = client.channels.retrieve(ch_name)
            if ch_config.get("is_index"):
                assert ch.data_type == sy.DataType.TIMESTAMP, (
                    f"Index channel should be TIMESTAMP, got {ch.data_type}"
                )
            else:
                expected_type = ch_config["data_type"]
                assert ch.data_type == expected_type, (
                    f"Channel {ch_name} should be {expected_type}, got {ch.data_type}"
                )

        channels_to_delete = [str(ch["name"]) for ch in reversed(channels)]
        console.channels.delete(channels_to_delete)

    def test_open_channel_plot(self) -> None:
        """Test opening a channel plot by double-clicking."""
        self.log("Testing open channel plot by double-click")

        plot = self.console.pages.open_plot_from_click(
            self.shared_data, self.console.channels
        )
        self._cleanup_pages.append(plot.page_name)

        line_plot = self.page.locator(".pluto-line-plot")
        line_plot.first.wait_for(state="visible", timeout=5000)
        plot.close()

    def test_rename_channel(self) -> None:
        """Test renaming a channel via context menu."""
        self.log("Testing rename channel")

        console = self.console

        suffix = random_name()
        data_name = f"rename_data_{suffix}"
        new_name = f"renamed_data_{suffix}"

        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )

        console.channels.rename(names=data_name, new_names=new_name)

        ch = self.client.channels.retrieve(new_name)
        assert ch.name == new_name, f"Expected channel name {new_name}, got {ch.name}"

        console.channels.delete([new_name])

    def test_alias_operations(self) -> None:
        """Test setting, verifying, clearing, and re-verifying a channel alias."""
        console = self.console
        client = self.client

        suffix = random_name()
        range_name = f"alias_range_{suffix}"
        data_name = f"alias_data_{suffix}"
        alias_name = f"MyAlias_{suffix}"

        console.ranges.create(range_name, persisted=True)
        console.ranges.explorer.open()
        console.ranges.toolbar.show()
        console.ranges.set_active(range_name)

        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )

        console.channels.set_alias(name=data_name, alias=alias_name)

        console.channels.show_channels()
        alias_visible = self.page.get_by_text(alias_name).count() > 0
        assert alias_visible, f"Alias '{alias_name}' should be visible in channel list"
        console.channels.hide_channels()

        rng = client.ranges.retrieve(name=range_name)
        data_ch = client.channels.retrieve(data_name)
        scoped_ch = rng[alias_name]
        assert scoped_ch.key == data_ch.key, (
            f"Alias should resolve to channel key {data_ch.key}, got {scoped_ch.key}"
        )

        console.channels.clear_alias(alias_name)

        console.channels.show_channels()
        alias_still_visible = self.page.get_by_text(alias_name).count() > 0
        assert not alias_still_visible, (
            f"Alias '{alias_name}' should not be visible after clearing"
        )
        console.channels.hide_channels()

        rng = client.ranges.retrieve(name=range_name)
        try:
            rng[alias_name]
            assert False, f"Alias '{alias_name}' should not resolve after clearing"
        except sy.QueryError:
            pass

        console.channels.delete([data_name])
        console.ranges.explorer.open()
        console.ranges.explorer.delete(range_name)

    def test_delete_channel(self) -> None:
        """Test deleting a channel via context menu."""
        self.log("Testing delete channel")

        console = self.console

        suffix = random_name()
        index_name = f"delete_idx_{suffix}"
        data_name = f"delete_data_{suffix}"

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        console.channels.delete([data_name])
        console.channels.delete([index_name])

    def test_copy_link(self) -> None:
        """Test copying a channel link via context menu."""
        self.log("Testing copy channel link")

        link = self.console.channels.copy_link(self.shared_data)

        channel = self.client.channels.retrieve(self.shared_data)
        assert_link_format(link, "channel", str(channel.key))

    def test_open_channel_plot_by_name(self) -> None:
        """Test opening a channel plot by searching its name in the command palette."""
        self.log("Testing open channel plot by name via command palette")

        plot = self.console.pages.open_from_search(Plot, self.shared_data)
        self._cleanup_pages.append(plot.page_name)
        plot.close()

    def test_open_create_channel_modal(self) -> None:
        """Test opening the Create Channel modal via command palette."""
        self.log("Testing open Create Channel modal via command palette")

        console = self.console
        console.channels.open_create_modal()
        console.channels.close_modal()

    def test_open_create_calculated_channel_modal(self) -> None:
        """Test opening the Create Calculated Channel modal via command palette."""
        self.log("Testing open Create Calculated Channel modal via command palette")

        console = self.console
        console.channels.open_create_calculated_modal()
        console.channels.close_modal()
