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
from typing import Any, Literal, TypeVar, overload

import synnax as sy
from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

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
from framework.utils import get_results_path

__all__ = ["WorkspaceClient", "PageType"]

T = TypeVar("T", bound="ConsolePage")

# SY-3670 — Shared JS snippet that walks the React fiber tree from #root
# to find the Redux store. Used by import_page and export_workspace as a
# browser-mode workaround for features that normally rely on Tauri APIs.
_FIND_REDUX_STORE_JS = """
const rootEl = document.getElementById('root');
if (!rootEl) throw new Error('Root element not found');
const containerKey = Object.keys(rootEl).find(
    k => k.startsWith('__reactContainer$')
);
if (!containerKey) throw new Error('React container not found');
let store = null;
const stack = [rootEl[containerKey]];
while (stack.length > 0) {
    const fiber = stack.pop();
    if (!fiber) continue;
    if (fiber.memoizedProps?.store?.dispatch) {
        store = fiber.memoizedProps.store;
        break;
    }
    if (fiber.child) stack.push(fiber.child);
    if (fiber.sibling) stack.push(fiber.sibling);
}
if (!store) throw new Error('Redux store not found');
"""


class WorkspaceClient:
    """Workspace management for Console UI automation."""

    ITEM_PREFIX = "workspace:"

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
        add_btn.click(force=True)

        self.layout.page.locator(".console-layout-selector__frame").wait_for(
            state="visible", timeout=15000
        )
        self.layout.page.get_by_role("button", name=page_type).first.click()

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
        page._initialize_from_workspace(tab, actual_name)
        return page

    def close_page(self, page_name: str) -> None:
        """Close a page by name. Ignores unsaved changes."""
        self.layout.close_tab(page_name)

    def get_item(self, name: str) -> Locator:
        """Get a workspace item locator from the resources toolbar.

        Note: Returns a Locator that can be waited on, even if the item isn't
        visible yet. Use exists() to check if an item is currently visible.

        Args:
            name: Name of the workspace

        Returns:
            Locator for the workspace item
        """
        return (
            self.layout.page.locator(f"div[id^='{self.ITEM_PREFIX}']")
            .filter(has_text=name)
            .first
        )

    def exists(self, name: str) -> bool:
        """Check if a workspace exists in the resources toolbar.

        Args:
            name: Name of the workspace to check

        Returns:
            True if workspace exists, False otherwise
        """
        self.layout.show_resource_toolbar("workspace")
        try:
            self.layout.page.locator(f"div[id^='{self.ITEM_PREFIX}']").first.wait_for(
                state="visible", timeout=5000
            )
        except PlaywrightTimeoutError:
            return False
        return self.tree.find_by_name(self.ITEM_PREFIX, name) is not None

    def wait_for_workspace_removed(self, name: str) -> None:
        """Wait for a workspace to be removed from the resources toolbar.

        Args:
            name: Name of the workspace to wait for removal
            timeout: Maximum time in milliseconds to wait
        """
        self.layout.show_resource_toolbar("workspace")
        workspace_item = self.layout.page.locator(
            f"div[id^='{self.ITEM_PREFIX}']"
        ).filter(has_text=name)
        workspace_item.first.wait_for(state="hidden", timeout=5000)

    def expand_active(self) -> None:
        """Expand the active workspace in the resources toolbar to show its contents."""
        self.layout.show_resource_toolbar("workspace")
        self.layout.page.locator(f"div[id^='{self.ITEM_PREFIX}']").first.wait_for(
            state="visible", timeout=5000
        )
        workspace_items = self.tree.find_by_prefix(self.ITEM_PREFIX)
        if not workspace_items:
            return
        workspace_item = workspace_items[0]
        caret = workspace_item.locator(".pluto--location-bottom")
        try:
            caret.wait_for(state="visible", timeout=500)
            return
        except PlaywrightTimeoutError:
            pass
        workspace_item.click()
        caret.wait_for(state="visible", timeout=5000)

    def get_page(self, name: str) -> Locator:
        """Get a page item locator from the workspace resources toolbar.

        Args:
            name: Name of the page (schematic, line plot, etc.)

        Returns:
            Locator for the page item
        """
        return self.layout.page.locator(".pluto-tree__item").filter(has_text=name).first

    def page_exists(self, name: str) -> bool:
        """Check if a page (schematic, line plot, etc.) exists in the workspace."""
        self.expand_active()
        try:
            self.get_page(name).wait_for(state="visible", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

    def wait_for_page_removed(self, name: str) -> None:
        """Wait for a page to be removed from the workspace."""
        page_item = self.get_page(name)
        page_item.wait_for(state="hidden", timeout=5000)

    def open_page(self, name: str) -> None:
        """Open a page by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the page to open
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.dblclick()
        self.layout.close_left_toolbar()

    def drag_page_to_mosaic(self, name: str) -> None:
        """Drag a page from the workspace resources toolbar onto the mosaic.

        Args:
            name: Name of the page to drag
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        mosaic = self.layout.page.locator(".console-mosaic").first
        page_item.drag_to(mosaic)
        self.layout.close_left_toolbar()

    def rename_page(self, old_name: str, new_name: str) -> None:
        """Rename a page via context menu in the workspace resources toolbar.

        Args:
            old_name: Current name of the page
            new_name: New name for the page
        """
        self.expand_active()
        page_item = self.get_page(old_name)
        page_item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(page_item, "Rename")
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        self.get_page(new_name).wait_for(state="visible", timeout=5000)
        self.wait_for_page_removed(old_name)
        self.layout.close_left_toolbar()

    def delete_page(self, name: str) -> None:
        """Delete a page via context menu in the workspace resources toolbar.

        Args:
            name: Name of the page to delete
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(page_item, "Delete")
        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        self.wait_for_page_removed(name)
        self.layout.close_left_toolbar()

    def delete_group(self, name: str) -> None:
        """Delete a group via context menu.

        Groups are deleted immediately without a confirmation dialog (unlike pages).
        The context menu shows "Delete" for collapsed groups and "Ungroup" for expanded
        groups with visible children.

        Args:
            name: Name of the group to delete
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        try:
            self.ctx_menu.action(page_item, "Delete")
        except PlaywrightTimeoutError:
            self.ctx_menu.click_option("Ungroup")
        self.layout.close_left_toolbar()

    def delete_pages(self, names: list[str]) -> None:
        """Delete multiple pages via multi-select and context menu.

        Args:
            names: List of page names to delete
        """
        if not names:
            return

        self.expand_active()

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            page_item.click(modifiers=["ControlOrMeta"])

        last_item = first_item if len(names) == 1 else self.get_page(names[-1])
        self.ctx_menu.action(last_item, "Delete")
        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        for name in names:
            self.wait_for_page_removed(name)
        self.layout.close_left_toolbar()

    def copy_page_link(self, name: str) -> str:
        """Copy link to a page via context menu.

        Args:
            name: Name of the page to copy link for

        Returns:
            The copied link from clipboard
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(page_item, "Copy link")
        self.layout.close_left_toolbar()
        return self.layout.read_clipboard()

    def group_pages(self, *, names: list[str], group_name: str) -> None:
        """Group multiple pages into a folder via multi-select and context menu.

        Args:
            names: List of page names to group
            group_name: Name for the new group/folder
        """
        if not names:
            return

        self.expand_active()

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            page_item.click(modifiers=["ControlOrMeta"])

        last_item = first_item if len(names) == 1 else self.get_page(names[-1])
        self.ctx_menu.action(last_item, "Group selection")
        self.layout.select_all_and_type(group_name)
        self.layout.press_enter()
        self.layout.close_left_toolbar()

    def export_page(self, name: str) -> dict[str, Any]:
        """Export a page via context menu.

        The file is saved to the tests/results directory.

        Args:
            name: Name of the page to export

        Returns:
            The exported JSON content as a dictionary
        """
        self.expand_active()
        page_item = self.get_page(name)
        try:
            page_item.wait_for(state="visible", timeout=5000)
        except PlaywrightTimeoutError as e:
            all_items = self.layout.page.locator(".pluto-tree__item").all()
            item_texts = [
                item.text_content() for item in all_items if item.is_visible()
            ]
            raise PlaywrightTimeoutError(
                f"Page '{name}' not found. Available items: {item_texts}"
            ) from e
        self.ctx_menu.open_on(page_item)
        self.layout.page.evaluate("delete window.showSaveFilePicker")

        with self.layout.page.expect_download(timeout=5000) as download_info:
            self.ctx_menu.click_option("Export")

        download = download_info.value
        save_path = get_results_path(f"{name}_export.json")
        download.save_as(save_path)
        self.layout.close_left_toolbar()

        with open(save_path, "r") as f:
            result: dict[str, Any] = json.load(f)
            return result

    def _evaluate_with_redux(self, js_body: str, args: Any = None) -> Any:
        """Execute JS with the Redux store available as ``store``.

        # SY-3670 — Uses React fiber walking to locate the Redux store
        # from the React root. This is a browser-mode workaround for
        # features that normally rely on Tauri APIs.

        Args:
            js_body: JS code to execute. ``store`` and ``args`` are in scope.
            args: Optional arguments forwarded to the JS function.
        """
        js = "(args) => {" + _FIND_REDUX_STORE_JS + js_body + "}"
        return self.layout.page.evaluate(js, args)

    def import_page(self, json_path: str, name: str) -> None:
        """Import a page from a JSON file via direct JS injection.

        Since the console uses Tauri's native file dialog for imports
        (unavailable in browser mode), this method bypasses the dialog
        by reading the JSON file and dispatching Redux actions directly.

        # SY-3670 will address this issue.

        Args:
            json_path: Path to the JSON file to import.
            name: Display name for the imported page tab.
        """
        with open(json_path, "r") as f:
            data: dict[str, Any] = json.load(f)

        resource_type = data.get("type")
        if resource_type is None:
            raise ValueError(f"JSON file missing 'type' field: {json_path}")

        type_config: dict[str, dict[str, str]] = {
            "lineplot": {"slice": "line", "icon": "Visualize"},
            "schematic": {"slice": "schematic", "icon": "Schematic"},
            "log": {"slice": "log", "icon": "Log"},
            "table": {"slice": "table", "icon": "Table"},
        }

        config = type_config.get(resource_type)
        if config is None:
            raise ValueError(f"Unsupported resource type: {resource_type}")

        self._evaluate_with_redux(
            """
            const [data, name, sliceName, icon] = args;
            const key = crypto.randomUUID();
            store.dispatch({
                type: sliceName + '/create',
                payload: { ...data, key }
            });
            store.dispatch({
                type: 'layout/place',
                payload: {
                    key, name,
                    location: 'mosaic',
                    type: data.type,
                    icon,
                    windowKey: 'main',
                }
            });
            """,
            [data, name, config["slice"], config["icon"]],
        )

        self.layout.get_tab(name).wait_for(state="visible", timeout=10000)

    def import_workspace(self, name: str, data: dict[str, Any]) -> None:
        """Import a workspace via command palette with JS injection fallback.

        Triggers "Import a workspace" from the command palette. Since the
        console uses Tauri's native directory dialog (unavailable in browser
        mode), the actual import falls back to dispatching Redux actions
        via JS injection.

        # SY-3670 will address the browser-mode limitation.

        Args:
            name: Name for the imported workspace.
            data: Export data dict with 'layout' and 'components' keys
                  (as returned by export_workspace).
        """
        self.layout.command_palette("Import a workspace")

        sliceMap: dict[str, str] = {
            "lineplot": "line",
            "schematic": "schematic",
            "log": "log",
            "table": "table",
        }

        self._evaluate_with_redux(
            """
            const [name, layoutData, components, sliceMap] = args;
            const wsKey = crypto.randomUUID();
            store.dispatch({
                type: 'workspace/setActive',
                payload: { key: wsKey, name, layout: layoutData },
            });
            store.dispatch({
                type: 'layout/setWorkspace',
                payload: { slice: layoutData, keepNav: false },
            });
            for (const [key, component] of Object.entries(components)) {
                const sliceName = sliceMap[component.type];
                if (!sliceName) continue;
                store.dispatch({
                    type: sliceName + '/create',
                    payload: { ...component, key },
                });
            }
            """,
            [name, data["layout"], data["components"], sliceMap],
        )

        self.layout.page.get_by_role("button").filter(has_text=name).wait_for(
            state="visible", timeout=10000
        )

    def export_workspace(self, name: str) -> dict[str, Any]:
        """Export a workspace via context menu with JS injection fallback.

        Opens the context menu on the workspace and clicks Export. Since the
        console uses Tauri's native directory dialog and file system APIs
        (unavailable in browser mode), the actual data extraction falls back
        to reading Redux state via JS injection.

        # SY-3670 will address the browser-mode limitation.

        Args:
            name: Name of the workspace to export.

        Returns:
            Dict with 'layout' (the layout state) and 'components'
            (dict of component key -> exported state with type field).
        """
        self.layout.show_resource_toolbar("workspace")
        workspace_item = self.get_item(name)
        workspace_item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(workspace_item, "Export")
        self.notifications.close_all()

        result: dict[str, Any] = self._evaluate_with_redux("""
            const state = store.getState();
            const layoutState = state.layout;
            const sliceMap = {
                lineplot: { slice: 'line', collection: 'plots' },
                schematic: { slice: 'schematic', collection: 'schematics' },
                log: { slice: 'log', collection: 'logs' },
                table: { slice: 'table', collection: 'tables' },
            };
            const components = {};
            const layouts = {};
            for (const [key, layout] of Object.entries(layoutState.layouts)) {
                if (layout.excludeFromWorkspace || layout.location === 'modal')
                    continue;
                layouts[key] = layout;
                const mapping = sliceMap[layout.type];
                if (!mapping) continue;
                const cs = state[mapping.slice]?.[mapping.collection]?.[key];
                if (cs) components[key] = { ...cs, type: layout.type };
            }
            return { layout: { ...layoutState, layouts }, components };
            """)

        save_path = get_results_path(f"{name}_export.json")
        with open(save_path, "w") as f:
            json.dump(result, f, indent=2)

        self.layout.close_left_toolbar()
        return result

    def snapshot_page_to_active_range(self, name: str, range_name: str) -> None:
        """Snapshot a page to the active range via context menu.

        Args:
            name: Name of the page to snapshot
            range_name: Name of the active range (for menu text matching)
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(page_item, f"Snapshot to {range_name}")
        self.layout.close_left_toolbar()

    def snapshot_pages_to_active_range(self, names: list[str], range_name: str) -> None:
        """Snapshot multiple pages to the active range via context menu.

        Args:
            names: List of page names to snapshot
            range_name: Name of the active range (for menu text matching)
        """
        if not names:
            return

        self.expand_active()

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            page_item.click(modifiers=["ControlOrMeta"])

        last_item = first_item if len(names) == 1 else self.get_page(names[-1])
        self.ctx_menu.action(last_item, f"Snapshot to {range_name}")
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

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            page_item.click(modifiers=["ControlOrMeta"])

        # For single item, reuse first_item; otherwise get the last item
        last_item = first_item if len(names) == 1 else self.get_page(names[-1])
        self.ctx_menu.action(last_item, "Copy")

        for name in names:
            copy_name = f"{name} (copy)"
            self.get_page(copy_name).wait_for(state="visible", timeout=5000)

        self.layout.close_left_toolbar()

    def create(self, name: str) -> bool:
        """Create a workspace via command palette.

        Args:
            name: Name of the workspace to create

        Returns:
            True if workspace was created, False if it already exists
        """
        if self.exists(name):
            return False

        if random.choice([True, False]):
            self.layout.command_palette("Create a workspace")
        else:
            self.layout.close_left_toolbar()
            selector = (
                self.layout.page.locator("button.pluto-dialog__trigger")
                .filter(has=self.layout.page.locator(".pluto-icon--workspace"))
                .first
            )
            selector.click(timeout=5000)
            self.layout.page.get_by_role("button", name="New", exact=True).click(
                timeout=5000
            )

        name_input = self.layout.page.locator("input[placeholder='Workspace Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)
        self.layout.page.get_by_role("button", name="Create", exact=True).click(
            timeout=5000
        )
        name_input.wait_for(state="hidden", timeout=5000)
        self.layout.show_resource_toolbar("workspace")
        self.get_item(name).wait_for(state="visible", timeout=5000)
        self.layout.close_left_toolbar()
        return True

    def select(self, name: str) -> None:
        """Select a workspace from the resources toolbar.

        Args:
            name: Name of the workspace to select
        """
        selector = (
            self.layout.page.locator("button.pluto-dialog__trigger")
            .filter(has=self.layout.page.locator(".pluto-icon--workspace"))
            .first
        )
        if name in selector.inner_text():
            return
        self.layout.show_resource_toolbar("workspace")
        self.get_item(name).dblclick(timeout=5000)
        self.layout.page.get_by_role("button").filter(has_text=name).wait_for(
            state="visible", timeout=5000
        )
        self.layout.close_left_toolbar()

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename a workspace via context menu.

        Args:
            old_name: Current name of the workspace
            new_name: New name for the workspace
        """
        self.layout.show_resource_toolbar("workspace")
        workspace = self.get_item(old_name)
        workspace.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(workspace, "Rename")
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        self.layout.close_left_toolbar()

    def delete(self, name: str) -> None:
        """Delete a workspace via context menu.

        Args:
            name: Name of the workspace to delete
        """
        self.layout.show_resource_toolbar("workspace")

        workspace = self.get_item(name)
        workspace.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(workspace, "Delete")

        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        self.wait_for_workspace_removed(name)
        self.layout.close_left_toolbar()

    def ensure_selected(self, name: str) -> None:
        """Create a workspace if it doesn't exist and select it.

        Args:
            name: Name of the workspace to ensure is selected
        """
        selector = self.layout.page.locator("button.pluto-dialog__trigger").filter(
            has=self.layout.page.locator(".pluto-icon--workspace")
        )
        if name in selector.inner_text(timeout=5000):
            return

        self.create(name)
        self.select(name)

    def open_plot(self, name: str) -> Plot:
        """Open a plot by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the plot to open.

        Returns:
            Plot instance for the opened plot.
        """
        self.open_page(name)
        return Plot.from_open_page(self.layout, self.client, name)

    def drag_plot_to_mosaic(self, name: str) -> Plot:
        """Drag a plot from the workspace resources toolbar onto the mosaic.

        Args:
            name: Name of the plot to drag.

        Returns:
            Plot instance for the opened plot.
        """
        self.drag_page_to_mosaic(name)
        return Plot.from_open_page(self.layout, self.client, name)

    def open_from_search(self, page_class: type[ConsolePage], name: str) -> ConsolePage:
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
        """Open a log by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the log to open.

        Returns:
            Log instance for the opened log.
        """
        self.open_page(name)
        return Log.from_open_page(self.layout, self.client, name)

    def drag_log_to_mosaic(self, name: str) -> Log:
        """Drag a log from the workspace resources toolbar onto the mosaic.

        Args:
            name: Name of the log to drag.

        Returns:
            Log instance for the opened log.
        """
        from console.log import Log

        self.drag_page_to_mosaic(name)
        return Log.from_open_page(self.layout, self.client, name)

    def open_schematic(self, name: str) -> Schematic:
        """Open a schematic by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the schematic to open.

        Returns:
            Schematic instance for the opened schematic.
        """
        self.open_page(name)
        return Schematic.from_open_page(self.layout, self.client, name)

    def drag_schematic_to_mosaic(self, name: str) -> Schematic:
        """Drag a schematic from the workspace resources toolbar onto the mosaic.

        Args:
            name: Name of the schematic to drag.

        Returns:
            Schematic instance for the opened schematic.
        """
        self.drag_page_to_mosaic(name)
        return Schematic.from_open_page(self.layout, self.client, name)

    def open_table(self, name: str) -> Table:
        """Open a table by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the table to open.

        Returns:
            Table instance for the opened table.
        """
        self.open_page(name)
        return Table.from_open_page(self.layout, self.client, name)

    def drag_table_to_mosaic(self, name: str) -> Table:
        """Drag a table from the workspace resources toolbar onto the mosaic.

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
