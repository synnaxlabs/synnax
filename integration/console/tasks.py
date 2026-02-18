#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import random
import re
from typing import Any, Literal

from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.context_menu import ContextMenu
from console.layout import LayoutClient
from framework.utils import get_results_path


class TaskClient:
    """Task toolbar management for Console UI automation."""

    ITEM_SELECTOR = ".pluto-list__item:has(.console-task__metadata)"

    def __init__(self, layout: LayoutClient):
        self.layout = layout
        self.ctx_menu = ContextMenu(layout.page)

    def show_toolbar(self) -> None:
        """Show the task toolbar in the left sidebar."""
        self.layout.show_resource_toolbar("task")

    def get_item(self, name: str) -> Locator:
        """Get a task item locator from the task toolbar.

        Args:
            name: Name of the task.

        Returns:
            Locator for the task item.
        """
        title = self.layout.page.locator(".console-task__title p").filter(
            has_text=re.compile(f"^{re.escape(name)}$")
        )
        return self.layout.page.locator(self.ITEM_SELECTOR).filter(has=title).first

    def _select_tasks(self, names: list[str]) -> Locator:
        """Multi-select tasks via Ctrl+Click, return the last item."""
        self.show_toolbar()
        return self.layout.ctrl_select_items(names, self.get_item)

    def wait_for_task(
        self,
        name: str,
    ) -> None:
        """Wait for a task to appear in the task toolbar."""
        self.show_toolbar()
        self.get_item(name).wait_for(state="visible", timeout=30000)

    def _is_running(self, item: Locator) -> bool:
        """Check if a task item is currently running by its icon."""
        return item.locator("button:has(.pluto-icon--pause)").is_visible()

    def wait_for_state(self, name: str, state: Literal["running", "stopped"]) -> None:
        """Wait for a task to reach the expected running state in the UI.

        Args:
            name: Name of the task.
            state: Expected state - "running" or "stopped".
        """
        self.show_toolbar()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        icon_class = "pause" if state == "running" else "play"
        item.locator(f"button:has(.pluto-icon--{icon_class})").wait_for(
            state="visible", timeout=15000
        )

    def rename_task(self, old_name: str, new_name: str) -> None:
        """Rename a task via context menu in the task toolbar.

        Uses inline text edit (Text.MaybeEditable), not a modal dialog.

        Args:
            old_name: Current name of the task.
            new_name: New name for the task.
        """
        self.show_toolbar()
        item = self.get_item(old_name)
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.ctx_menu.action(item, "Rename")
        self.layout.page.locator("[contenteditable='true']").first.wait_for(
            state="visible", timeout=5000
        )
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        self._handle_rename_confirmation()
        self.get_item(new_name).wait_for(state="visible", timeout=5000)

    def _handle_rename_confirmation(self) -> None:
        """Handle the rename confirmation dialog if the task is running."""
        try:
            rename_btn = self.layout.page.get_by_role(
                "button", name="Rename", exact=True
            )
            rename_btn.wait_for(state="visible", timeout=2000)
            rename_btn.click()
        except PlaywrightTimeoutError:
            pass

    def delete_task(self, name: str) -> None:
        """Delete a task via context menu in the task toolbar.

        Args:
            name: Name of the task to delete.
        """
        self.show_toolbar()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.layout.delete_with_confirmation(item)
        item.wait_for(state="hidden", timeout=5000)

    def delete_tasks(self, names: list[str]) -> None:
        """Delete multiple tasks via multi-select and context menu.

        Args:
            names: Names of the tasks to delete.
        """
        if not names:
            return
        last = self._select_tasks(names)
        self.ctx_menu.action(last, "Delete")
        self.layout.confirm_delete()
        for name in names:
            self.get_item(name).wait_for(state="hidden", timeout=5000)

    def export_task(self, name: str) -> dict[str, Any]:
        """Export a task via context menu in the task toolbar.

        Args:
            name: Name of the task to export.

        Returns:
            The exported JSON content as a dictionary.
        """
        self.show_toolbar()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.ctx_menu.open_on(item)
        self.layout.page.evaluate("delete window.showSaveFilePicker")
        with self.layout.page.expect_download(timeout=5000) as download_info:
            self.ctx_menu.click_option("Export")
        download = download_info.value
        save_path = get_results_path(f"{name}_export.json")
        download.save_as(save_path)
        with open(save_path, "r") as f:
            result: dict[str, Any] = json.load(f)
            return result

    def copy_link(self, name: str) -> str:
        """Copy a link to a task via context menu.

        Args:
            name: Name of the task.

        Returns:
            The copied link from clipboard.
        """
        self.show_toolbar()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.ctx_menu.action(item, "Copy link")
        return self.layout.read_clipboard()

    def start_task(self, name: str) -> None:
        """Start a single task by clicking its play button. No-op if already running.

        Waits for the task to reach the running state before returning.

        Args:
            name: Name of the task to start.
        """
        self.show_toolbar()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        if self._is_running(item):
            return
        item.locator("button:has(.pluto-icon--play)").click()
        self.wait_for_state(name, "running")

    def stop_task(self, name: str) -> None:
        """Stop a single task by clicking its pause button. No-op if already stopped.

        Waits for the task to reach the stopped state before returning.

        Args:
            name: Name of the task to stop.
        """
        self.show_toolbar()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        if not self._is_running(item):
            return
        item.locator("button:has(.pluto-icon--pause)").click()
        self.wait_for_state(name, "stopped")

    def start_tasks(self, names: list[str]) -> None:
        """Start multiple tasks via multi-select and context menu.

        Waits for all tasks to reach the running state before returning.

        Args:
            names: Names of the tasks to start.
        """
        if not names:
            return
        last = self._select_tasks(names)
        self.ctx_menu.action(last, "Start")
        for name in names:
            self.wait_for_state(name, "running")

    def stop_tasks(self, names: list[str]) -> None:
        """Stop multiple tasks via multi-select and context menu.

        Waits for all tasks to reach the stopped state before returning.

        Args:
            names: Names of the tasks to stop.
        """
        if not names:
            return
        last = self._select_tasks(names)
        self.ctx_menu.action(last, "Stop")
        for name in names:
            self.wait_for_state(name, "stopped")

    def disable_data_saving(self, name: str) -> None:
        """Disable data saving for a single task. No-op if already disabled or
        unsupported (e.g. write tasks).

        Args:
            name: Name of the task.
        """
        self._toggle_data_saving(name, enable=False)

    def enable_data_saving(self, name: str) -> None:
        """Enable data saving for a single task. No-op if already enabled or
        unsupported (e.g. write tasks).

        Args:
            name: Name of the task.
        """
        self._toggle_data_saving(name, enable=True)

    def _toggle_data_saving(self, name: str, *, enable: bool) -> None:
        action = "Enable data saving" if enable else "Disable data saving"
        opposite = "Disable data saving" if enable else "Enable data saving"
        self.show_toolbar()
        self.layout.page.wait_for_timeout(300)
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.ctx_menu.open_on(item)

        menu = self.ctx_menu._visible_menu()
        action_loc = menu.get_by_text(action, exact=True).first
        opposite_loc = menu.get_by_text(opposite, exact=True).first
        try:
            action_loc.or_(opposite_loc).wait_for(state="visible", timeout=5000)
        except PlaywrightTimeoutError:
            self.ctx_menu.close()
            return

        if action_loc.is_visible():
            self.ctx_menu.click_option(action)
        else:
            self.ctx_menu.close()

    def disable_data_saving_tasks(self, names: list[str]) -> None:
        """Disable data saving for multiple tasks via multi-select and context menu.

        Args:
            names: Names of the tasks.
        """
        self._toggle_data_saving_tasks(names, enable=False)

    def enable_data_saving_tasks(self, names: list[str]) -> None:
        """Enable data saving for multiple tasks via multi-select and context menu.

        Args:
            names: Names of the tasks.
        """
        self._toggle_data_saving_tasks(names, enable=True)

    def _toggle_data_saving_tasks(self, names: list[str], *, enable: bool) -> None:
        if not names:
            return
        action = "Enable data saving" if enable else "Disable data saving"
        last = self._select_tasks(names)
        self.ctx_menu.action(last, action)

    def open_task_config(self, name: str) -> None:
        """Open a task's configuration via context menu or double-click (random).

        Args:
            name: Name of the task to open.
        """
        self.show_toolbar()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        if random.random() < 0.5:
            item.click()
            self.ctx_menu.action(item, "Edit configuration")
        else:
            item.dblclick()

    def snapshot_tasks(self, names: list[str], range_name: str) -> None:
        """Snapshot tasks to the active range via context menu.

        Args:
            names: Names of the tasks to snapshot.
            range_name: Name of the active range.
        """
        if not names:
            return
        last = self._select_tasks(names)
        self.ctx_menu.action(last, f"Snapshot to {range_name}", exact=False)
