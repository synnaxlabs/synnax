#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.case import ConsoleCase
from console.log import Log
from framework.utils import assert_link_format, get_random_name


class LogLifecycle(ConsoleCase):
    """Test log operations including rename from mosaic tab title."""

    suffix: str
    idx_name: str
    data_name: str
    virtual_name: str
    _shared_log_name: str | None
    _cleanup_groups: list[str]
    ctx_log_name: str
    ctx_log_link: str

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self._shared_log_name = None
        self._cleanup_groups = []
        self.ctx_log_name = ""

    def setup_channels(self) -> None:
        """Create all test channels."""
        self.idx_name = f"log_test_idx_{self.suffix}"
        self.data_name = f"log_test_data_{self.suffix}"
        self.virtual_name = f"log_virtual_{self.suffix}"

        index_ch = self.client.channels.create(
            name=self.idx_name,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name=self.data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name=self.virtual_name,
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )

    def teardown(self) -> None:
        for name in self._cleanup_groups:
            try:
                self.console.workspace.delete_group(name)
            except PlaywrightTimeoutError:
                pass
        if self._shared_log_name is not None:
            self._cleanup_pages.append(self._shared_log_name)
        super().teardown()

    def run(self) -> None:
        """Run all log lifecycle tests."""
        self.setup_channels()

        log = self.console.workspace.create_log(f"Log Test {self.suffix}")
        self._shared_log_name = log.page_name

        self.test_no_channel_configured(log)
        self.test_no_data_received(log)
        self.test_persisted_channel_streaming(log)
        self.test_rename_from_tab(log)
        self.test_copy_link(log)
        self.test_pause_resume_scrolling(log)
        self.test_copy_all_text(log)
        self.test_show_timestamps(log)
        self.test_virtual_channel_streaming(log)

        log_link = log.copy_link()
        log_name = log.page_name
        self._shared_log_name = log_name
        log.close()
        assert not log.is_open, "Log should be closed after close()"

        # Resources Toolbar
        self.test_open_log_from_resources(log_name, log_link)
        self.test_drag_log_onto_mosaic(log_name, log_link)

        # Search and Command Palette
        self.test_open_log_from_search(log_name, log_link)

        # Clean up shared log before ctx tests to prevent accumulation
        self.console.workspace.delete_page(log_name)
        self._shared_log_name = None

        # Resources Toolbar > Context Menu
        ctx_log = self.console.workspace.create_log(f"Context Menu Test {self.suffix}")
        self.ctx_log_name = ctx_log.page_name
        self._cleanup_pages.append(self.ctx_log_name)
        self.ctx_log_link = ctx_log.copy_link()
        ctx_log.close()

        self.test_ctx_operations()
        self.test_ctx_group_logs()
        self.test_ctx_delete_multiple_logs()
        self.test_ctx_delete_log()

    def test_no_channel_configured(self, log: Log) -> None:
        """Test that log shows 'No channel configured' when no channel is set."""
        self.log("Testing no channel configured state")

        assert log.needs_channel_configured(), "Log should show 'No channel configured'"
        assert log.is_empty(), "Log should be empty when no channel configured"
        assert not log.is_streaming(), "Log should not be streaming without channel"

    def test_no_data_received(self, log: Log) -> None:
        """Test that log shows 'No data received yet' for channel with no data."""
        self.log("Testing no data received state")

        log.set_channel(self.data_name)

        assert log.is_waiting_for_data(), "Log should show 'No data received yet'"
        assert log.is_empty(), "Log should be empty when no data received"
        assert not log.is_streaming(), "Log should not be streaming without data"

    def test_persisted_channel_streaming(self, log: Log) -> None:
        """Test that log streams data from a persisted channel and survives reload."""
        self.log("Testing persisted channel streaming")

        with self.client.open_writer(
            sy.TimeStamp.now(),
            channels=[self.idx_name, self.data_name],
        ) as w:
            for i in range(5):
                w.write({self.idx_name: sy.TimeStamp.now(), self.data_name: (42.0 + i)})
                sy.sleep(0.1)

        assert log.wait_until_streaming(), "Log should be streaming after data write"
        assert not log.is_empty(), "Log should not be empty after data write"
        assert not log.is_waiting_for_data(), "Log should not be waiting for data"

        self.console.reload()

        assert (
            log.wait_until_streaming()
        ), "Log should still be streaming after reload (persisted)"
        assert (
            not log.is_waiting_for_data()
        ), "Log should NOT be waiting for data after reload"

    def test_virtual_channel_streaming(self, log: Log) -> None:
        """Test that log streams data from a virtual (non-persisted) channel."""
        self.log("Testing virtual channel streaming")

        log.set_channel(self.virtual_name)
        assert (
            log.wait_until_waiting_for_data()
        ), "Log should be waiting for data initially (virtual channel)"

        with self.client.open_writer(
            sy.TimeStamp.now(),
            channels=[self.virtual_name],
            enable_auto_commit=True,
        ) as writer:
            for i in range(5):
                writer.write({self.virtual_name: float(i)})
                sy.sleep(0.1)
            assert (
                log.wait_until_streaming()
            ), "Log should be streaming virtual channel data"
            assert not log.is_empty(), "Log should not be empty with virtual data"

        self.console.reload()

        assert (
            log.wait_until_waiting_for_data()
        ), "Log should be waiting for data after reload (virtual channel not persisted)"

    def test_rename_from_tab(self, log: Log) -> None:
        """Test renaming a log by double-clicking the mosaic tab title."""
        self.log("Testing rename log from mosaic tab title")

        original_name = log.page_name
        new_name = f"Renamed Log {self.suffix}"

        log.rename(new_name)

        new_tab = self.console.layout.get_tab(new_name)
        assert new_tab.is_visible(), f"Tab with new name '{new_name}' should be visible"

        old_tab = self.console.layout.get_tab(original_name)
        assert (
            old_tab.count() == 0
        ), f"Tab with old name '{original_name}' should not exist"

    def test_copy_link(self, log: Log) -> None:
        """Test copying a link to the log via toolbar button."""
        self.log("Testing copy link to log")

        link = log.copy_link()

        assert_link_format(link, "log")

    def test_pause_resume_scrolling(self, log: Log) -> None:
        """Test pausing and resuming log scrolling."""
        self.log("Testing pause/resume scrolling")

        assert not log.is_scrolling_paused(), "Log should not be paused initially"

        log.pause_scrolling()
        assert log.is_scrolling_paused(), "Log should be paused after pause_scrolling()"

        log.resume_scrolling()
        assert (
            not log.is_scrolling_paused()
        ), "Log should not be paused after resume_scrolling()"

    def test_copy_all_text(self, log: Log) -> None:
        """Test that Copy All button copies log text to clipboard."""
        self.log("Testing copy all text")
        copied = log.click_copy_all()
        assert len(copied) > 0, "Copied text should not be empty"
        assert "42" in copied, "Copied text should contain written data"

    def test_show_timestamps(self, log: Log) -> None:
        """Test enabling Show Timestamps and verifying timestamps in copied text."""
        self.log("Testing show timestamps")
        log.toggle_show_timestamps(True)
        assert log.has_timestamps_enabled(), "Timestamps should be enabled"

        copied = log.click_copy_all()
        assert (
            "[" in copied and "]" in copied
        ), "Copied text should contain timestamp brackets"

        log.toggle_show_timestamps(False)
        assert not log.has_timestamps_enabled(), "Timestamps should be disabled"

        copied_no_ts = log.click_copy_all()
        assert (
            "[" not in copied_no_ts
        ), "Copied text should not contain timestamps when disabled"

    def test_open_log_from_resources(self, log_name: str, expected_link: str) -> None:
        """Test opening a log by double-clicking it in the workspace resources toolbar."""
        self.log("Testing open log from resources toolbar")

        log = self.console.workspace.open_log(log_name)

        assert log.pane_locator is not None, "Log pane should be visible"
        assert log.pane_locator.is_visible(), "Log pane should be visible"

        opened_link = log.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened log link should match: expected {expected_link}, got {opened_link}"

        log.close()
        assert not log.is_open, "Log should be closed after close()"

    def test_drag_log_onto_mosaic(self, log_name: str, expected_link: str) -> None:
        """Test dragging a log from the resources toolbar onto the mosaic."""
        self.log("Testing drag log onto mosaic")

        log = self.console.workspace.drag_log_to_mosaic(log_name)

        assert log.pane_locator is not None, "Log pane should be visible"
        assert log.pane_locator.is_visible(), "Log pane should be visible"

        opened_link = log.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened log link should match: expected {expected_link}, got {opened_link}"

        log.close()
        assert not log.is_open, "Log should be closed after close()"

    def test_open_log_from_search(self, log_name: str, expected_link: str) -> None:
        """Test opening a log by searching its name in the command palette."""
        self.log("Testing open log from search palette")

        log = self.console.workspace.open_from_search(Log, log_name)

        assert log.pane_locator is not None, "Log pane should be visible"
        assert log.pane_locator.is_visible(), "Log pane should be visible"

        opened_link = log.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened log link should match: expected {expected_link}, got {opened_link}"

        log.close()
        assert not log.is_open, "Log should be closed after close()"

    def test_ctx_operations(self) -> None:
        """Test non-destructive context menu operations using shared log."""
        assert self.ctx_log_name is not None

        self.log("Testing copy link via context menu")
        link = self.console.workspace.copy_page_link(self.ctx_log_name)
        assert (
            link == self.ctx_log_link
        ), f"Context menu link should match: expected {self.ctx_log_link}, got {link}"

        self.log("Testing export log via context menu")
        exported = self.console.workspace.export_page(self.ctx_log_name)
        assert "key" in exported, "Exported JSON should contain 'key'"
        assert len(exported["key"]) == 36, "Log key should be a UUID"

        self.log("Testing rename log via context menu")
        new_name = f"Renamed Log {self.suffix}"
        self.console.workspace.rename_page(self.ctx_log_name, new_name)
        assert self.console.workspace.page_exists(
            new_name
        ), f"Renamed log '{new_name}' should exist"
        self.ctx_log_name = new_name

    def test_ctx_delete_log(self) -> None:
        """Test deleting a log via context menu in the workspace resources toolbar."""
        self.log("Testing delete log via context menu")
        assert self.ctx_log_name is not None
        self.console.workspace.delete_page(self.ctx_log_name)

    def test_ctx_delete_multiple_logs(self) -> None:
        """Test deleting multiple logs via multi-select and context menu."""
        self.log("Testing delete multiple logs via context menu")

        suffix = get_random_name()
        log_names = []

        for i in range(3):
            log = self.console.workspace.create_log(f"Multi Delete {suffix} {i}")
            log_names.append(log.page_name)
            self._cleanup_pages.append(log.page_name)
            log.close()

        self.console.workspace.delete_pages(log_names)
        for name in log_names:
            self._cleanup_pages.remove(name)

    def test_ctx_group_logs(self) -> None:
        """Test grouping multiple logs via multi-select and context menu."""
        self.log("Testing group logs via context menu")

        suffix = get_random_name()
        log_names = []

        for i in range(2):
            log = self.console.workspace.create_log(f"Group Test {suffix} {i}")
            log_names.append(log.page_name)
            self._cleanup_pages.append(log.page_name)
            log.close()

        group_name = f"Log Group {suffix}"
        self.console.workspace.group_pages(names=log_names, group_name=group_name)
        self._cleanup_groups.append(group_name)

        assert self.console.workspace.page_exists(
            group_name
        ), "Group should exist after grouping"

        self.console.workspace.delete_group(group_name, child_names=log_names)
        self._cleanup_groups.remove(group_name)
        for name in log_names:
            self._cleanup_pages.remove(name)
