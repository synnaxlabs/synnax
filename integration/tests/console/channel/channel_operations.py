#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

import synnax as sy

from console.case import ConsoleCase


class ChannelOperations(ConsoleCase):
    """Test channel operations beyond basic CRUD."""

    def run(self) -> None:
        """Run all channel operation tests."""
        self.test_create_multiple_channels()
        self.test_open_channel_plot()
        self.test_group_channels()
        self.test_copy_link()

    def test_create_multiple_channels(self) -> None:
        """Test creating multiple channels using the 'Create More' checkbox."""
        self.log("Testing create multiple channels with 'Create More'")

        console = self.console
        client = self.client

        # Use unique prefix to avoid conflicts with previous runs
        prefix = str(uuid.uuid4())[:6]

        # Define channels to create
        index_name = f"multi_idx_{prefix}"
        channels = [
            {"name": index_name, "is_index": True},
            {
                "name": f"multi_f32_{prefix}",
                "data_type": sy.DataType.FLOAT32,
                "index": index_name,
            },
            {
                "name": f"multi_i64_{prefix}",
                "data_type": sy.DataType.INT64,
                "index": index_name,
            },
        ]

        # Create all channels using create_with_create_more
        created = console.channels.create_with_create_more(channels)

        # Verify all channels were created
        assert len(created) == 3, f"Expected 3 channels created, got {len(created)}"
        self.log(f"Created channels: {created}")

        # Verify channels exist in the server
        for ch_config in channels:
            ch_name = ch_config["name"]
            exists, _ = console.channels.existing_channel(ch_name)
            assert exists, f"Channel {ch_name} should exist"

            # Verify data type via client
            ch = client.channels.retrieve(ch_name)
            if ch_config.get("is_index"):
                assert (
                    ch.data_type == sy.DataType.TIMESTAMP
                ), f"Index channel should be TIMESTAMP, got {ch.data_type}"
            else:
                expected_type = ch_config["data_type"]
                assert (
                    ch.data_type == expected_type
                ), f"Channel {ch_name} should be {expected_type}, got {ch.data_type}"

        self.log("Create multiple channels test passed")

        # Cleanup - delete channels in reverse order (data channels first, then index)
        channels_to_delete = [ch["name"] for ch in reversed(channels)]
        console.channels.delete(channels_to_delete)

    def test_open_channel_plot(self) -> None:
        """Test opening a channel plot by double-clicking."""
        self.log("Testing open channel plot by double-click")

        console = self.console

        # Use unique prefix to avoid conflicts
        prefix = str(uuid.uuid4())[:6]

        # Create test channels
        index_name = f"plot_idx_{prefix}"
        data_name = f"plot_data_{prefix}"

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        # Double-click the data channel to open its plot
        console.channels.open_plot(data_name)

        # Verify a line plot was opened by checking for the line plot element
        self.page.wait_for_timeout(1000)  # Wait for plot to fully render

        line_plot = self.page.locator(".pluto-line-plot")
        plot_visible = line_plot.count() > 0 and line_plot.first.is_visible()
        assert (
            plot_visible
        ), "Expected a line plot to be visible after double-clicking channel"

        self.log("Open channel plot test passed")

        # Cleanup
        console.channels.delete([data_name, index_name])

    def test_group_channels(self) -> None:
        """Test grouping multiple channels via context menu."""
        self.log("Testing group channels")

        console = self.console

        # Use unique prefix to avoid conflicts
        prefix = str(uuid.uuid4())[:6]

        # Create test channels
        index_name = f"group_idx_{prefix}"
        ch1_name = f"group_ch1_{prefix}"
        ch2_name = f"group_ch2_{prefix}"
        group_name = f"TestGroup_{prefix}"

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=ch1_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )
        console.channels.create(
            name=ch2_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        # Group the two data channels
        console.channels.group([ch1_name, ch2_name], group_name)

        # Verify the group exists by looking for it in the channel list
        console.channels.show_channels()
        self.page.wait_for_timeout(500)

        group_element = self.page.get_by_text(group_name)
        group_visible = group_element.count() > 0
        assert group_visible, f"Expected group '{group_name}' to be visible"

        self.log("Group channels test passed")

        # Cleanup: Expand group, delete data channels first, then index
        console.channels.show_channels()

        # Click on the group to expand it
        group_expander = self.page.locator(f"text={group_name}").first
        group_expander.click()
        self.page.wait_for_timeout(500)

        # Delete nested channels by finding them directly and right-clicking
        for ch_name in [ch2_name, ch1_name]:
            ch_element = self.page.locator(f"text={ch_name}").first
            if ch_element.count() > 0 and ch_element.is_visible():
                ch_element.click(button="right")
                self.page.wait_for_timeout(100)
                self.page.get_by_text("Delete", exact=True).first.click()
                self.page.wait_for_timeout(100)
                # Confirm delete if modal appears
                delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
                if delete_btn.count() > 0:
                    delete_btn.first.click()
                self.page.wait_for_timeout(300)

        # Then delete the index
        console.channels.delete([index_name])

    def test_copy_link(self) -> None:
        """Test copying a channel link via context menu."""
        self.log("Testing copy channel link")

        console = self.console

        # Use unique prefix to avoid conflicts
        prefix = str(uuid.uuid4())[:6]

        # Create test channel
        index_name = f"link_idx_{prefix}"
        data_name = f"link_data_{prefix}"

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        # Copy the link
        link = console.channels.copy_link(data_name)

        # Verify a link was copied (or at least no error occurred)
        # Clipboard access may not be available in all browsers
        if link:
            assert (
                "channel" in link.lower() or data_name in link
            ), f"Expected link to contain 'channel' or channel name, got: {link}"
            self.log(f"Copied link: {link}")
        else:
            # If clipboard isn't accessible, just verify no error occurred
            self.log("Copy link executed (clipboard not accessible for verification)")

        self.log("Copy link test passed")

        # Cleanup
        console.channels.delete([data_name, index_name])
