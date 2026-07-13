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
import shutil
import tempfile
from typing import Any, Literal, TypeVar, overload

from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

import synnax as sy
from console.channels import ChannelClient
from console.context_menu import ContextMenu
from console.layout import LayoutClient
from console.log import Log
from console.notifications import NotificationsClient
from console.page import ConsolePage, PageType
from console.plot import Plot
from console.schematic import Schematic
from console.table import Table
from console.task.analog_read import AnalogRead
from console.task.analog_write import AnalogWrite
from console.task.counter_read import CounterRead
from console.task_page import TaskPage
from console.tree import Tree
from framework.run_dir import resolve_results_path

__all__ = ["ProjectClient", "PageType"]

T = TypeVar("T", bound="ConsolePage")

# Playwright has expect_file_chooser for <input type="file"> but no analogue
# for the File System Access API's showDirectoryPicker, which is what the real
# browser-mode export uses. This snippet installs an in-memory mock that
# captures writes into window.__synnaxExportedFiles as
# { "relativePath": "contents" }; production export code runs unchanged. The
# Python side reads the map back and materializes a real on-disk directory the
# import test can consume verbatim.
_INSTALL_DIRECTORY_PICKER_MOCK_JS = """
window.__synnaxExportedFiles = {};
window.showDirectoryPicker = async () => ({
    name: '__synnaxExportMock',
    getDirectoryHandle: async (subdir, opts) => {
        if (!opts || !opts.create)
            throw new DOMException('Not found', 'NotFoundError');
        return {
            name: subdir,
            getFileHandle: async (name) => ({
                name,
                createWritable: async () => ({
                    write: async (data) => {
                        const text = typeof data === 'string'
                            ? data
                            : await new Response(data).text();
                        window.__synnaxExportedFiles[subdir + '/' + name] = text;
                    },
                    close: async () => {},
                }),
            }),
        };
    },
});
"""


class ProjectClient:
    """Project management for Console UI automation."""

    ITEM_PREFIX = "project:"

    def __init__(
        self,
        layout: LayoutClient,
        client: sy.Synnax,
    ):
        self.layout = layout
        self.client = client
        self.ctx_menu = ContextMenu(layout.page)
        self.notifications = NotificationsClient(layout.page)
        self.tree = Tree(layout.page)
        self._active_project: str | None = None

    def create_page(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, Locator, str]:
        """Create a new page via New Page (+) button or command palette (randomly chosen).

        Returns:
            Tuple of (pane_locator, tab_locator, page_id)
        """
        self.layout.close_left_toolbar()
        if random.random() < 0.5:
            return self.create_page_by_new_page_button(page_type, page_name)
        return self.create_page_by_command_palette(page_type, page_name)

    def create_page_by_new_page_button(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, Locator, str]:
        """Create a new page via the New Page (+) button.

        Returns:
            Tuple of (pane_locator, tab_locator, page_id)
        """
        self.layout.close_left_toolbar()
        add_btn = self.layout.page.locator(
            ".console-mosaic > .pluto-tabs-selector .pluto-tabs-selector__actions button:has(.pluto-icon--add)"
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

        return self._handle_new_page(page_type, page_name)

    def create_page_by_command_palette(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, Locator, str]:
        """Create a new page via command palette.

        Returns:
            Tuple of (pane_locator, tab_locator, page_id)
        """
        self.layout.close_left_toolbar()

        vowels = ["A", "E", "I", "O", "U"]
        article = (
            "an"
            if page_type[0].upper() in vowels or page_type.startswith("NI")
            else "a"
        )
        self.layout.command_palette(f"Create {article} {page_type}")
        return self._handle_new_page(page_type, page_name)

    def _handle_new_page(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, Locator, str]:
        """Handle the new page creation after clicking create button.

        Returns:
            Tuple of (pane_locator, tab_locator, page_id)
        """
        modal_was_open = self.layout.is_modal_open()
        tab_name: str = page_type

        if modal_was_open:
            tab_name = page_name if page_name is not None else page_type
            name_input = self.layout.page.get_by_role("textbox", name="Name")
            name_input.fill(tab_name)
            name_input.press("ControlOrMeta+Enter")

        page_tab = self.layout.get_tab(tab_name)
        page_tab.wait_for(state="visible", timeout=15000)
        page_id = page_tab.inner_text().strip()

        if page_name is not None and not modal_was_open:
            self.layout.rename_tab(old_name=tab_name, new_name=page_name)
            page_id = page_name
            page_tab = self.layout.get_tab(page_name)

        pluto_labels = {
            "Log": ".pluto-log",
            "Line Plot": ".pluto-line-plot",
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

    def _create_and_initialize_page(
        self, page_type: PageType, page_name: str | None, page_class: type[T]
    ) -> T:
        """Helper to create a page and initialize it with proper locators.

        Args:
            page_type: Type of page to create
            page_name: Optional name for the page
            page_class: The page class to instantiate (Plot, Log, Schematic, etc.)

        Returns:
            Initialized page instance
        """
        pane, tab, actual_name = self.create_page(page_type, page_name)
        page = page_class(self.layout, self.client, actual_name, pane_locator=pane)
        page._initialize_from_project(tab, actual_name)
        return page

    def close_page(self, page_name: str) -> None:
        """Close a page by name. Ignores unsaved changes."""
        self.layout.close_tab(page_name)

    def get_item(self, name: str) -> Locator:
        """Get a project item locator from the resources toolbar.

        Note: Returns a Locator that can be waited on, even if the item isn't
        visible yet. Use exists() to check if an item is currently visible.

        Args:
            name: Name of the project

        Returns:
            Locator for the project item
        """
        return (
            self.layout.page.locator(f"div[id^='{self.ITEM_PREFIX}']")
            .filter(has_text=name)
            .first
        )

    def exists(self, name: str) -> bool:
        """Check if a project exists in the resources toolbar.

        Args:
            name: Name of the project to check

        Returns:
            True if project exists, False otherwise
        """
        self.layout.show_resource_toolbar("project")
        try:
            self.layout.page.locator(f"div[id^='{self.ITEM_PREFIX}']").first.wait_for(
                state="visible", timeout=5000
            )
        except PlaywrightTimeoutError:
            return False
        return self.tree.find_by_name(self.ITEM_PREFIX, name) is not None

    def wait_for_project_removed(self, name: str) -> None:
        """Wait for a project to be removed from the resources toolbar.

        Args:
            name: Name of the project to wait for removal
            timeout: Maximum time in milliseconds to wait
        """
        if self.on_splash():
            return
        self.layout.show_resource_toolbar("project")
        project_item = self.layout.page.locator(
            f"div[id^='{self.ITEM_PREFIX}']"
        ).filter(has_text=name)
        project_item.first.wait_for(state="hidden", timeout=5000)

    def expand_active(self) -> None:
        """Expand the active project in the resources toolbar to show its contents.

        Targets the active project by name. Concurrently-running tests each add
        their own project to the shared tree, so expanding whichever node sorts
        first (expand_root) would expand the wrong project and the test's pages
        would never be revealed. Falls back to expand_root if no project has
        been activated through this client yet.
        """
        self.layout.show_resource_toolbar("project")
        if self._active_project is not None:
            self.tree.expand_named(self.ITEM_PREFIX, self._active_project)
        else:
            self.tree.expand_root(self.ITEM_PREFIX)
        self.layout.page.locator(".pluto-tree__item").first.wait_for(
            state="visible", timeout=5000
        )

    def get_page(self, name: str) -> Locator:
        """Get a page item locator from the project resources toolbar.

        Matches the page label exactly; substring matches are ignored so a
        rename to a superstring (e.g. ``foo`` → ``foo_renamed``) does not
        leave the original locator pointing at the renamed item.

        Args:
            name: Name of the page (schematic, line plot, etc.)

        Returns:
            Locator for the page item
        """
        pattern = re.compile(rf"^\s*{re.escape(name)}\s*$")
        return (
            self.layout.page.locator(".pluto-tree__item").filter(has_text=pattern).first
        )

    def _scroll_to_page(self, name: str) -> bool:
        """Scroll the project tree to find a page that may be off-screen.

        The Pluto tree uses virtual scrolling, so items outside the viewport
        are not in the DOM. This method scrolls the tree container
        incrementally until the item appears or the end is reached.

        Returns:
            True if the item was found and scrolled into view.
        """
        page_item = self.get_page(name)
        try:
            page_item.wait_for(state="attached", timeout=1000)
            page_item.scroll_into_view_if_needed()
            return True
        except PlaywrightTimeoutError:
            pass
        container = self.layout.page.locator(".pluto-tree .pluto-list__items").first
        try:
            container.wait_for(state="attached", timeout=2000)
        except PlaywrightTimeoutError:
            return False
        prev_scroll = -1
        for _ in range(50):
            curr_scroll = container.evaluate(
                "el => ({ top: el.scrollTop, height: el.scrollHeight })"
            )
            if prev_scroll != -1 and curr_scroll["top"] == prev_scroll:
                break
            prev_scroll = curr_scroll["top"]
            container.evaluate("el => el.scrollTop += 200")
            self.layout.page.wait_for_timeout(100)
            try:
                page_item.wait_for(state="attached", timeout=300)
                page_item.scroll_into_view_if_needed()
                return True
            except PlaywrightTimeoutError:
                continue
        return False

    def page_exists(self, name: str) -> bool:
        """Check if a page (schematic, line plot, etc.) exists in the project."""
        self.expand_active()
        if self._scroll_to_page(name):
            page_item = self.get_page(name)
            try:
                page_item.wait_for(state="visible", timeout=5000)
                return True
            except PlaywrightTimeoutError:
                pass
        return False

    def wait_for_page_removed(self, name: str) -> None:
        """Wait for a page to be removed from the project."""
        page_item = self.get_page(name)
        page_item.wait_for(state="hidden", timeout=5000)

    def _find_page(self, name: str) -> Locator:
        """Find a page in the project tree, scrolling if needed.

        First tries a simple wait (handles most cases where the tree isn't
        crowded). Falls back to scrolling for virtual-scrolled trees.

        Raises:
            PlaywrightTimeoutError: If the page cannot be found.
        """
        self.expand_active()
        page_item = self.get_page(name)
        try:
            page_item.wait_for(state="visible", timeout=5000)
            return page_item
        except PlaywrightTimeoutError:
            pass
        if not self._scroll_to_page(name):
            raise PlaywrightTimeoutError(f"Page '{name}' not found in project tree")
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        return page_item

    def open_page(self, name: str) -> None:
        """Open a page by double-clicking it in the project resources toolbar.

        Args:
            name: Name of the page to open
        """
        page_item = self._find_page(name)
        page_item.dblclick()
        self.layout.close_left_toolbar()

    def drag_page_to_mosaic(self, name: str) -> None:
        """Drag a page from the project resources toolbar onto the mosaic.

        Args:
            name: Name of the page to drag
        """
        for attempt in range(2):
            self.layout.show_resource_toolbar("project")
            page_item = self._find_page(name)
            mosaic = self.layout.page.locator(".console-mosaic").first
            page_item.drag_to(mosaic)
            self.layout.close_left_toolbar()
            try:
                self.layout.wait_for_tab(name)
                return
            except PlaywrightTimeoutError:
                if attempt == 1:
                    raise

    def rename_page(self, old_name: str, new_name: str) -> None:
        """Rename a page via context menu in the project resources toolbar.

        Args:
            old_name: Current name of the page
            new_name: New name for the page
        """
        page_item = self._find_page(old_name)
        self.ctx_menu.action(page_item, "Rename")
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        self.get_page(new_name).wait_for(state="visible", timeout=5000)
        self.wait_for_page_removed(old_name)
        self.layout.close_left_toolbar()

    def delete_page(self, name: str) -> None:
        """Delete a page via context menu in the project resources toolbar.

        Args:
            name: Name of the page to delete
        """
        page_item = self._find_page(name)
        self.ctx_menu.action(page_item, "Delete")
        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        self.wait_for_page_removed(name)
        self.layout.close_left_toolbar()

    def delete_group(self, name: str, child_names: list[str] | None = None) -> None:
        """Delete a group via context menu.

        Args:
            name: Name of the group to delete.
            child_names: Names of child pages to delete first. Required because
                the Pluto tree uses flat rendering (children are siblings, not
                nested inside the group div), so they must be found by name.
        """
        if child_names:
            self.expand_active()
            group = self.tree.get_group(name)
            self.tree.expand(group)
            for child_name in child_names:
                page_item = self.get_page(child_name)
                page_item.wait_for(state="attached", timeout=5000)
                page_item.scroll_into_view_if_needed()
                self.ctx_menu.action(page_item, "Delete")
                delete_btn = self.layout.page.get_by_role(
                    "button", name="Delete", exact=True
                )
                delete_btn.wait_for(state="visible", timeout=5000)
                delete_btn.click(timeout=5000)
                self.wait_for_page_removed(child_name)
        self.expand_active()
        self.tree.delete_group(name)
        self.layout.close_left_toolbar()

    def delete_pages(self, page_names: list[str]) -> None:
        """Delete pages by walking the project tree depth-first.

        Expands groups as needed to find target pages, deletes each
        individually, and cleans up empty groups on the way back up.
        Tracks visited groups by DOM id to avoid infinite loops in the
        flat-rendered tree. Falls back to scrolling for pages that may
        be off-screen due to virtual scrolling.

        Args:
            page_names: Names of pages to delete.
        """
        if not page_names:
            return
        remaining = list(page_names)
        try:
            self.expand_active()
        except PlaywrightTimeoutError:
            self.layout.close_left_toolbar()
            return
        self._walk_and_delete(remaining, visited=set())
        # Scroll fallback for pages missed by the walk (virtual scrolling).
        for name in list(remaining):
            if not self._scroll_to_page(name):
                continue
            page_item = self.get_page(name)
            if not page_item.is_visible():
                continue
            self._delete_page_item(page_item, name, remaining)
        self.layout.close_left_toolbar()

    def _delete_page_item(
        self, page_item: Locator, name: str, remaining: list[str]
    ) -> None:
        self.ctx_menu.action(page_item, "Delete")
        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        delete_btn.wait_for(state="hidden", timeout=5000)
        remaining.remove(name)

    def _walk_and_delete(self, remaining: list[str], visited: set[str]) -> None:
        # Delete any visible target pages at this level.
        for name in list(remaining):
            page_item = self.get_page(name)
            if not page_item.is_visible():
                continue
            self._delete_page_item(page_item, name, remaining)

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

    def copy_page_link(self, name: str) -> str:
        """Copy link to a page via context menu.

        Args:
            name: Name of the page to copy link for

        Returns:
            The copied link from clipboard
        """
        page_item = self._find_page(name)
        self.ctx_menu.action(page_item, "Copy link")
        self.layout.close_left_toolbar()
        return self.layout.read_clipboard()

    def group_pages(self, *, names: list[str], group_name: str) -> None:
        """Group multiple pages into a folder via multi-select and context menu."""
        self.expand_active()
        items = []
        for n in names:
            self._scroll_to_page(n)
            items.append(self.get_page(n))
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
        self.tree.move_to_group(self.get_page(item_name), group_name)
        self.layout.close_left_toolbar()

    def export_page(self, name: str) -> dict[str, Any]:
        """Export a page via context menu.

        The file is saved to the tests/results directory.

        Args:
            name: Name of the page to export

        Returns:
            The exported JSON content as a dictionary
        """
        page_item = self._find_page(name)
        self.ctx_menu.open_on(page_item)
        self.layout.page.evaluate("delete window.showSaveFilePicker")

        with self.layout.page.expect_download(timeout=5000) as download_info:
            self.ctx_menu.click_option("Export")

        download = download_info.value
        save_path = resolve_results_path(f"{name}_export.json")
        download.save_as(save_path)
        self.layout.close_left_toolbar()

        with open(save_path, "r") as f:
            result: dict[str, Any] = json.load(f)
            return result

    def _install_directory_picker_mock(self) -> None:
        """Replace window.showDirectoryPicker with an in-memory capture."""
        self.layout.page.evaluate(_INSTALL_DIRECTORY_PICKER_MOCK_JS)

    def _drain_exported_files(self) -> dict[str, str]:
        """Read and clear files captured by the showDirectoryPicker mock."""
        files: dict[str, str] = self.layout.page.evaluate(
            "() => { const f = window.__synnaxExportedFiles || {};"
            " window.__synnaxExportedFiles = {}; return f; }"
        )
        return files

    def import_page(self, json_path: str, name: str) -> None:
        """Import a component via the real "Import component(s)" command palette flow.

        The import pipeline derives the tab name from the chosen filename (via
        trimFileName), so we copy ``json_path`` into a temp file named
        ``{name}.json`` to control the resulting tab name independently of
        the source fixture's filename.

        Waits for the page to appear in the project resource tree before
        returning. Non-schematic types (lineplot/log/table) only persist to
        the server via the mounted tab's debounced ``useSyncComponent`` hook;
        if the caller closes the tab before that debounce flushes, the save
        is cancelled and the tree row never appears. Waiting on the tree row
        guarantees the server-side resource exists.

        Args:
            json_path: Path to the JSON file to import.
            name: Display name for the imported page tab.
        """
        with tempfile.TemporaryDirectory() as tmp_dir:
            tmp_path = os.path.join(tmp_dir, f"{name}.json")
            shutil.copyfile(json_path, tmp_path)
            with self.layout.page.expect_file_chooser() as fc_info:
                self.layout.command_palette("Import component(s)")
            fc_info.value.set_files(tmp_path)
            self.layout.get_tab(name).wait_for(state="visible", timeout=10000)
            if not self.page_exists(name):
                raise AssertionError(
                    f"Imported page {name!r} did not appear in project tree"
                )
            self.layout.close_left_toolbar()

    def import_project_from_directory(self, directory_path: str) -> None:
        """Import a project via the real "Import a project" command flow.

        Opens the command palette, fulfills the resulting directory chooser
        with ``directory_path`` (Playwright walks it and uploads each file
        with its webkitRelativePath set), then waits for the project
        selector to display the directory's basename. The directory must
        contain ``LAYOUT.json`` and one ``{component_name}.json`` file per
        layout entry, matching the Console export format.
        """
        with self.layout.page.expect_file_chooser() as fc_info:
            self.layout.command_palette("Import a project")
        fc_info.value.set_files(directory_path)
        expected_name = os.path.basename(directory_path.rstrip(os.sep))
        self.layout.page.get_by_role("button").filter(has_text=expected_name).wait_for(
            state="visible", timeout=10000
        )

    def export_project(self, name: str) -> str:
        """Export a project via the real Export context menu action.

        Installs an in-memory mock of ``window.showDirectoryPicker`` so the
        real export code path runs unchanged but writes captured into a JS
        Map instead of touching the OS file picker. Waits for the production
        "Exported {name} to ..." success notification, which fires only
        after every file write completes, then drains the map and
        materializes the captured files into a real directory under the
        results dir.
        """
        self._install_directory_picker_mock()
        self.layout.show_resource_toolbar("project")
        project_item = self.get_item(name)
        project_item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(project_item, "Export")

        if not self.notifications.wait_for(f"Exported {name}"):
            raise AssertionError(
                f"Export of project {name!r} did not emit a success notification"
            )
        files = self._drain_exported_files()

        export_dir = resolve_results_path(f"{name}_export")
        if os.path.isdir(export_dir):
            shutil.rmtree(export_dir)
        os.makedirs(export_dir)
        for rel_path, contents in files.items():
            basename = os.path.basename(rel_path)
            with open(os.path.join(export_dir, basename), "w") as f:
                f.write(contents)

        self.notifications.close_all()
        self.layout.close_left_toolbar()
        return export_dir

    def snapshot_pages_to_active_range(self, names: list[str], range_name: str) -> None:
        """Snapshot multiple pages to the active range via context menu.

        Args:
            names: List of page names to snapshot
            range_name: Name of the active range (for menu text matching)
        """
        if not names:
            return
        self.expand_active()
        last = self.layout.ctrl_select_items(names, self.get_page)
        self.ctx_menu.action(last, f"Snapshot to {range_name}")
        self.layout.close_left_toolbar()

    def copy_page(self, name: str, new_name: str) -> None:
        """Make a copy of a page via context menu.

        Args:
            name: Name of the page to copy
            new_name: Name for the new copy
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(page_item, "Copy")

        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        self.get_page(new_name).wait_for(state="visible", timeout=5000)
        self.layout.close_left_toolbar()

    def copy_pages(self, names: list[str]) -> None:
        """Copy multiple pages via context menu.

        Note: When copying multiple pages, each gets a " (copy)" suffix automatically.

        Args:
            names: List of page names to copy
        """
        if not names:
            return
        self.expand_active()
        last = self.layout.ctrl_select_items(names, self.get_page)
        self.ctx_menu.action(last, "Copy")
        for name in names:
            copy_name = f"{name} (copy)"
            self.get_page(copy_name).wait_for(state="visible", timeout=5000)
        self.layout.close_left_toolbar()

    def create(self, name: str) -> bool:
        """Create a project via command palette.

        Args:
            name: Name of the project to create

        Returns:
            True if project was created, False if it already exists
        """
        if self.on_splash():
            self._create_from_splash(name)
            self._wait_for_app()
            return True

        if self.exists(name):
            return False

        if random.choice([True, False]):
            self.layout.command_palette("Create a project")
        else:
            self.layout.close_left_toolbar()
            selector = (
                self.layout.page.locator("button.pluto-dialog__trigger")
                .filter(has=self.layout.page.locator(".pluto-icon--project"))
                .first
            )
            selector.click(timeout=5000)
            self.layout.page.get_by_role("button", name="New", exact=True).click(
                timeout=5000
            )

        name_input = self.layout.page.locator("input[placeholder='Project Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)
        self.layout.page.get_by_role("button", name="Create", exact=True).click(
            timeout=5000
        )
        name_input.wait_for(state="hidden", timeout=5000)
        self.layout.show_resource_toolbar("project")
        self.get_item(name).wait_for(state="visible", timeout=5000)
        self.layout.close_left_toolbar()
        return True

    def select(self, name: str) -> None:
        """Select a project from the resources toolbar.

        Args:
            name: Name of the project to select
        """
        if self.on_splash():
            if self._select_from_splash(name):
                self._wait_for_app()
            return

        selector = (
            self.layout.page.locator("button.pluto-dialog__trigger")
            .filter(has=self.layout.page.locator(".pluto-icon--project"))
            .first
        )
        if name in selector.inner_text():
            self._active_project = name
            return
        self.layout.show_resource_toolbar("project")
        self.get_item(name).dblclick(timeout=5000)
        self.layout.page.get_by_role("button").filter(has_text=name).wait_for(
            state="visible", timeout=5000
        )
        self._active_project = name
        self.layout.close_left_toolbar()

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename a project via context menu.

        Args:
            old_name: Current name of the project
            new_name: New name for the project
        """
        self.layout.show_resource_toolbar("project")
        project = self.get_item(old_name)
        project.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(project, "Rename")
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        if self._active_project == old_name:
            self._active_project = new_name
        self.layout.close_left_toolbar()

    def delete(self, name: str) -> None:
        """Delete a project via context menu.

        Args:
            name: Name of the project to delete
        """
        if self.on_splash():
            # The resources toolbar only exists inside the app. Select the
            # project to delete so it becomes active and the app loads; the
            # delete then drops back to the Splash screen.
            if not self._select_from_splash(name):
                return
            self._wait_for_app()

        self.layout.show_resource_toolbar("project")

        project = self.get_item(name)
        project.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(project, "Delete")

        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        self.wait_for_project_removed(name)
        if self._active_project == name:
            self._active_project = None
        self.layout.close_left_toolbar()

    def select_bootstrap(self, name: str) -> None:
        """Select the per-test project ``name`` from the Splash screen after login.

        The project is provisioned server-side before the browser reaches the
        Splash screen, so this waits for it to appear in the Splash list,
        selects it, and returns once the main app palette is visible.

        Args:
            name: Name of the project to select
        """
        self.on_splash()
        item = (
            self.layout.page.locator(".console-project-splash__list")
            .get_by_text(name, exact=True)
            .first
        )
        item.wait_for(state="visible", timeout=10000)
        item.click(timeout=5000)
        self._wait_for_app()
        self._active_project = name

    def on_splash(self) -> bool:
        """Report whether the project Splash screen is showing.

        Deleting or clearing the active project drops the console to the Splash
        screen (an active project is required to use the app), so any helper
        that mutates project state may be entered from either screen. Waits for
        whichever of the Splash screen or the main app palette mounts first,
        then reports which one is visible.
        """
        self.layout.page.wait_for_selector(
            ".console-project-splash, .console-palette button",
            state="visible",
            timeout=15000,
        )
        return self.layout.page.locator(".console-project-splash").is_visible()

    def _wait_for_app(self) -> None:
        """Wait for the main app palette, the signal that a project is active."""
        self.layout.page.wait_for_selector(
            ".console-palette button", state="visible", timeout=15000
        )

    def _select_from_splash(self, name: str) -> bool:
        """Select ``name`` from the Splash project list.

        Returns False if the list is absent (no projects exist) or ``name`` is
        not in it, leaving the caller to create the project instead.
        """
        list_container = self.layout.page.locator(".console-project-splash__list")
        try:
            list_container.wait_for(state="visible", timeout=2000)
        except PlaywrightTimeoutError:
            return False
        item = list_container.get_by_text(name, exact=True)
        if item.count() == 0:
            return False
        item.first.click(timeout=5000)
        self._active_project = name
        return True

    def _create_from_splash(self, name: str) -> None:
        """Create ``name`` via the Splash New Project form."""
        name_input = self.layout.page.locator(
            ".console-project-splash__form input[placeholder='Project name']"
        )
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)
        self.layout.page.get_by_role("button", name="Create Project", exact=True).click(
            timeout=5000
        )
        self._active_project = name

    def open_plot(self, name: str) -> Plot:
        """Open a plot by double-clicking it in the project resources toolbar.

        Args:
            name: Name of the plot to open.

        Returns:
            Plot instance for the opened plot.
        """
        self.open_page(name)
        return Plot.from_open_page(self.layout, self.client, name)

    def drag_plot_to_mosaic(self, name: str) -> Plot:
        """Drag a plot from the project resources toolbar onto the mosaic.

        Args:
            name: Name of the plot to drag.

        Returns:
            Plot instance for the opened plot.
        """
        self.drag_page_to_mosaic(name)
        return Plot.from_open_page(self.layout, self.client, name)

    def open_from_search(self, page_class: type[T], name: str) -> T:
        """Open an existing page by searching its name in the command palette.

        Args:
            page_class: The page class to instantiate (Plot, Log, Table, Schematic, etc.)
            name: Name of the page to search for (could be page name or channel name)

        Returns:
            Instance of the specified page class
        """
        self.layout.search_palette(name)

        pane = self.layout.page.locator(page_class.pluto_label)
        pane.first.wait_for(state="visible", timeout=5000)

        active_tab = (
            self.layout.page.locator(".pluto-tabs-selector")
            .locator("div")
            .filter(has=self.layout.page.locator("[aria-label='pluto-tabs__close']"))
            .last
        )
        actual_name = active_tab.inner_text().strip()

        return page_class(
            self.layout, self.client, actual_name, pane_locator=pane.first
        )

    def create_plot(self, name: str) -> Plot:
        """Create a new plot page in the UI and return a wrapper.

        This method:
        1. Creates the page in the Console UI via Playwright (create_page)
        2. Returns a Python Plot object that wraps the UI page

        Args:
            name: Name for the new plot

        Returns:
            Plot instance wrapping the created UI page
        """
        return self._create_and_initialize_page("Line Plot", name, Plot)

    def create_log(self, name: str) -> Log:
        """Create a new log page in the UI and return a wrapper.

        This method:
        1. Creates the page in the Console UI via Playwright (create_page)
        2. Returns a Python Log object that wraps the UI page

        Args:
            name: Name for the new log

        Returns:
            Log instance wrapping the created UI page
        """
        return self._create_and_initialize_page("Log", name, Log)

    def create_schematic(self, name: str) -> Schematic:
        """Create a new schematic page in the UI and return a wrapper.

        This method:
        1. Creates the page in the Console UI via Playwright (create_page)
        2. Returns a Python Schematic object that wraps the UI page

        Args:
            name: Name for the new schematic

        Returns:
            Schematic instance wrapping the created UI page
        """
        return self._create_and_initialize_page("Schematic", name, Schematic)

    def create_table(self, name: str) -> Table:
        """Create a new table page in the UI and return a wrapper.

        This method:
        1. Creates the page in the Console UI via Playwright (create_page)
        2. Returns a Python Table object that wraps the UI page

        Args:
            name: Name for the new table

        Returns:
            Table instance wrapping the created UI page
        """
        return self._create_and_initialize_page("Table", name, Table)

    def open_plot_from_click(self, channel_name: str, channels: ChannelClient) -> Plot:
        """Open a plot by double-clicking a channel in the channels sidebar.

        Args:
            channel_name: Name of the channel to double-click
            channels: ChannelClient for showing/hiding channels sidebar

        Returns:
            Plot instance for the opened plot
        """
        channels.show_channels()

        channel_item = self.tree.find_by_name("channel:", channel_name)
        if channel_item is None:
            raise ValueError(f"Channel '{channel_name}' not found")
        channel_item.wait_for(state="visible", timeout=5000)
        channel_item.dblclick()

        plot_pane = self.layout.page.locator(".pluto-line-plot")
        plot_pane.first.wait_for(state="visible", timeout=5000)

        tabs = self.layout.page.locator(".pluto-tabs-selector div").filter(
            has=self.layout.page.locator("[aria-label='pluto-tabs__close']")
        )
        tab_count = tabs.count()
        actual_tab_name = "Line Plot"
        if tab_count > 0:
            last_tab = tabs.nth(tab_count - 1)
            actual_tab_name = last_tab.inner_text().strip()

        plot = Plot.from_open_page(self.layout, self.client, actual_tab_name)

        channels.hide_channels()
        return plot

    def open_log(self, name: str) -> Log:
        """Open a log by double-clicking it in the project resources toolbar.

        Args:
            name: Name of the log to open.

        Returns:
            Log instance for the opened log.
        """
        self.open_page(name)
        return Log.from_open_page(self.layout, self.client, name)

    def drag_log_to_mosaic(self, name: str) -> Log:
        """Drag a log from the project resources toolbar onto the mosaic.

        Args:
            name: Name of the log to drag.

        Returns:
            Log instance for the opened log.
        """
        from console.log import Log

        self.drag_page_to_mosaic(name)
        return Log.from_open_page(self.layout, self.client, name)

    def open_schematic(self, name: str) -> Schematic:
        """Open a schematic by double-clicking it in the project resources toolbar.

        Args:
            name: Name of the schematic to open.

        Returns:
            Schematic instance for the opened schematic.
        """
        self.open_page(name)
        return Schematic.from_open_page(self.layout, self.client, name)

    def drag_schematic_to_mosaic(self, name: str) -> Schematic:
        """Drag a schematic from the project resources toolbar onto the mosaic.

        Args:
            name: Name of the schematic to drag.

        Returns:
            Schematic instance for the opened schematic.
        """
        self.drag_page_to_mosaic(name)
        return Schematic.from_open_page(self.layout, self.client, name)

    def open_table(self, name: str) -> Table:
        """Open a table by double-clicking it in the project resources toolbar.

        Args:
            name: Name of the table to open.

        Returns:
            Table instance for the opened table.
        """
        self.open_page(name)
        return Table.from_open_page(self.layout, self.client, name)

    def drag_table_to_mosaic(self, name: str) -> Table:
        """Drag a table from the project resources toolbar onto the mosaic.

        Args:
            name: Name of the table to drag.

        Returns:
            Table instance for the opened table.
        """
        self.drag_page_to_mosaic(name)
        return Table.from_open_page(self.layout, self.client, name)

    @overload
    def create_task(
        self, task_type: Literal["NI Analog Read Task"], name: str
    ) -> AnalogRead: ...

    @overload
    def create_task(
        self, task_type: Literal["NI Analog Write Task"], name: str
    ) -> AnalogWrite: ...

    @overload
    def create_task(
        self, task_type: Literal["NI Counter Read Task"], name: str
    ) -> CounterRead: ...

    def create_task(self, task_type: PageType, name: str) -> TaskPage:
        """Create a new task page in the UI and return a wrapper.

        This method:
        1. Creates the task page in the Console UI via Playwright (create_page)
        2. Returns a Python TaskPage subclass instance that wraps the UI page

        Currently supports NI tasks:
        - NI Analog Read Task
        - NI Analog Write Task
        - NI Counter Read Task

        Designed to be extensible for future task types:
        - NI Digital Read Task
        - NI Digital Write Task
        - LabJack Read Task
        - LabJack Write Task
        - OPC UA Read Task
        - OPC UA Write Task
        - Modbus Read Task
        - Modbus Write Task

        Args:
            task_type: Type of task to create (must match PageType)
            name: Name for the new task

        Returns:
            TaskPage subclass instance wrapping the created UI page

        Raises:
            ValueError: If task_type is not supported
        """
        # Map task types to their corresponding classes
        task_class_map: dict[str, type[TaskPage]] = {
            "NI Analog Read Task": AnalogRead,
            "NI Analog Write Task": AnalogWrite,
            "NI Counter Read Task": CounterRead,
            # "NI Digital Read Task": DigitalRead,
            # "NI Digital Write Task": DigitalWrite,
            # "LabJack Read Task": LabJackRead,
            # "LabJack Write Task": LabJackWrite,
            # "OPC UA Read Task": OPCUARead,
            # "OPC UA Write Task": OPCUAWrite,
        }

        if task_type not in task_class_map:
            raise ValueError(
                f"Unsupported task type: {task_type}. "
                f"Supported types: {list(task_class_map.keys())}"
            )

        task_class = task_class_map[task_type]
        return self._create_and_initialize_page(task_type, name, task_class)
