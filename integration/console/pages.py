#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import random
import re
import tempfile
from typing import Any, TypeVar

from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

import synnax as sy
from console.base import ResourceClient
from console.channels import ChannelClient
from console.layout import LayoutClient
from console.page import ConsolePage, PageType
from console.plot import Plot
from console.project import ProjectClient
from framework.run_dir import resolve_results_path
from framework.utils import resolve_channel_placeholders

T = TypeVar("T", bound=ConsolePage)


class PagesClient(ResourceClient):
    """Page lifecycle within the active project: creation, the project resource
    tree, and the mosaic."""

    def __init__(
        self,
        layout: LayoutClient,
        client: sy.Synnax,
        project: ProjectClient,
    ):
        super().__init__(layout)
        self.client = client
        self.project = project

    def create(self, page_class: type[T], name: str) -> T:
        """Create a page of ``page_class`` in the UI and return a typed wrapper.

        :param page_class: The page class to create (Plot, Log, Schematic, ...).
        :param name: Name for the new page.
        :returns: Instance of ``page_class`` wrapping the created UI page.
        """
        pane, tab, actual_name = self.create_by_type(page_class.page_type, name)
        page = page_class(self.layout, self.client, actual_name, pane_locator=pane)
        page._initialize_from_project(tab, actual_name)
        return page

    def open(self, page_class: type[T], name: str) -> T:
        """Open a page by double-clicking it in the project resources toolbar.

        :param page_class: The page class to wrap the opened page with.
        :param name: Name of the page to open.
        :returns: Instance of ``page_class`` wrapping the opened page.
        """
        self.open_by_name(name)
        return page_class.from_open_page(self.layout, self.client, name)

    def open_by_name(self, name: str) -> None:
        """Open a page by double-clicking it in the project resources toolbar,
        without wrapping it in a page class.

        :param name: Name of the page to open.
        """
        page_item = self._find(name)
        page_item.dblclick()
        self.layout.close_left_toolbar()

    def drag_to_mosaic(self, page_class: type[T], name: str) -> T:
        """Drag a page from the project resources toolbar onto the mosaic.

        :param page_class: The page class to wrap the opened page with.
        :param name: Name of the page to drag.
        :returns: Instance of ``page_class`` wrapping the opened page.
        """
        for attempt in range(2):
            self.layout.show_resource_toolbar("Projects")
            page_item = self._find(name)
            mosaic = self.layout.page.locator(".console-mosaic").first
            page_item.drag_to(mosaic)
            self.layout.close_left_toolbar()
            try:
                self.layout.wait_for_tab(name)
                break
            except PlaywrightTimeoutError:
                if attempt == 1:
                    raise
        return page_class.from_open_page(self.layout, self.client, name)

    def bind_open(self, page_class: type[T], name: str) -> T:
        """Bind a wrapper to a tab that is already open, without opening one.

        Opening a page writes the panel document, so a user without panel write
        can only reach a view someone else left open. This focuses that tab and
        wraps its pane.

        :param page_class: The page class to instantiate.
        :param name: Name of the already-open tab.
        :returns: Instance of ``page_class`` wrapping the open page.
        """
        self.layout.get_read_only_tab(name).wait_for(state="visible", timeout=10000)
        self.layout.focus(name)
        pane = self.layout.page.locator(page_class.pluto_label)
        pane.first.wait_for(state="visible", timeout=5000)
        return page_class(self.layout, self.client, name, pane_locator=pane.first)

    def open_from_search(self, page_class: type[T], name: str) -> T:
        """Open an existing page by searching its name in the command palette.

        :param page_class: The page class to instantiate.
        :param name: Name to search for (page name or channel name).
        :returns: Instance of ``page_class`` wrapping the opened page.
        """
        self.layout.search_palette(name)

        pane = self.layout.page.locator(page_class.pluto_label)
        pane.first.wait_for(state="visible", timeout=5000)

        active_tab = (
            self.layout.page.locator(LayoutClient.TAB_SELECTOR)
            .filter(has=self.layout.page.locator("[aria-label='Close']"))
            .last
        )
        actual_name = active_tab.inner_text().strip()

        return page_class(
            self.layout, self.client, actual_name, pane_locator=pane.first
        )

    def open_plot_from_click(self, channel_name: str, channels: ChannelClient) -> Plot:
        """Open a plot by double-clicking a channel in the channels sidebar.

        :param channel_name: Name of the channel to double-click.
        :param channels: ChannelClient for showing/hiding the channels sidebar.
        :returns: Plot instance for the opened plot.
        """
        channels.show_channels()

        channel_item = self.tree.find_by_name("channel:", channel_name)
        if channel_item is None:
            raise ValueError(f"Channel '{channel_name}' not found")
        channel_item.wait_for(state="visible", timeout=5000)
        channel_item.dblclick()

        plot_pane = self.layout.page.locator(".pluto-line-plot")
        plot_pane.first.wait_for(state="visible", timeout=5000)

        tabs = self.layout.page.locator(LayoutClient.TAB_SELECTOR).filter(
            has=self.layout.page.locator("[aria-label='Close']")
        )
        tab_count = tabs.count()
        actual_tab_name = "Line plot"
        if tab_count > 0:
            last_tab = tabs.nth(tab_count - 1)
            actual_tab_name = last_tab.inner_text().strip()

        plot = Plot.from_open_page(self.layout, self.client, actual_tab_name)

        channels.hide_channels()
        return plot

    def create_by_type(
        self, page_type: PageType, name: str | None = None
    ) -> tuple[Locator, Locator, str]:
        """Create a page via the New Page (+) button or command palette
        (randomly chosen).

        :returns: Tuple of (pane_locator, tab_locator, page_id).
        """
        self.layout.close_left_toolbar()
        if random.random() < 0.5:
            return self.create_by_new_page_button(page_type, name)
        return self.create_by_command_palette(page_type, name)

    def create_by_new_page_button(
        self, page_type: PageType, name: str | None = None
    ) -> tuple[Locator, Locator, str]:
        """Create a page via the New Page (+) button.

        :returns: Tuple of (pane_locator, tab_locator, page_id).
        """
        self.layout.close_left_toolbar()
        add_btn = self.layout.page.locator(
            f"{LayoutClient.TAB_STRIP_SELECTOR} button:has(.pluto-icon--add)"
        ).first
        add_btn.wait_for(state="visible", timeout=5000)
        add_btn.dispatch_event("click")
        layout_selector = self.layout.page.locator(".console-layout-selector__frame")
        # Retry once if the layout selector doesn't appear — on Windows CI
        # the first dispatch can race with notification rendering.
        try:
            layout_selector.wait_for(state="visible", timeout=5000)
        except PlaywrightTimeoutError:
            add_btn.dispatch_event("click")
            layout_selector.wait_for(state="visible", timeout=15000)
        type_btn = self.layout.page.get_by_role("button", name=page_type).first
        type_btn.wait_for(state="visible", timeout=5000)
        type_btn.click()

        return self._handle_new_page(page_type, name)

    def create_by_command_palette(
        self, page_type: PageType, name: str | None = None
    ) -> tuple[Locator, Locator, str]:
        """Create a page via the command palette.

        :returns: Tuple of (pane_locator, tab_locator, page_id).
        """
        self.layout.close_left_toolbar()

        self.layout.command_palette(f"Create {page_type}")
        return self._handle_new_page(page_type, name)

    def _handle_new_page(
        self, page_type: PageType, name: str | None = None
    ) -> tuple[Locator, Locator, str]:
        """Handle the new page creation after clicking create button.

        :returns: Tuple of (pane_locator, tab_locator, page_id).
        """
        modal_was_open = self.layout.is_modal_open()
        tab_name: str = page_type

        if modal_was_open:
            tab_name = name if name is not None else page_type
            name_input = self.layout.page.get_by_role("textbox", name="Name")
            name_input.fill(tab_name)
            name_input.press("ControlOrMeta+Enter")

        page_tab = self.layout.get_tab(tab_name)
        page_tab.wait_for(state="visible", timeout=15000)
        page_id = page_tab.inner_text().strip()

        if name is not None and not modal_was_open:
            self.layout.rename_tab(old_name=tab_name, new_name=name)
            page_id = name
            page_tab = self.layout.get_tab(name)

        pluto_labels = {
            "Log": ".pluto-log",
            "Line plot": ".pluto-line-plot",
            "Schematic": ".pluto-schematic",
            "Table": ".pluto-table",
        }
        pluto_label = pluto_labels.get(page_type, "")
        if pluto_label:
            pane = self.layout.page.locator(pluto_label).first
            pane.wait_for(state="visible", timeout=5000)
        else:
            pane = page_tab

        return pane, page_tab, page_id

    def expand_active(self) -> None:
        """Expand the active project in the resources toolbar to show its
        contents.

        Targets the active project by name. Concurrently-running tests each add
        their own project to the shared tree, so expanding whichever node sorts
        first (expand_root) would expand the wrong project and the test's pages
        would never be revealed. Falls back to expand_root if no project has
        been activated yet.
        """
        self.layout.show_resource_toolbar("Projects")
        active = self.project.active
        if active is not None:
            self.tree.expand_named(ProjectClient.ITEM_PREFIX, active)
        else:
            self.tree.expand_root(ProjectClient.ITEM_PREFIX)
        self.layout.page.get_by_role("treeitem").first.wait_for(
            state="visible", timeout=5000
        )

    def get(self, name: str) -> Locator:
        """Get a page item locator from the project resources toolbar.

        Matches the page label exactly; substring matches are ignored so a
        rename to a superstring (e.g. ``foo`` → ``foo_renamed``) does not
        leave the original locator pointing at the renamed item.

        :param name: Name of the page (schematic, line plot, etc.).
        :returns: Locator for the page item.
        """
        pattern = re.compile(rf"^\s*{re.escape(name)}\s*$")
        return self.layout.page.get_by_role("treeitem").filter(has_text=pattern).first

    def _scroll_to(self, name: str) -> bool:
        """Scroll the project tree to find a page that may be off-screen.

        The Pluto tree uses virtual scrolling, so items outside the viewport
        are not in the DOM. This method scrolls the tree container
        incrementally until the item appears or the end is reached.

        :returns: True if the item was found and scrolled into view.
        """
        page_item = self.get(name)
        try:
            page_item.wait_for(state="attached", timeout=1000)
            page_item.scroll_into_view_if_needed()
            return True
        except PlaywrightTimeoutError:
            pass
        container = self.layout.page.get_by_role("tree").first
        try:
            container.wait_for(state="attached", timeout=2000)
        except PlaywrightTimeoutError:
            return False
        # The list is virtualized, so the wheel is the only way to reach detached rows.
        # The rendered text stops changing once the list hits bottom.
        container.hover()
        prev_rows: str | None = None
        for _ in range(50):
            curr_rows = container.inner_text()
            if prev_rows is not None and curr_rows == prev_rows:
                break
            prev_rows = curr_rows
            self.layout.page.mouse.wheel(0, 200)
            self.layout.page.wait_for_timeout(100)
            try:
                page_item.wait_for(state="attached", timeout=300)
                page_item.scroll_into_view_if_needed()
                return True
            except PlaywrightTimeoutError:
                continue
        return False

    def exists(self, name: str) -> bool:
        """Check if a page (schematic, line plot, etc.) exists in the project."""
        self.expand_active()
        if self._scroll_to(name):
            page_item = self.get(name)
            try:
                page_item.wait_for(state="visible", timeout=5000)
                return True
            except PlaywrightTimeoutError:
                pass
        return False

    def wait_for_removed(self, name: str) -> None:
        """Wait for a page to be removed from the project."""
        page_item = self.get(name)
        page_item.wait_for(state="hidden", timeout=5000)

    def _find(self, name: str) -> Locator:
        """Find a page in the project tree, scrolling if needed.

        First tries a simple wait (handles most cases where the tree isn't
        crowded). Falls back to scrolling for virtual-scrolled trees.

        :raises PlaywrightTimeoutError: If the page cannot be found.
        """
        self.expand_active()
        page_item = self.get(name)
        try:
            page_item.wait_for(state="visible", timeout=5000)
            return page_item
        except PlaywrightTimeoutError:
            pass
        if not self._scroll_to(name):
            raise PlaywrightTimeoutError(f"Page '{name}' not found in project tree")
        page_item = self.get(name)
        page_item.wait_for(state="visible", timeout=5000)
        return page_item

    def rename(self, old_name: str, new_name: str) -> None:
        """Rename a page via context menu in the project resources toolbar.

        :param old_name: Current name of the page.
        :param new_name: New name for the page.
        """
        page_item = self._find(old_name)
        self.ctx_menu.action(page_item, "Rename")
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        self.get(new_name).wait_for(state="visible", timeout=5000)
        self.wait_for_removed(old_name)
        self.layout.close_left_toolbar()

    def delete(self, name: str) -> None:
        """Delete a page via context menu in the project resources toolbar.

        :param name: Name of the page to delete.
        """
        page_item = self._find(name)
        self.ctx_menu.action(page_item, "Delete")
        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        # Notifications stack over the confirmation dialog and swallow the click.
        self.notifications.close_all()
        delete_btn.click(timeout=5000)
        self.wait_for_removed(name)
        self.layout.close_left_toolbar()

    def delete_group(self, name: str, child_names: list[str] | None = None) -> None:
        """Delete a group via context menu.

        :param name: Name of the group to delete.
        :param child_names: Names of child pages to delete first. Required
            because the Pluto tree uses flat rendering (children are siblings,
            not nested inside the group div), so they must be found by name.
        """
        if child_names:
            self.expand_active()
            group = self.tree.get_group(name)
            self.tree.expand(group)
            for child_name in child_names:
                page_item = self.get(child_name)
                page_item.wait_for(state="attached", timeout=5000)
                page_item.scroll_into_view_if_needed()
                self.ctx_menu.action(page_item, "Delete")
                delete_btn = self.layout.page.get_by_role(
                    "button", name="Delete", exact=True
                )
                delete_btn.wait_for(state="visible", timeout=5000)
                delete_btn.click(timeout=5000)
                self.wait_for_removed(child_name)
        self.expand_active()
        self.tree.delete_group(name)
        self.layout.close_left_toolbar()

    def delete_many(self, names: list[str]) -> None:
        """Delete pages by walking the project tree depth-first.

        Expands groups as needed to find target pages, deletes each
        individually, and cleans up empty groups on the way back up.
        Tracks visited groups by DOM id to avoid infinite loops in the
        flat-rendered tree. Falls back to scrolling for pages that may
        be off-screen due to virtual scrolling.

        :param names: Names of pages to delete.
        """
        if not names:
            return
        remaining = list(names)
        try:
            self.expand_active()
        except PlaywrightTimeoutError:
            self.layout.close_left_toolbar()
            return
        self._walk_and_delete(remaining, visited=set())
        # Scroll fallback for pages missed by the walk (virtual scrolling).
        for name in list(remaining):
            if not self._scroll_to(name):
                continue
            page_item = self.get(name)
            if not page_item.is_visible():
                continue
            self._delete_item(page_item, name, remaining)
        self.layout.close_left_toolbar()

    def _delete_item(self, page_item: Locator, name: str, remaining: list[str]) -> None:
        self.ctx_menu.action(page_item, "Delete")
        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        delete_btn.wait_for(state="hidden", timeout=5000)
        remaining.remove(name)

    def _walk_and_delete(self, remaining: list[str], visited: set[str]) -> None:
        # Delete any visible target pages at this level.
        for name in list(remaining):
            page_item = self.get(name)
            if not page_item.is_visible():
                continue
            self._delete_item(page_item, name, remaining)

        # Recurse into unvisited groups, then delete them on the way back up.
        for group in self.tree.find_by_prefix("group:"):
            group_id = group.get_attribute("id") or ""
            if group_id in visited:
                continue
            visited.add(group_id)
            self.tree.expand(group)
            self._walk_and_delete(remaining, visited)
            # Post-order: delete the group only if it's now empty.
            locator = self.layout.page.locator(f"div[id='{group_id}']").first
            try:
                locator.wait_for(state="visible", timeout=2000)
                self.tree.delete_group(locator, only_if_empty=True)
            except PlaywrightTimeoutError:
                pass

    def copy_link(self, name: str) -> str:
        """Copy link to a page via context menu.

        :param name: Name of the page to copy the link for.
        :returns: The copied link from the clipboard.
        """
        page_item = self._find(name)
        self.ctx_menu.action(page_item, "Copy link")
        self.layout.close_left_toolbar()
        return self.layout.read_clipboard()

    def group(self, *, names: list[str], group_name: str) -> None:
        """Group multiple pages into a folder via multi-select and context
        menu."""
        self.expand_active()
        items = []
        for n in names:
            self._scroll_to(n)
            items.append(self.get(n))
        self.tree.group(items, group_name)
        self.layout.close_left_toolbar()

    def rename_group(self, old_name: str, new_name: str) -> None:
        """Rename a group via context menu."""
        self.expand_active()
        self.tree.rename_group(old_name, new_name)
        self.layout.close_left_toolbar()

    def move_to_group(self, item_name: str, group_name: str) -> None:
        """Move a page or group into a target group via drag-and-drop."""
        self.expand_active()
        self.tree.move_to_group(self.get(item_name), group_name)
        self.layout.close_left_toolbar()

    def export(self, name: str) -> dict[str, Any]:
        """Export a page via context menu.

        The file is saved to the tests/results directory.

        :param name: Name of the page to export.
        :returns: The exported JSON content as a dictionary.
        """
        page_item = self._find(name)
        self.ctx_menu.open_on(page_item)

        with self.layout.page.expect_download(timeout=5000) as download_info:
            self.ctx_menu.click_option("Export")

        download = download_info.value
        save_path = resolve_results_path(f"{name}_export.json")
        download.save_as(save_path)
        self.layout.close_left_toolbar()

        with open(save_path, "r", encoding="utf-8") as f:
            result: dict[str, Any] = json.load(f)
            return result

    def import_file(self, json_path: str, name: str) -> None:
        """Import a component via the "Import components" command palette flow.

        The import pipeline derives the tab name from the chosen filename (via
        trimFileName), so we copy ``json_path`` into a temp file named
        ``{name}.json`` to control the resulting tab name independently of
        the source fixture's filename.

        Waits for the page to appear in the project resource tree before
        returning, which proves the server-side resource exists.

        :param json_path: Path to the JSON file to import.
        :param name: Display name for the imported page tab.
        """
        with tempfile.TemporaryDirectory() as tmp_dir:
            tmp_path = os.path.join(tmp_dir, f"{name}.json")
            with open(json_path, "r", encoding="utf-8") as f:
                text = resolve_channel_placeholders(self.client, f.read())
            with open(tmp_path, "w", encoding="utf-8") as f:
                f.write(text)
            self.layout.choose_import_file(tmp_path)
            self.layout.get_tab(name).wait_for(state="visible", timeout=10000)
            if not self.exists(name):
                raise AssertionError(
                    f"Imported page {name!r} did not appear in project tree"
                )
            self.layout.close_left_toolbar()

    def import_file_expect_error(self, json_path: str, error_text: str) -> None:
        """Import a file the Core must reject, and assert the error surfaces.

        The import pipeline reports a per-file failure as a "Failed to import"
        notification carrying the Core's error as its description.

        :param json_path: Path to the JSON file to import.
        :param error_text: Substring the error notification must contain.
        """
        self.layout.choose_import_file(json_path)
        if not self.notifications.wait_for("Failed to import"):
            raise AssertionError(
                f"Import of {os.path.basename(json_path)} did not surface a "
                "failure notification"
            )
        if not self.notifications.wait_for(error_text):
            raise AssertionError(
                f"Import failure notification does not mention {error_text!r}"
            )
        self.notifications.close_all()

    def snapshot_to_active_range(self, names: list[str], range_name: str) -> None:
        """Snapshot multiple pages to the active range via context menu.

        :param names: List of page names to snapshot.
        :param range_name: Name of the active range (for menu text matching).
        """
        if not names:
            return
        self.expand_active()
        last = self.layout.ctrl_select_items(names, self.get)
        self.ctx_menu.action(last, f"Snapshot to {range_name}")
        self.layout.close_left_toolbar()

    def copy(self, name: str, new_name: str) -> None:
        """Make a copy of a page via context menu.

        :param name: Name of the page to copy.
        :param new_name: Name for the new copy.
        """
        self.expand_active()
        page_item = self.get(name)
        page_item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(page_item, "Copy")

        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        self.get(new_name).wait_for(state="visible", timeout=5000)
        self.layout.close_left_toolbar()

    def copy_many(self, names: list[str]) -> None:
        """Copy multiple pages via context menu.

        Each copy gets a " (copy)" suffix automatically.

        :param names: List of page names to copy.
        """
        if not names:
            return
        self.expand_active()
        last = self.layout.ctrl_select_items(names, self.get)
        self.ctx_menu.action(last, "Copy")
        for name in names:
            copy_name = f"{name} (copy)"
            self.get(copy_name).wait_for(state="visible", timeout=5000)
        self.layout.close_left_toolbar()
