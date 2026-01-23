#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import time

import synnax as sy

from console.case import ConsoleCase
from console.plot import Plot
from framework.utils import assert_link_format, get_random_name


def _debug_timing(label: str):
    print(f"[TIMING] {time.strftime('%H:%M:%S')} - {label}")

SRC_CH = "channel_operations_uptime"


class ChannelOperations(ConsoleCase):
    """Test channel lifecycle operations."""

    suffix: str
    calc_x2: str
    calc_x6: str
    calc_editable: str
    shared_index: str
    shared_data: str

    def setup(self) -> None:
        _debug_timing("setup() START")
        setup_start = time.time()
        super().setup()
        _debug_timing(f"setup() super().setup() done in {time.time() - setup_start:.2f}s")
        self.suffix = get_random_name()
        _debug_timing("setup() calling _create_shared_channels()")
        shared_start = time.time()
        self._create_shared_channels()
        _debug_timing(f"setup() _create_shared_channels() done in {time.time() - shared_start:.2f}s")
        _debug_timing("setup() calling _create_shared_calc_channels()")
        calc_start = time.time()
        self._create_shared_calc_channels()
        _debug_timing(f"setup() _create_shared_calc_channels() done in {time.time() - calc_start:.2f}s")
        _debug_timing(f"setup() DONE, TOTAL: {time.time() - setup_start:.2f}s")

    def _create_shared_calc_channels(self) -> None:
        """Create shared calculated channels for reuse across tests."""
        self.calc_x2 = f"calc_x2_{self.suffix}"
        self.calc_x6 = f"calc_x6_{self.suffix}"
        self.calc_editable = f"calc_edit_{self.suffix}"

        _debug_timing(f"_create_shared_calc_channels() creating {self.calc_x2}")
        start = time.time()
        error = self.console.channels.create_calculated(
            name=self.calc_x2,
            expression=f"return {SRC_CH} * 2",
        )
        _debug_timing(f"_create_shared_calc_channels() {self.calc_x2} done in {time.time() - start:.2f}s")
        assert error is None, f"Failed to create {self.calc_x2}: {error}"

        _debug_timing(f"_create_shared_calc_channels() creating {self.calc_x6}")
        start = time.time()
        error = self.console.channels.create_calculated(
            name=self.calc_x6,
            expression=f"return {self.calc_x2} * 3",
        )
        _debug_timing(f"_create_shared_calc_channels() {self.calc_x6} done in {time.time() - start:.2f}s")
        assert error is None, f"Failed to create {self.calc_x6}: {error}"

        _debug_timing(f"_create_shared_calc_channels() creating {self.calc_editable}")
        start = time.time()
        error = self.console.channels.create_calculated(
            name=self.calc_editable,
            expression=f"return {SRC_CH} * 2",
        )
        _debug_timing(f"_create_shared_calc_channels() {self.calc_editable} done in {time.time() - start:.2f}s")
        assert error is None, f"Failed to create {self.calc_editable}: {error}"

    def _create_shared_channels(self) -> None:
        """Create shared index + data channel for read-only tests."""
        self.shared_index = f"shared_idx_{self.suffix}"
        self.shared_data = f"shared_data_{self.suffix}"

        _debug_timing(f"_create_shared_channels() creating index via client")
        start = time.time()
        self.client.channels.create(
            name=self.shared_index,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        _debug_timing(f"_create_shared_channels() client create done in {time.time() - start:.2f}s")
        _debug_timing(f"_create_shared_channels() waiting for index channel to appear")
        wait_start = time.time()
        self.console.channels.wait_for_channels(self.shared_index, timeout=5.0)
        _debug_timing(f"_create_shared_channels() wait done in {time.time() - wait_start:.2f}s")
        _debug_timing(f"_create_shared_channels() creating data via console")
        start = time.time()
        self.console.channels.create(
            name=self.shared_data,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )
        _debug_timing(f"_create_shared_channels() console create done in {time.time() - start:.2f}s")

    def teardown(self) -> None:
        _debug_timing("teardown() START")
        teardown_start = time.time()
        _debug_timing("teardown() deleting shared channels")
        self.console.channels.delete(
            [
                self.calc_x6,
                self.calc_x2,
                self.calc_editable,
                self.shared_data,
                self.shared_index,
            ]
        )
        _debug_timing(f"teardown() delete done in {time.time() - teardown_start:.2f}s")
        super().teardown()
        _debug_timing(f"teardown() DONE, TOTAL: {time.time() - teardown_start:.2f}s")

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
        self.test_plot_calculated_channel()
        self.test_erroneous_calculated_channel()

    def test_create_multiple_channels(self) -> None:
        """Test creating multiple channels using the 'Create More' checkbox."""
        self.log("Testing create multiple channels with 'Create More'")

        console = self.console
        client = self.client

        suffix = get_random_name()

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

        # Create all channels using create_with_create_more
        created = console.channels.create_with_create_more(channels)

        expected_count = 1 + len(data_types)
        assert (
            len(created) == expected_count
        ), f"Expected {expected_count} channels created, got {len(created)}"
        self.log(f"Created channels: {created}")

        _debug_timing("test_create_multiple_channels() verifying channels exist")
        verify_start = time.time()
        for ch_config in channels:
            ch_name = ch_config["name"]
            _debug_timing(f"test_create_multiple_channels() checking exists for '{ch_name}'")
            assert console.channels.exists(ch_name), f"Channel {ch_name} should exist"

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
        _debug_timing(f"test_create_multiple_channels() verification done in {time.time() - verify_start:.2f}s")

        _debug_timing("test_create_multiple_channels() starting cleanup")
        cleanup_start = time.time()
        channels_to_delete = [ch["name"] for ch in reversed(channels)]
        console.channels.delete(channels_to_delete)
        _debug_timing(f"test_create_multiple_channels() cleanup done in {time.time() - cleanup_start:.2f}s")

    def test_open_channel_plot(self) -> None:
        """Test opening a channel plot by double-clicking."""
        self.log("Testing open channel plot by double-click")

        plot = self.console.channels.open_plot_from_click(self.client, self.shared_data)

        line_plot = self.page.locator(".pluto-line-plot")
        line_plot.first.wait_for(state="visible", timeout=5000)
        assert (
            line_plot.first.is_visible()
        ), "Expected a line plot to be visible after double-clicking channel"
        plot.close()

    def test_rename_channel(self) -> None:
        """Test renaming a channel via context menu."""
        self.log("Testing rename channel")
        _debug_timing("test_rename_channel() START")
        test_start = time.time()

        console = self.console

        suffix = get_random_name()
        data_name = f"rename_data_{suffix}"
        new_name = f"renamed_data_{suffix}"

        _debug_timing("test_rename_channel() creating data channel")
        start = time.time()
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )
        _debug_timing(f"test_rename_channel() data created in {time.time() - start:.2f}s")

        _debug_timing("test_rename_channel() renaming channel")
        start = time.time()
        console.channels.rename(names=data_name, new_names=new_name)
        _debug_timing(f"test_rename_channel() rename done in {time.time() - start:.2f}s")

        ch = self.client.channels.retrieve(new_name)
        assert ch.name == new_name, f"Expected channel name {new_name}, got {ch.name}"

        _debug_timing("test_rename_channel() deleting channels")
        start = time.time()
        console.channels.delete([new_name])
        _debug_timing(f"test_rename_channel() delete done in {time.time() - start:.2f}s, TOTAL: {time.time() - test_start:.2f}s")

    def test_group_channels(self) -> None:
        """Test grouping multiple channels via context menu."""
        self.log("Testing group channels")
        _debug_timing("test_group_channels() START")
        test_start = time.time()

        console = self.console

        suffix = get_random_name()

        ch1_name = f"group_ch1_{suffix}"
        ch2_name = f"group_ch2_{suffix}"
        group_name = f"TestGroup_{suffix}"

        _debug_timing("test_group_channels() creating ch1")
        start = time.time()
        console.channels.create(
            name=ch1_name,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )
        _debug_timing(f"test_group_channels() ch1 created in {time.time() - start:.2f}s")

        _debug_timing("test_group_channels() creating ch2")
        start = time.time()
        console.channels.create(
            name=ch2_name,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )
        _debug_timing(f"test_group_channels() ch2 created in {time.time() - start:.2f}s")

        _debug_timing("test_group_channels() grouping channels")
        start = time.time()
        console.channels.group(names=[ch1_name, ch2_name], group_name=group_name)
        _debug_timing(f"test_group_channels() group done in {time.time() - start:.2f}s")

        # Verify the group exists by looking for it in the channel list
        console.channels.show_channels()

        group_element = self.page.get_by_text(group_name)
        group_element.first.wait_for(state="visible", timeout=5000)
        assert group_element.count() > 0, f"Expected group '{group_name}' to be visible"

        _debug_timing("test_group_channels() starting cleanup")
        cleanup_start = time.time()
        console.channels.show_channels()

        group_expander = self.page.locator(f"text={group_name}").first
        group_expander.click()

        self.page.locator(f"text={ch1_name}").first.wait_for(
            state="visible", timeout=2000
        )

        for ch_name in [ch2_name, ch1_name]:
            _debug_timing(f"test_group_channels() deleting nested channel '{ch_name}'")
            ch_element = self.page.locator(f"text={ch_name}").first
            if ch_element.count() > 0 and ch_element.is_visible():
                ch_element.click(button="right")

                delete_option = self.page.get_by_text("Delete", exact=True).first
                delete_option.wait_for(state="visible", timeout=2000)
                delete_option.click()

                delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
                if delete_btn.count() > 0:
                    delete_btn.first.click()
                    delete_btn.first.wait_for(state="hidden", timeout=5000)

        _debug_timing(f"test_group_channels() cleanup done in {time.time() - cleanup_start:.2f}s, TOTAL: {time.time() - test_start:.2f}s")

    def test_edit_calculated_channel(self) -> None:
        """Test editing a calculated channel's calculation via context menu."""
        self.log("Testing edit calculated channel")

        console = self.console
        client = self.client

        initial_multiplier = 2
        updated_multiplier = 30
        updated_expr = f"return {SRC_CH} * {updated_multiplier}"

        frame = client.read_latest([self.calc_editable, SRC_CH], n=1)
        uptime_val = int(frame[SRC_CH][-1])
        calc_val = int(frame[self.calc_editable][-1])
        expected_val = uptime_val * initial_multiplier
        assert expected_val == calc_val, f"expected {expected_val}, got {calc_val}"

        console.channels.edit_calculated(self.calc_editable, updated_expr)
        sy.sleep(0.2)
        frame = client.read_latest([self.calc_editable, SRC_CH], n=1)
        uptime_val = int(frame[SRC_CH][-1])
        calc_val = int(frame[self.calc_editable][-1])
        expected_val = uptime_val * updated_multiplier
        assert expected_val == calc_val, f"expected {expected_val}, got {calc_val}"

    def test_set_alias_under_range(self) -> None:
        """Test setting an alias for a channel under a range via context menu."""
        self.log("Testing set alias under range")
        _debug_timing("test_set_alias_under_range() START")
        test_start = time.time()

        console = self.console
        client = self.client

        suffix = get_random_name()
        range_name = f"alias_range_{suffix}"
        data_name = f"alias_data_{suffix}"
        alias_name = f"MyAlias_{suffix}"

        _debug_timing("test_set_alias_under_range() creating range")
        start = time.time()
        console.ranges.create(range_name, persisted=True)
        _debug_timing(f"test_set_alias_under_range() range created in {time.time() - start:.2f}s")

        _debug_timing("test_set_alias_under_range() opening explorer and favoriting")
        start = time.time()
        console.ranges.open_explorer()
        console.ranges.favorite_from_explorer(range_name)
        console.ranges.show_toolbar()
        console.ranges.set_active(range_name)
        _debug_timing(f"test_set_alias_under_range() range setup done in {time.time() - start:.2f}s")

        _debug_timing("test_set_alias_under_range() creating data channel")
        start = time.time()
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )
        _debug_timing(f"test_set_alias_under_range() data channel created in {time.time() - start:.2f}s")

        _debug_timing("test_set_alias_under_range() setting alias")
        start = time.time()
        console.channels.set_alias(name=data_name, alias=alias_name)
        _debug_timing(f"test_set_alias_under_range() alias set in {time.time() - start:.2f}s")

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

        _debug_timing("test_set_alias_under_range() starting cleanup")
        cleanup_start = time.time()
        console.channels.delete([alias_name])
        console.ranges.open_explorer()
        console.ranges.delete_from_explorer(range_name)
        _debug_timing(f"test_set_alias_under_range() cleanup done in {time.time() - cleanup_start:.2f}s, TOTAL: {time.time() - test_start:.2f}s")

    def test_clear_alias_under_range(self) -> None:
        """Test clearing an alias for a channel via context menu."""
        self.log("Testing clear alias under range")

        console = self.console
        client = self.client

        suffix = get_random_name()
        range_name = f"clear_alias_range_{suffix}"
        data_name = f"clear_alias_data_{suffix}"
        alias_name = f"ClearAlias_{suffix}"

        console.ranges.create(range_name, persisted=True)
        console.ranges.open_explorer()
        console.ranges.favorite_from_explorer(range_name)
        console.ranges.show_toolbar()
        console.ranges.set_active(range_name)

        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )

        console.channels.set_alias(name=data_name, alias=alias_name)

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
        except sy.QueryError:
            pass
        except Exception as e:
            raise AssertionError(
                f"Expected QueryError when accessing cleared alias, got {type(e).__name__}: {e}"
            )

        console.channels.delete([data_name])
        console.ranges.open_explorer()
        console.ranges.delete_from_explorer(range_name)

    def test_delete_channel(self) -> None:
        """Test deleting a channel via context menu."""
        self.log("Testing delete channel")
        _debug_timing("test_delete_channel() START")
        test_start = time.time()

        console = self.console

        suffix = get_random_name()
        index_name = f"delete_idx_{suffix}"
        data_name = f"delete_data_{suffix}"

        _debug_timing("test_delete_channel() creating channels")
        start = time.time()
        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )
        _debug_timing(f"test_delete_channel() channels created in {time.time() - start:.2f}s")

        _debug_timing("test_delete_channel() deleting data channel")
        start = time.time()
        console.channels.delete([data_name])
        _debug_timing(f"test_delete_channel() data deleted in {time.time() - start:.2f}s")

        _debug_timing("test_delete_channel() verifying data channel deleted")
        start = time.time()
        assert not console.channels.exists(
            data_name
        ), f"Channel {data_name} should not appear in UI"
        _debug_timing(f"test_delete_channel() data exists check in {time.time() - start:.2f}s")

        _debug_timing("test_delete_channel() deleting index channel")
        start = time.time()
        console.channels.delete([index_name])
        _debug_timing(f"test_delete_channel() index deleted in {time.time() - start:.2f}s")

        _debug_timing("test_delete_channel() verifying index channel deleted")
        start = time.time()
        assert not console.channels.exists(
            index_name
        ), f"Index channel {index_name} should not appear in UI"
        _debug_timing(f"test_delete_channel() index exists check in {time.time() - start:.2f}s, TOTAL: {time.time() - test_start:.2f}s")

    def test_copy_link(self) -> None:
        """Test copying a channel link via context menu."""
        self.log("Testing copy channel link")
        _debug_timing("test_copy_link() START")
        test_start = time.time()

        link = self.console.channels.copy_link(self.shared_data)

        channel = self.client.channels.retrieve(self.shared_data)
        assert_link_format(link, "channel", str(channel.key))
        _debug_timing(f"test_copy_link() DONE in {time.time() - test_start:.2f}s")

    def test_open_channel_plot_by_name(self) -> None:
        """Test opening a channel plot by searching its name in the command palette."""
        self.log("Testing open channel plot by name via command palette")

        plot = Plot.open_from_search(self.client, self.console, self.shared_data)
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

    def test_plot_calculated_channel(self) -> None:
        """Test plotting a nested calculated channel (calc channel referencing another calc channel)."""
        self.log("Testing plot nested calculated channel")

        client = self.client

        plot = Plot(client, self.console, f"Nested Calc Plot {self.suffix}")
        plot.add_channels("Y1", [SRC_CH, self.calc_x2, self.calc_x6])
        csv_content = plot.download_csv()

        assert self.calc_x2 in csv_content, f"CSV should contain {self.calc_x2}"
        assert self.calc_x6 in csv_content, f"CSV should contain {self.calc_x6}"

        lines = csv_content.strip().split("\n")
        header = lines[0].split(",")
        src_idx = header.index(SRC_CH)
        calc_x2_idx = header.index(self.calc_x2)
        calc_x6_idx = header.index(self.calc_x6)

        for line in lines[1:]:
            values = line.split(",")
            src_val = int(values[src_idx])
            calc_x2_val = int(values[calc_x2_idx])
            calc_x6_val = int(values[calc_x6_idx])

            expected_x2 = src_val * 2
            expected_x6 = src_val * 2 * 3
            assert (
                calc_x2_val == expected_x2
            ), f"calc_x2 mismatch: {src_val} * 2 = {expected_x2}, got {calc_x2_val}"
            assert (
                calc_x6_val == expected_x6
            ), f"calc_x6 mismatch: {src_val} * 6 = {expected_x6}, got {calc_x6_val}"

        plot.close()

    def test_erroneous_calculated_channel(self) -> None:
        """Test that erroneous calculated channel expressions are handled gracefully."""
        _debug_timing("test_erroneous_calculated_channel() START")
        test_start = time.time()
        console = self.console
        console.notifications.close_all()

        self.log("Testing erroneous calculated channel (nonexistent channel)")
        calc_name = f"calc_err_{self.suffix}"
        bad_ch_expression = "return nonexistent_channel_xyz * 3"

        _debug_timing("test_erroneous_calculated_channel() creating calc with bad channel")
        start = time.time()
        error = console.channels.create_calculated(
            name=calc_name, expression=bad_ch_expression
        )
        _debug_timing(f"test_erroneous_calculated_channel() bad channel done in {time.time() - start:.2f}s")

        assert error is not None, "Expected error for nonexistent channel"
        assert (
            "Failed to update calculated channel" in error
        ), f"Error should mention failure: {error}"
        assert (
            "undefined symbol" in error
        ), f"Error should mention undefined symbol: {error}"
        assert (
            "nonexistent_channel_xyz" in error
        ), f"Error should mention nonexistent channel: {error}"

        self.log("Testing erroneous calculated channel (bad syntax)")

        bad_syntax_expression = "return * 3"
        calc_name_2 = f"calc_err_syntax_{self.suffix}"

        _debug_timing("test_erroneous_calculated_channel() creating calc with bad syntax")
        start = time.time()
        error = console.channels.create_calculated(
            name=calc_name_2, expression=bad_syntax_expression
        )
        _debug_timing(f"test_erroneous_calculated_channel() bad syntax done in {time.time() - start:.2f}s, TOTAL: {time.time() - test_start:.2f}s")

        assert error is not None, "Expected error for bad syntax"
        assert (
            "Failed to update calculated channel" in error
        ), f"Error should mention failure: {error}"
        assert "error" in error.lower(), f"Error should contain 'error': {error}"
