#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import uuid

import synnax as sy

from console.case import ConsoleCase


class ChannelOperations(ConsoleCase):
    """Test channel lifecycle operations."""

    def run(self) -> None:
        """Run all channel operation tests."""

        # Create Channels Modal
        self.test_create_multiple_channels()

        # Resources Toolbar
        self.test_open_channel_plot()

        ## Context Menu
        self.test_rename_channel()
        self.test_group_channels()
        self.test_edit_calculated_channel()
        self.test_set_alias_under_range()
        self.test_clear_alias_under_range()
        self.test_delete_channel()
        self.test_copy_link()

        # Search and Command Palette
        self.test_open_channel_plot_by_name()
        self.test_open_create_channel_modal()
        self.test_open_create_calculated_channel_modal()

        # Calculated Channels
        # test_plot_a_basic_calculated_channel()
        # plot_a_nested_calculated_channel()
        # test_intentionally_create_a_channel_with_an_erroneous_expression()
        # test_run_and_plot_channels_from_python_calc_channel_stress_py() # with --rate at 10, 100, 1000 Hz

        # Miscellaneous
        # test_open_a_channel_plot_from_a_link()
        # test_rename_a_channel_and_ensure_synchronization_across_ui_elements()
        # test_set_an_alias_for_a_channel_and_ensure_synchronization_across_ui_elements()
        # test_remove_an_alias_for_a_channel_and_ensure_synchronization_across_ui_elements()
        self.test_hard_reload_console()

    def test_create_multiple_channels(self) -> None:
        """Test creating multiple channels using the 'Create More' checkbox."""
        self.log("Testing create multiple channels with 'Create More'")

        console = self.console
        client = self.client

        # Use unique prefix to avoid conflicts with previous runs
        prefix = str(uuid.uuid4())[:6]

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

        index_name = f"multi_idx_{prefix}"
        channels: list[dict[str, str | int | bool]] = [
            {"name": index_name, "is_index": True}
        ]
        for dt in data_types:
            channels.append(
                {
                    "name": f"{str(dt)}_ch_{prefix}",
                    "data_type": dt,
                    "index": index_name,
                }
            )

        # Create all channels using create_with_create_more
        created = console.channels.create_with_create_more(channels)

        expected_count = 1 + len(data_types)
        assert (
            len(created) == expected_count
        ), f"Expected {expected_count} channels created, got {len(created)}"
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
        line_plot = self.page.locator(".pluto-line-plot")
        line_plot.first.wait_for(state="visible", timeout=5000)
        assert (
            line_plot.first.is_visible()
        ), "Expected a line plot to be visible after double-clicking channel"

        # Cleanup
        console.channels.delete([data_name, index_name])

    def test_rename_channel(self) -> None:
        """Test renaming a channel via context menu."""
        self.log("Testing rename channel")

        console = self.console

        prefix = str(uuid.uuid4())[:6]
        index_name = f"rename_idx_{prefix}"
        data_name = f"rename_data_{prefix}"
        new_name = f"renamed_data_{prefix}"

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        console.channels.rename(data_name, new_name)

        ch = self.client.channels.retrieve(new_name)
        assert ch.name == new_name, f"Expected channel name {new_name}, got {ch.name}"

        console.channels.delete([new_name, index_name])

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

        group_element = self.page.get_by_text(group_name)
        group_element.first.wait_for(state="visible", timeout=5000)
        assert group_element.count() > 0, f"Expected group '{group_name}' to be visible"

        # Cleanup: Expand group, delete data channels first, then index
        console.channels.show_channels()

        # Click on the group to expand it
        group_expander = self.page.locator(f"text={group_name}").first
        group_expander.click()

        # Wait for first channel to become visible (group expanded)
        self.page.locator(f"text={ch1_name}").first.wait_for(
            state="visible", timeout=2000
        )

        # Delete nested channels by finding them directly and right-clicking
        for ch_name in [ch2_name, ch1_name]:
            ch_element = self.page.locator(f"text={ch_name}").first
            if ch_element.count() > 0 and ch_element.is_visible():
                ch_element.click(button="right")

                delete_option = self.page.get_by_text("Delete", exact=True).first
                delete_option.wait_for(state="visible", timeout=2000)
                delete_option.click()

                # Confirm delete if modal appears
                delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
                if delete_btn.count() > 0:
                    delete_btn.first.click()
                    # Wait for modal to close
                    delete_btn.first.wait_for(state="hidden", timeout=5000)

        # Then delete the index
        console.channels.delete([index_name])

    def test_edit_calculated_channel(self) -> None:
        """Test editing a calculated channel's calculation via context menu."""
        self.log("Testing edit calculated channel")

        console = self.console
        client = self.client

        initial_multiplier = 2
        updated_multiplier = 30
        prefix = str(uuid.uuid4())[:6]
        calc_name = f"calc_ch_{prefix}"
        initial_expr = f"return channel_operations_uptime * {initial_multiplier}"
        updated_expr = f"return channel_operations_uptime * {updated_multiplier}"

        console.channels.create_calculated(
            name=calc_name,
            expression=initial_expr,
        )

        calc_ch = client.channels.retrieve(calc_name)
        assert calc_ch.virtual, "Calculated channel should be virtual"
        assert (
            initial_expr in calc_ch.expression
        ), f"Expected expression to contain '{initial_expr}', got '{calc_ch.expression}'"

        uptime_val = int(client.read_latest("channel_operations_uptime", n=1)[-1])
        calc_val = int(client.read_latest(calc_name, n=1)[-1])
        expected_val = uptime_val * initial_multiplier
        assert expected_val == calc_val, f"expected {expected_val}, got {calc_val}"

        console.channels.edit_calculated(calc_name, updated_expr)
        uptime_val = int(client.read_latest("channel_operations_uptime", n=1)[-1])
        calc_val = int(client.read_latest(calc_name, n=1)[-1])
        expected_val = uptime_val * updated_multiplier
        assert expected_val == calc_val, f"expected {expected_val}, got {calc_val}"

        console.channels.delete([calc_name])

    def test_set_alias_under_range(self) -> None:
        """Test setting an alias for a channel under a range via context menu."""
        self.log("Testing set alias under range")

        console = self.console
        client = self.client

        prefix = str(uuid.uuid4())[:6]
        range_name = f"alias_range_{prefix}"
        index_name = f"alias_idx_{prefix}"
        data_name = f"alias_data_{prefix}"
        alias_name = f"MyAlias_{prefix}"

        console.ranges.create(range_name, persisted=True)
        console.ranges.open_explorer()
        console.ranges.favorite_from_explorer(range_name)
        console.ranges.show_toolbar()
        console.ranges.set_active(range_name)

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        console.channels.set_alias(data_name, alias_name)

        console.channels.show_channels()
        alias_visible = self.page.get_by_text(alias_name).count() > 0
        assert alias_visible, f"Alias '{alias_name}' should be visible in channel list"
        console.channels.hide_channels()

        rng = client.ranges.retrieve(name=range_name)
        data_ch = client.channels.retrieve(data_name)
        scoped_ch = rng[alias_name]
        assert (
            scoped_ch.key == data_ch.key
        ), f"Alias should resolve to channel key {data_ch.key}, got {scoped_ch.key}"

        # Channel now displays as alias_name in the UI
        console.channels.delete([alias_name, index_name])
        console.ranges.open_explorer()
        console.ranges.delete_from_explorer(range_name)

    def test_clear_alias_under_range(self) -> None:
        """Test clearing an alias for a channel via context menu."""
        self.log("Testing clear alias under range")

        console = self.console
        client = self.client

        prefix = str(uuid.uuid4())[:6]
        range_name = f"clear_alias_range_{prefix}"
        index_name = f"clear_alias_idx_{prefix}"
        data_name = f"clear_alias_data_{prefix}"
        alias_name = f"ClearAlias_{prefix}"

        console.ranges.create(range_name, persisted=True)
        console.ranges.open_explorer()
        console.ranges.favorite_from_explorer(range_name)
        console.ranges.show_toolbar()
        console.ranges.set_active(range_name)

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        console.channels.set_alias(data_name, alias_name)

        console.channels.show_channels()
        alias_visible = self.page.get_by_text(alias_name).count() > 0
        assert alias_visible, f"Alias '{alias_name}' should be visible before clearing"
        console.channels.hide_channels()

        rng = client.ranges.retrieve(name=range_name)
        data_ch = client.channels.retrieve(data_name)
        scoped_ch = rng[alias_name]
        assert scoped_ch.key == data_ch.key, "Alias should resolve before clearing"

        # Channel displays with alias_name in UI after set_alias
        console.channels.clear_alias(alias_name)

        console.channels.show_channels()
        alias_still_visible = self.page.get_by_text(alias_name).count() > 0
        assert (
            not alias_still_visible
        ), f"Alias '{alias_name}' should not be visible after clearing"
        console.channels.hide_channels()

        rng = client.ranges.retrieve(name=range_name)
        try:
            rng[alias_name]
            assert False, f"Alias '{alias_name}' should not resolve after clearing"
        except (KeyError, sy.QueryError):
            pass

        console.channels.delete([data_name, index_name])
        console.ranges.open_explorer()
        console.ranges.delete_from_explorer(range_name)

    def test_delete_channel(self) -> None:
        """Test deleting a channel via context menu."""
        self.log("Testing delete channel")

        console = self.console

        prefix = str(uuid.uuid4())[:6]
        index_name = f"delete_idx_{prefix}"
        data_name = f"delete_data_{prefix}"

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        console.channels.delete([data_name])
        exists, _ = console.channels.existing_channel(data_name)
        assert not exists, f"Channel {data_name} should not appear in UI"

        console.channels.delete([index_name])
        exists, _ = console.channels.existing_channel(index_name)
        assert not exists, f"Index channel {index_name} should not appear in UI"

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

        # Verify a link was copied (or at least no error occurred)
        # Clipboard access may not be available in all browsers
        link = console.channels.copy_link(data_name)
        if link:
            assert (
                "channel" in link.lower() or data_name in link
            ), f"Expected link to contain 'channel' or channel name, got: {link}"
            self.log(f"Copied link: {link}")
        else:
            self.log("Copy link executed (clipboard not accessible for verification)")

        # Cleanup
        console.channels.delete([data_name, index_name])

    def test_hard_reload_console(self) -> None:
        """Test hard reload via channel context menu."""
        self.log("Testing hard reload console")
        console = self.console
        console.channels.hard_reload()

        # Wait for the channels button to be visible, indicating the console has reloaded
        console.channels.channels_button.wait_for(state="visible", timeout=10000)

    def test_open_channel_plot_by_name(self) -> None:
        """Test opening a channel plot by searching its name in the command palette."""
        self.log("Testing open channel plot by name via command palette")

        console = self.console
        prefix = str(uuid.uuid4())[:6]
        index_name = f"search_idx_{prefix}"
        data_name = f"search_data_{prefix}"

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        console.channels.open_plot_by_search(data_name)

        console.channels.delete([data_name, index_name])

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
