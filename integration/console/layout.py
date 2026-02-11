#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import re
from collections.abc import Generator
from contextlib import contextmanager
from typing import Literal

import synnax as sy
from playwright.sync_api import Locator, Page
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.context_menu import ContextMenu
from console.notifications import NotificationsClient

AriaRole = Literal[
    "alert",
    "alertdialog",
    "application",
    "article",
    "banner",
    "blockquote",
    "button",
    "caption",
    "cell",
    "checkbox",
    "code",
    "columnheader",
    "combobox",
    "complementary",
    "contentinfo",
    "definition",
    "deletion",
    "dialog",
    "directory",
    "document",
    "emphasis",
    "feed",
    "figure",
    "form",
    "generic",
    "grid",
    "gridcell",
    "group",
    "heading",
    "img",
    "insertion",
    "link",
    "list",
    "listbox",
    "listitem",
    "log",
    "main",
    "marquee",
    "math",
    "menu",
    "menubar",
    "menuitem",
    "menuitemcheckbox",
    "menuitemradio",
    "meter",
    "navigation",
    "none",
    "note",
    "option",
    "paragraph",
    "presentation",
    "progressbar",
    "radio",
    "radiogroup",
    "region",
    "row",
    "rowgroup",
    "rowheader",
    "scrollbar",
    "search",
    "searchbox",
    "separator",
    "slider",
    "spinbutton",
    "status",
    "strong",
    "subscript",
    "superscript",
    "switch",
    "tab",
    "table",
    "tablist",
    "tabpanel",
    "term",
    "textbox",
    "time",
    "timer",
    "toolbar",
    "tooltip",
    "tree",
    "treegrid",
    "treeitem",
]


class LayoutClient:
    """
    Layout and UI primitives for Console automation.

    This is the PRIMARY abstraction layer over Playwright. All UI operations
    (command palette, forms, keyboard, navigation) should go through this client.
    """

    MODAL_SELECTOR = "div.pluto-dialog__dialog.pluto--modal.pluto--visible"

    def __init__(self, page: Page):
        self.page = page
        self.ctx_menu = ContextMenu(self.page)  # For internal tab operations
        self.notifications = NotificationsClient(self.page)

    def command_palette(self, command: str, retries: int = 3) -> None:
        """Execute a command via the command palette."""
        self._palette(
            query=command,
            input_text=f">{command}",
            empty_message="No commands found",
            error_prefix="Command palette",
            retries=retries,
        )

    def search_palette(self, query: str, retries: int = 3) -> None:
        """Search for a resource via the command palette (without > prefix)."""
        self._palette(
            query=query,
            input_text=query,
            empty_message="No results found",
            error_prefix="Search palette",
            retries=retries,
        )

    def _palette(
        self,
        *,
        query: str,
        input_text: str,
        empty_message: str,
        error_prefix: str,
        retries: int,
    ) -> None:
        """Internal palette implementation used by command_palette and search_palette."""
        for attempt in range(retries):
            palette_btn = self.page.locator(".console-palette button").first
            palette_btn.wait_for(state="visible", timeout=5000)
            palette_btn.click(timeout=5000)

            palette_input = self.page.locator(
                ".console-palette__input input[role='textbox']"
            )
            palette_input.wait_for(state="visible", timeout=5000)
            palette_input.press("ControlOrMeta+a")
            palette_input.type(input_text, timeout=5000)

            try:
                self.page.locator(
                    ".console-palette__list .pluto-list__item"
                ).first.wait_for(state="attached", timeout=10000)
            except PlaywrightTimeoutError:
                no_results = self.page.get_by_text(empty_message).is_visible()
                if no_results and attempt < retries - 1:
                    self.page.keyboard.press("Escape")
                    sy.sleep(2)
                    continue

                input_value = palette_input.input_value()
                palette_open = self.page.locator(
                    ".console-palette__content"
                ).is_visible()
                list_container = self.page.locator(".console-palette__list")
                list_visible = list_container.is_visible()
                list_html = ""
                try:
                    list_html = list_container.inner_html(timeout=5000)[:1000]
                except Exception:
                    list_html = "<failed to get>"
                raise RuntimeError(
                    f"{error_prefix} list items not appearing. "
                    f"Input: '{input_value}'. "
                    f"Palette open: {palette_open}. "
                    f"List visible: {list_visible}. "
                    f"List HTML: {list_html}"
                )

            target_result = (
                self.page.locator(".console-palette__list .pluto-list__item")
                .filter(has_text=query)
                .first
            )
            try:
                target_result.wait_for(state="visible", timeout=5000)
            except PlaywrightTimeoutError:
                input_value = palette_input.input_value()
                list_items = self.page.locator(
                    ".console-palette__list .pluto-list__virtualizer > div"
                ).all()
                options = []
                for item in list_items:
                    try:
                        options.append(item.inner_text(timeout=5000))
                    except PlaywrightTimeoutError:
                        options.append("<failed to get text>")
                raise RuntimeError(
                    f"{error_prefix}: Could not find '{query}'. "
                    f"Input value: '{input_value}'. "
                    f"Available options: {options}"
                )
            target_result.click(timeout=5000)
            return

    def is_modal_open(self) -> bool:
        """Check if a modal dialog is currently open."""
        return self.page.locator(self.MODAL_SELECTOR).count() > 0

    def check_for_errors(self) -> bool:
        """Check notifications for errors.

        Returns:
            True if errors were found, False otherwise.
        """
        for notification in self.notifications.check():
            message = notification.get("message", "")
            if "Failed" in message or "Error" in message:
                self.notifications.close(0)
                return True
        return False

    def show_resource_toolbar(self, resource: str) -> None:
        """Show a resource toolbar by clicking its icon in the sidebar."""
        nav_drawer = self.page.locator(
            ".console-nav__drawer.pluto--visible:not(.pluto--location-bottom)"
        )
        items = self.page.locator(f"div[id^='{resource}:']")
        drawer_count = nav_drawer.count()
        items_count = items.count()
        items_visible = items.first.is_visible() if items_count > 0 else False
        if drawer_count > 0 and items_count > 0 and items_visible:
            return

        button = self.page.locator("button.console-main-nav__item").filter(
            has=self.page.locator(f"svg.pluto-icon--{resource}")
        )
        btn_class = button.first.get_attribute("class") or ""
        if "selected" not in btn_class:
            button.click(timeout=5000)
        nav_drawer.wait_for(state="visible", timeout=5000)

    def close_left_toolbar(self) -> None:
        """Close any open side nav drawer (left/right, not bottom visualization toolbar)."""
        nav_drawer = self.page.locator(
            ".console-nav__drawer.pluto--visible:not(.pluto--location-bottom)"
        )
        if nav_drawer.count() == 0 or not nav_drawer.first.is_visible():
            return
        active_nav_btn = self.page.locator(
            "button.console-main-nav__item.pluto--selected"
        ).first
        if active_nav_btn.count() == 0:
            return
        drawer_class = nav_drawer.first.get_attribute("class") or ""
        is_expanded = "pluto--expanded" in drawer_class
        if is_expanded:
            # First click: collapse from expanded to anchored
            active_nav_btn.click()

            try:
                self.page.locator(
                    ".console-nav__drawer.pluto--visible.pluto--expanded:not(.pluto--location-bottom)"
                ).wait_for(state="hidden", timeout=2000)
            except PlaywrightTimeoutError:
                # Retry
                active_nav_btn.click()
                sy.sleep(0.2)

            anchored_drawer = self.page.locator(
                ".console-nav__drawer.pluto--visible:not(.pluto--expanded):not(.pluto--location-bottom)"
            )
            if anchored_drawer.count() > 0 and anchored_drawer.first.is_visible():
                # Re-find the selected button to close anchored drawer
                selected_btn = self.page.locator(
                    "button.console-main-nav__item.pluto--selected"
                ).first
                if selected_btn.count() > 0:
                    selected_btn.click()
        else:
            # Drawer is anchored (not expanded), single click closes it
            active_nav_btn.click()

        nav_drawer.wait_for(state="hidden", timeout=5000)

    def get_version(self) -> str:
        """Get the version string displayed in the navbar badge.

        Returns:
            The version string (e.g., "v0.51.0").
        """
        version_badge = self.page.get_by_role("button").filter(
            has_text=re.compile(r"^v\d+\.\d+\.\d+$")
        )
        version_badge.first.wait_for(state="visible", timeout=5000)
        return version_badge.first.inner_text().strip()

    def fill_input_field(self, input_label: str, value: str) -> None:
        """Fill an input field by label."""
        input_field = (
            self.page.locator(f"text={input_label}")
            .locator("..")
            .locator("input")
            .first
        )
        input_field.wait_for(state="attached", timeout=300)
        input_field.fill(value)

    def get_input_field(self, input_label: str) -> str:
        """Get the value of an input field by label."""
        input_field = (
            self.page.locator(f"text={input_label}")
            .locator("..")
            .locator("input")
            .first
        )
        input_field.wait_for(state="attached", timeout=400)
        return input_field.input_value(timeout=200)

    def click_btn(self, button_label: str) -> None:
        """Click a button by label."""
        button = (
            self.page.locator(f"text={button_label}")
            .locator("..")
            .locator("button")
            .first
        )
        button.wait_for(state="attached", timeout=300)
        button.click()

    def click_checkbox(self, checkbox_label: str) -> None:
        """Click a checkbox by label."""
        checkbox = (
            self.page.locator(f"text={checkbox_label}")
            .locator("..")
            .locator("input[type='checkbox']")
            .first
        )
        checkbox.wait_for(state="attached", timeout=300)
        checkbox.click()

    def get_toggle(self, toggle_label: str) -> bool:
        """Get the value of a toggle by label."""
        toggle = (
            self.page.locator(f"text={toggle_label}")
            .locator("..")
            .locator("input[type='checkbox']")
            .first
        )
        return toggle.is_checked()

    def get_dropdown_value(self, dropdown_label: str) -> str:
        """Get the current value of a dropdown by label."""
        dropdown_button = (
            self.page.locator(f"text={dropdown_label}")
            .locator("..")
            .locator("button")
            .first
        )
        dropdown_button.wait_for(state="attached", timeout=5000)
        return dropdown_button.inner_text().strip()

    def get_selected_button(self, button_options: list[str]) -> str:
        """Get the currently selected button from a button group (no label)."""
        for option in button_options:
            button = self.page.get_by_text(option).first
            if button.count() > 0:
                button.wait_for(state="attached", timeout=5000)
                class_name = button.get_attribute("class") or ""
                if "pluto-btn--filled" in class_name:
                    return option

        raise RuntimeError(f"No selected button found from options: {button_options}")

    def select_from_dropdown(
        self, text: str, placeholder: str | None = None, exact: bool = False
    ) -> None:
        """Select an item from an open dropdown."""
        sy.sleep(0.3)
        target_item = f".pluto-list__item:not(.pluto-tree__item):has-text('{text}')"

        search_input = None
        if placeholder is not None:
            search_input = self.page.locator(f"input[placeholder*='{placeholder}']")
        if search_input is None or search_input.count() == 0:
            search_input = self.page.locator("input[placeholder*='Search']")
        if search_input.count() > 0:
            search_input.wait_for(state="attached", timeout=5000)
            current_value = search_input.input_value()
            if current_value != text:
                search_input.fill(text)
            sy.sleep(0.2)

        for _ in range(5):
            try:
                self.page.wait_for_selector(target_item, timeout=5000)
                if exact:
                    for candidate in self.page.locator(target_item).all():
                        if candidate.inner_text().strip() == text:
                            candidate.click()
                            return
                else:
                    item = self.page.locator(target_item).first
                    item.wait_for(state="attached", timeout=5000)
                    item.click()
                    return
            except Exception:
                sy.sleep(1)
                continue

        items = self.page.locator(
            ".pluto-list__item:not(.pluto-tree__item)"
        ).all_text_contents()
        raise RuntimeError(
            f"Could not find item '{text}' in dropdown. Available items: {items}"
        )

    def click(self, selector: str | Locator) -> None:
        """Click an element by text selector or Locator.

        Args:
            selector: Either a text string to search for, or a Playwright Locator
        """
        if isinstance(selector, str):
            element = self.page.get_by_text(selector, exact=True).first
            element.click(timeout=500)
        else:
            with self._bring_to_front(selector) as el:
                el.click(timeout=500)

        sy.sleep(0.1)

    def meta_click(self, selector: str | Locator) -> None:
        """Click an element with platform-appropriate modifier key held.

        Args:
            selector: Either a text string to search for, or a Playwright Locator
        """
        if isinstance(selector, str):
            element = self.page.get_by_text(selector, exact=True).first
            element.click(timeout=500, modifiers=["ControlOrMeta"])
        else:
            with self._bring_to_front(selector) as el:
                el.click(timeout=500, modifiers=["ControlOrMeta"])

        sy.sleep(0.1)

    @contextmanager
    def _bring_to_front(self, element: Locator) -> Generator[Locator, None, None]:
        """Context manager that temporarily brings an element to the front.

        Sets z-index to 9999 to ensure the element is clickable even if other
        elements are overlapping it. Restores the original z-index on exit.

        TODO: This is a workaround for overlapping elements in the Console UI.
        Once the underlying z-index bug is fixed, this method should be removed.

        Args:
            element: The Playwright Locator to bring to front

        Yields:
            The same element, now with z-index set to 9999
        """
        original_z_index = element.evaluate("el => el.style.zIndex || 'auto'")
        element.evaluate("el => el.style.zIndex = '9999'")
        try:
            yield element
        finally:
            element.evaluate(f"el => el.style.zIndex = '{original_z_index}'")

    def get_tab(self, name: str) -> Locator:
        """Get a tab locator by its name.

        Args:
            name: The name/title of the tab to find

        Returns:
            Locator for the tab element
        """
        return (
            self.page.locator(".pluto-tabs-selector")
            .locator("div")
            .filter(has_text=re.compile(f"^{re.escape(name)}$"))
            .filter(has=self.page.locator("[aria-label='pluto-tabs__close']"))
            .first
        )

    def wait_for_tab(self, name: str) -> None:
        """Wait for a tab to be visible.

        Args:
            name: The name/title of the tab to wait for.
        """
        self.get_tab(name).wait_for(state="visible", timeout=5000)

    def close_tab(self, name: str) -> None:
        """Close a tab using a randomly selected modality.

        Randomly chooses between:
        - Click close button (X)
        - Context menu -> Close

        Args:
            name: Name of the tab to close
        """
        self.close_left_toolbar()
        tab = self.get_tab(name)
        tab.wait_for(state="visible", timeout=5000)

        modality = random.choice(["button", "context_menu"])
        if modality == "button":
            tab.get_by_label("pluto-tabs__close").click()
        else:
            self.ctx_menu.action(tab.locator("p"), "Close", exact=False)

        if self.page.get_by_text("Lose Unsaved Changes").count() > 0:
            self.page.get_by_role("button", name="Confirm").click()

    def rename_tab(self, *, old_name: str, new_name: str) -> None:
        """Rename a tab using a randomly selected modality.

        Randomly chooses between:
        - Double-click on tab name
        - Context menu -> Rename

        Args:
            old_name: Current name of the tab
            new_name: New name for the tab
        """
        self.close_left_toolbar()
        tab = self.get_tab(old_name)
        tab.wait_for(state="visible", timeout=5000)

        modality = random.choice(["dblclick", "context_menu"])

        # Ensure focus
        tab.click()

        if modality == "dblclick":
            tab.locator("p").first.dblclick()
        else:
            self.ctx_menu.action(tab.locator("p"), "Rename", exact=False)

        # The tab name uses Text.Editable which becomes contentEditable (not an input)
        editable_text = tab.locator("p[contenteditable='true']").first
        try:
            editable_text.wait_for(state="visible", timeout=2000)
        except PlaywrightTimeoutError:
            # Fallback to more general selector
            editable_text = tab.locator(
                ".pluto-text--editable[contenteditable='true']"
            ).first
            editable_text.wait_for(state="visible", timeout=2000)

        self.select_all_and_type(new_name)
        self.press_enter()

        sy.sleep(0.3)
        self.get_tab(new_name).wait_for(state="visible", timeout=10000)

    def split_horizontal(self, tab_name: str) -> None:
        """Split a leaf horizontally via context menu.

        Args:
            tab_name: Name of the tab to split
        """
        tab = self.get_tab(tab_name)
        self.ctx_menu.action(tab, "Split Horizontally", exact=False)

    def split_vertical(self, tab_name: str) -> None:
        """Split a leaf vertically via context menu.

        Args:
            tab_name: Name of the tab to split
        """
        tab = self.get_tab(tab_name)
        self.ctx_menu.action(tab, "Split Vertically", exact=False)

    def focus(self, tab_name: str) -> None:
        """Focus on a leaf (maximize it) via context menu.

        Args:
            tab_name: Name of the tab to focus
        """
        tab = self.get_tab(tab_name)
        self.ctx_menu.action(tab, "Focus", exact=False)

    def show_visualization_toolbar(self) -> None:
        """Show the visualization toolbar by pressing V."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.pluto--location-bottom.pluto--visible"
        )
        if bottom_drawer.count() == 0 or not bottom_drawer.is_visible():
            self.page.keyboard.press("V")
        bottom_drawer.wait_for(state="visible", timeout=5000)

    def hide_visualization_toolbar(self) -> None:
        """Hide the visualization toolbar by pressing Escape then V."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.pluto--location-bottom.pluto--visible"
        )
        if bottom_drawer.count() == 0 or not bottom_drawer.is_visible():
            return

        self.page.keyboard.press("Escape")
        self.page.keyboard.press("V")

        try:
            bottom_drawer.wait_for(state="hidden", timeout=2000)
        except PlaywrightTimeoutError:
            self.close_left_toolbar()
            self.page.locator(".pluto-tabs-selector__btn").first.click()
            self.page.keyboard.press("V")
            bottom_drawer.wait_for(state="hidden", timeout=5000)

    def get_visualization_toolbar_title(self) -> str:
        """Get the title from the visualization toolbar header."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.pluto--location-bottom.pluto--visible"
        )
        # Use combined selector to handle different page type structures
        header = bottom_drawer.locator(
            "header .pluto-breadcrumb__segment, header .pluto-header__text"
        ).first
        header.wait_for(state="visible", timeout=5000)
        return header.inner_text().strip()

    # ============================================================
    # Playwright Wrapper Methods
    # These methods provide a consistent interface for common Playwright operations,
    # reducing direct Playwright coupling in client code.
    # ============================================================

    def wait_for_visible(self, locator: Locator) -> None:
        """Wait for a locator to become visible.

        Args:
            locator: The Playwright Locator to wait for.
        """
        locator.wait_for(state="visible", timeout=5000)

    def wait_for_hidden(self, locator: Locator) -> None:
        """Wait for a locator to become hidden.

        Args:
            locator: The Playwright Locator to wait for.
        """
        locator.wait_for(state="hidden", timeout=5000)

    def press_key(self, key: str) -> None:
        """Press a keyboard key.

        Args:
            key: The key to press (e.g., "Enter", "Escape", "ControlOrMeta+a").
        """
        self.page.keyboard.press(key)

    def press_escape(self) -> None:
        """Press the Escape key."""
        self.page.keyboard.press("Escape")

    def press_enter(self) -> None:
        """Press the Enter key."""
        self.page.keyboard.press("Enter")

    def press_meta_enter(self) -> None:
        """Press Ctrl/Cmd+Enter."""
        self.page.keyboard.press("ControlOrMeta+Enter")

    def press_delete(self) -> None:
        """Press the Delete key."""
        self.page.keyboard.press("Delete")

    def select_all(self) -> None:
        """Select all text in the focused element."""
        sy.sleep(0.1)
        self.page.keyboard.press("ControlOrMeta+a")

    def select_all_and_type(self, text: str) -> None:
        """Select all text in the focused element and type new text."""
        self.select_all()
        sy.sleep(0.1)
        self.page.keyboard.type(text)

    def type_text(self, text: str) -> None:
        """Type text using the keyboard.

        Args:
            text: The text to type.
        """
        self.page.keyboard.type(text)

    def get_by_text(self, text: str, *, exact: bool = False) -> Locator:
        """Get a locator for an element containing the specified text.

        Args:
            text: The text to search for.
            exact: If True, match the exact text. If False, match substring.

        Returns:
            A Playwright Locator for the element.
        """
        return self.page.get_by_text(text, exact=exact)

    def click_role(self, role: AriaRole, name: str) -> None:
        """Click on an element by its ARIA role and accessible name.

        Args:
            role: The ARIA role (e.g., "button", "checkbox", "textbox").
            name: The accessible name of the element.
        """
        self.page.get_by_role(role, name=name).click()

    def locator(self, selector: str) -> Locator:
        """Create a locator for the given CSS selector.

        Args:
            selector: CSS selector string.

        Returns:
            A Playwright Locator for the element(s).
        """
        return self.page.locator(selector)

    def wait_for_selector_visible(self, selector: str) -> Locator:
        """Wait for a selector to become visible and return its locator.

        Args:
            selector: CSS selector string.

        Returns:
            A Playwright Locator for the visible element.
        """
        loc = self.page.locator(selector)
        loc.wait_for(state="visible", timeout=5000)
        return loc

    def wait_for_selector_hidden(self, selector: str) -> None:
        """Wait for a selector to become hidden.

        Args:
            selector: CSS selector string.
        """
        self.page.locator(selector).wait_for(state="hidden", timeout=5000)

    def sleep(self, ms: int) -> None:
        """Wait for a specified number of milliseconds.

        Args:
            ms: The number of milliseconds to wait.
        """
        self.page.wait_for_timeout(ms)

    def read_clipboard(self) -> str:
        """Read text from the clipboard.

        Returns:
            The clipboard text.
        """
        return str(self.page.evaluate("navigator.clipboard.readText()"))

    def context_menu_action(self, item: Locator, action: str) -> None:
        """Perform a context menu action on an item.

        Args:
            item: The Locator for the element to right-click.
            action: The exact text of the menu action to click.
        """
        self.ctx_menu.action(item, action)

    def show_toolbar(self, shortcut_key: str, item_prefix: str) -> None:
        """Show a navigation toolbar using keyboard shortcut.

        Args:
            shortcut_key: The keyboard shortcut (e.g., "d", "u", "r").
            item_prefix: The ID prefix of items in the panel (e.g., "rack:", "role:").
        """
        items = self.page.locator(f"div[id^='{item_prefix}']")
        if items.count() > 0 and items.first.is_visible():
            return
        self.press_key(shortcut_key)
        items.first.wait_for(state="visible", timeout=5000)

    def delete_with_confirmation(self, item: Locator) -> None:
        """Delete an item via context menu with confirmation modal."""
        self.ctx_menu.action(item, "Delete")
        modal = self.page.locator(self.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        modal.get_by_role("button", name="Delete", exact=True).click()
        modal.wait_for(state="hidden", timeout=5000)

    def open_modal(self, command: str, selector: str) -> None:
        """Open a modal via command palette.

        Args:
            command: The command to execute in the palette.
            selector: CSS selector for the modal to wait for.
        """
        self.command_palette(command)
        self.page.locator(selector).wait_for(state="visible", timeout=5000)

    def close_modal(self, selector: str) -> None:
        """Close a modal via close button.

        Args:
            selector: CSS selector for the modal to wait for hidden.
        """
        close_btn = self.page.locator(
            ".pluto-dialog__dialog button:has(svg.pluto-icon--close)"
        ).first
        close_btn.click()
        self.page.locator(selector).wait_for(state="hidden", timeout=5000)
