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
from collections.abc import Callable, Generator
from contextlib import contextmanager
from typing import Literal

from playwright.sync_api import Locator, Page, expect
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

import synnax as sy
from console.context_menu import ContextMenu
from console.notifications import NotificationsClient
from console.tree import Tree

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
    # Focusing a tab collapses the mosaic to a single overlaid leaf instead of
    # opening a modal dialog.
    FOCUS_SELECTOR = ".pluto-panel-mosaic__overlaid-leaf"
    # Scoped to the mosaic: the top nav renders a second tab strip for panels.
    TAB_STRIP_SELECTOR = (
        ".console-mosaic .pluto-mosaic__leaf > .pluto-tabs > .pluto-tabs__selector"
    )
    TAB_SELECTOR = f"{TAB_STRIP_SELECTOR} > .pluto-tabs__tab"

    def __init__(self, page: Page):
        self.page = page
        self.ctx_menu = ContextMenu(self.page)
        self.notifications = NotificationsClient(self.page)
        self.tree = Tree(self)

    @property
    def dialog(self) -> Locator:
        """The currently open dialog.

        Closed dialogs unmount, except passthrough ones, which stay mounted at
        opacity 0 and still expose their role; the open-state class excludes
        them.
        """
        return self.page.locator("[role='dialog'].pluto--visible")

    def command_palette(self, command: str, retries: int = 3) -> None:
        """Execute a command via the command palette."""
        self._palette(
            query=command,
            input_text=f">{command}",
            empty_message="No commands found",
            error_prefix="Command palette",
            retries=retries,
        )

    def choose_import_file(self, path: str | list[str]) -> None:
        """Send ``path`` through the "Import components" palette file chooser.

        Only drives the chooser; callers assert the import's outcome on their
        own surface (project tree, Arc panel, notifications).

        :param path: Path, or list of paths, of the JSON file(s) to import.
        """
        with self.page.expect_file_chooser() as fc_info:
            self.command_palette("Import components")
        fc_info.value.set_files(path)

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
                self.dialog.get_by_role("option").first.wait_for(
                    state="attached", timeout=10000
                )
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
                self.dialog.get_by_role("option").filter(has_text=query).first
            )
            try:
                target_result.wait_for(state="visible", timeout=5000)
            except PlaywrightTimeoutError:
                input_value = palette_input.input_value()
                options = self.dialog.get_by_role("option").all_text_contents()
                raise RuntimeError(
                    f"{error_prefix}: Could not find '{query}'. "
                    f"Input value: '{input_value}'. "
                    f"Available options: {options}"
                )
            target_result.click(timeout=5000)
            return

    def command_exists(self, command: str) -> bool:
        """Report whether the command palette offers ``command``.

        Opens the palette, types the command name, checks for a matching
        entry, and closes the palette again. Access-controlled commands are
        hidden from the palette, so this is the probe for permission checks.

        :param command: The exact visible label of the palette command.
        :returns: True if the command is offered.
        """
        palette_btn = self.page.locator(".console-palette button").first
        palette_btn.wait_for(state="visible", timeout=5000)
        palette_btn.click(timeout=5000)
        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.wait_for(state="visible", timeout=5000)
        palette_input.press("ControlOrMeta+a")
        palette_input.type(f">{command}", timeout=5000)
        entry = self.dialog.get_by_role("option").filter(has_text=command).first
        no_results = self.page.get_by_text("No commands found")
        # A hidden command can still leave fuzzy matches in the list, in which
        # case neither the entry nor the empty message ever shows: fall through
        # to a plain visibility check after the wait.
        try:
            entry.or_(no_results).first.wait_for(state="visible", timeout=3000)
        except PlaywrightTimeoutError:
            pass
        exists = entry.is_visible()
        self.press_escape()
        palette_input.wait_for(state="hidden", timeout=5000)
        return exists

    def is_modal_open(self) -> bool:
        """Check if a modal dialog is currently open."""
        return self.page.locator(self.MODAL_SELECTOR).count() > 0

    def check_for_errors(self, match: str) -> bool:
        """Check notifications for a specific error.

        Args:
            match: Substring to search for in the notification message.

        Returns:
            True if a matching error was found, False otherwise.
        """
        for notification in self.notifications.check():
            message = notification.get("message", "")
            if match.lower() in message.lower():
                self.notifications.close(0)
                return True
        return False

    # Resource toolbar toggles live in the left navbar's main content section. The
    # end section holds the Component toggle.
    _RESOURCE_NAV_ITEMS = (
        ".pluto-navbar.pluto--location-left "
        ".pluto-navbar__content:not(.pluto--end) button.console-main-nav__item"
    )

    def show_resource_toolbar(self, name: str) -> None:
        """Show a resource toolbar by its nav item's name (e.g. "Tasks")."""
        nav_drawer = self.page.locator(
            ".console-nav__drawer.pluto--visible:not(.pluto--location-bottom)"
        )
        item = self.page.get_by_role("menuitem", name=name, exact=True)
        selected = "pluto--selected" in (item.get_attribute("class") or "")
        if selected and nav_drawer.count() > 0 and nav_drawer.first.is_visible():
            return
        if not selected:
            item.click(timeout=5000)
        nav_drawer.wait_for(state="visible", timeout=5000)

    def close_left_toolbar(self) -> None:
        """Close any open side nav drawer (left/right, not bottom visualization toolbar)."""
        nav_drawer = self.page.locator(
            ".console-nav__drawer.pluto--visible:not(.pluto--location-bottom)"
        )
        if nav_drawer.count() == 0 or not nav_drawer.first.is_visible():
            return
        active_nav_btn = self.page.locator(
            f"{self._RESOURCE_NAV_ITEMS}.pluto--selected"
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
                    f"{self._RESOURCE_NAV_ITEMS}.pluto--selected"
                ).first
                if selected_btn.count() > 0:
                    selected_btn.click()
        else:
            # Drawer is anchored (not expanded), single click closes it
            active_nav_btn.click()

        nav_drawer.wait_for(state="hidden", timeout=5000)

    def fill_input_field(self, input_label: str, value: str) -> None:
        """Fill an input field by label."""
        input_field = (
            self.page.locator(f"text={input_label}")
            .locator("..")
            .locator("input")
            .first
        )
        input_field.wait_for(state="attached", timeout=5000)
        input_field.fill(value)

    def get_input_field(self, input_label: str) -> str:
        """Get the value of an input field by label."""
        input_field = (
            self.page.locator(f"text={input_label}")
            .locator("..")
            .locator("input")
            .first
        )
        input_field.wait_for(state="attached", timeout=5000)
        return input_field.input_value(timeout=2000)

    def click_btn(self, button_label: str) -> None:
        """Click a button by label."""
        button = (
            self.page.locator(f"text={button_label}")
            .locator("..")
            .locator("button")
            .first
        )
        button.wait_for(state="attached", timeout=300)
        try:
            button.click(timeout=5000)
        except PlaywrightTimeoutError:
            # Toasts stack over the bottom of a form and swallow the click.
            self.notifications.close_all()
            button.click(timeout=5000)

    def click_checkbox(self, checkbox_label: str) -> None:
        """Click a checkbox by label."""
        checkbox = (
            self.page.locator(f"text={checkbox_label}")
            .locator("..")
            .locator("input[type='checkbox']")
            .first
        )
        checkbox.wait_for(state="attached", timeout=300)
        try:
            checkbox.click(timeout=5000)
        except PlaywrightTimeoutError:
            # Toasts stack over the bottom of a form and swallow the click.
            self.notifications.close_all()
            checkbox.click(timeout=5000)

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
            text_el = self.page.get_by_text(option, exact=True).first
            if text_el.count() == 0:
                continue
            button = text_el.locator("xpath=ancestor-or-self::button[1]").first
            if button.count() == 0:
                continue
            button.wait_for(state="attached", timeout=5000)
            class_name = button.get_attribute("class") or ""
            if "pluto--selected" in class_name:
                return option

        raise RuntimeError(f"No selected button found from options: {button_options}")

    def select_labels(self, labels: list[str], scope: Locator | None = None) -> None:
        """Pick labels from a "Select labels" dropdown.

        :param labels: The label names to select.
        :param scope: Where the trigger lives. Defaults to the whole page.
        """
        parent = self.page if scope is None else scope
        parent.get_by_text("Select labels", exact=True).click(timeout=5000)
        for name in labels:
            self.select_from_dropdown(name, exact=True)
        self.press_escape()

    def select_from_dropdown(
        self,
        text: str,
        placeholder: str | None = None,
        exact: bool = False,
        reopen: Callable[[], None] | None = None,
    ) -> None:
        """Select an item from an open dropdown.

        :param text: Visible text of the item to select.
        :param placeholder: Search input placeholder to filter with before selecting.
        :param exact: Require an exact text match instead of a substring match.
        :param reopen: Re-opens the dropdown. Called before a retry when the dialog
            closed before the item was found (e.g. a re-render dismissed it).
        """
        target = self.dialog.get_by_role("option", name=text, exact=exact)
        generic = self.dialog.locator("input[placeholder*='Search']")
        specific = (
            self.dialog.locator(f"input[placeholder*='{placeholder}']")
            if placeholder is not None
            else generic
        )
        loaded = self.dialog.get_by_role("option").or_(
            self.dialog.locator(".pluto-list__items--empty")
        )
        loading = self.dialog.locator(".pluto-icon--loading")

        def apply_search(refill: bool = False) -> None:
            # The dropdown renders its search input and options together, so either
            # one appearing means it is open.
            generic.or_(target).first.wait_for(state="visible", timeout=5000)
            # Type only once the first page has settled: a search that races the
            # initial answer is overwritten by it.
            loaded.first.wait_for(state="visible", timeout=5000)
            expect(loading).to_have_count(0, timeout=5000)
            search_input = specific if specific.count() > 0 else generic
            if search_input.count() == 0:
                return
            if refill:
                search_input.fill("")
            if search_input.input_value() != text:
                search_input.fill(text)
            sy.sleep(0.1)

        try:
            apply_search()
        except (PlaywrightTimeoutError, AssertionError):
            pass  # The retry loop below waits again and runs the recovery path.
        last_recovery_error: Exception | None = None
        for _ in range(5):
            try:
                item = target.first
                item.wait_for(state="visible", timeout=5000)
                item.click(timeout=5000)
                return
            except Exception:
                try:
                    # Toasts overlap the dropdown and swallow the click.
                    self.notifications.close_all()
                    if reopen is not None and self.dialog.count() == 0:
                        reopen()
                        apply_search()
                    else:
                        # A fresh search replaces a list a late answer overwrote.
                        apply_search(refill=True)
                except Exception as e:
                    # A failed recovery consumes this retry instead of aborting.
                    last_recovery_error = e
                    sy.sleep(1)
                continue

        items = self.dialog.get_by_role("option").all_text_contents()
        message = f"Could not find item '{text}' in dropdown. Available items: {items}"
        if last_recovery_error is not None:
            message += f" (last recovery attempt failed: {last_recovery_error})"
        raise RuntimeError(message)

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
                el.click(timeout=500, force=True)

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
            self.page.locator(self.TAB_SELECTOR)
            .filter(has_text=re.compile(f"^{re.escape(name)}$"))
            .filter(has=self.page.locator("[aria-label='Close']"))
            .first
        )

    def get_read_only_tab(self, name: str) -> Locator:
        """Get a tab locator by name without requiring a close button.

        A user who cannot write the panel gets no close button, so get_tab's
        filter never matches for them.

        :param name: The name/title of the tab to find.
        :returns: Locator for the tab element.
        """
        return (
            self.page.locator(self.TAB_SELECTOR)
            .filter(has_text=re.compile(f"^{re.escape(name)}$"))
            .first
        )

    def tab_is_closable(self, name: str) -> bool:
        """Report whether the named tab offers a close button.

        :param name: The name/title of the tab to check.
        """
        tab = self.get_read_only_tab(name)
        tab.wait_for(state="visible", timeout=5000)
        return tab.get_by_label("Close", exact=True).count() > 0

    def tab_menu_has_option(self, name: str, option: str) -> bool:
        """Report whether the named tab's context menu offers the option.

        :param name: The name/title of the tab whose menu to open.
        :param option: The menu option text to look for.
        """
        tab = self.get_read_only_tab(name)
        tab.wait_for(state="visible", timeout=5000)
        self.ctx_menu.open_on(tab)
        try:
            return self.ctx_menu.has_option(option)
        finally:
            self.ctx_menu.close()

    def select_tab(self, name: str) -> None:
        """Bring the named tab to the front without requiring a close button.

        :param name: The name/title of the tab to select.
        """
        tab = self.get_read_only_tab(name)
        tab.wait_for(state="visible", timeout=5000)
        tab.click()

    def mosaic_is_static(self) -> bool:
        """Report whether the mosaic withholds every structural write.

        :returns: True when no tab offers a close button and no leaf offers an
            add button.
        """
        closes = self.page.locator(f"{self.TAB_SELECTOR} [aria-label='Close']")
        adds = self.page.locator(".pluto-panel-mosaic__create")
        return closes.count() == 0 and adds.count() == 0

    def get_tombstone(self, name: str) -> Locator:
        """Get the tombstone a deleted resource's tab shows in place of content.

        Args:
            name: The name the resource had when it was deleted

        Returns:
            Locator for the tombstone element
        """
        return self.page.get_by_role("group", name=f"{name} was deleted", exact=True)

    def restore_tombstone(self, name: str) -> None:
        """Click Restore on a deleted resource's tombstone.

        Args:
            name: The name the resource had when it was deleted
        """
        tombstone = self.get_tombstone(name)
        tombstone.get_by_role("button", name="Restore", exact=True).click()

    def close_tombstone(self, name: str) -> None:
        """Click Close on a deleted resource's tombstone.

        Args:
            name: The name the resource had when it was deleted
        """
        tombstone = self.get_tombstone(name)
        tombstone.get_by_role("button", name="Close", exact=True).click()

    def tab_names(self) -> list[str]:
        """Return the names of every open tab in the mosaic."""
        tabs = self.page.locator(self.TAB_SELECTOR)
        return [tabs.nth(i).inner_text().strip() for i in range(tabs.count())]

    def create_panel(self) -> None:
        """Create a panel through the panel selector's add button."""
        add_btn = self.page.locator(
            ".console-panel-selector button:has(.pluto-icon--add)"
        ).first
        add_btn.wait_for(state="visible", timeout=5000)
        add_btn.click()
        self.page.locator(self.TAB_STRIP_SELECTOR).first.wait_for(
            state="visible", timeout=10000
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
            # The close button reveals on tab hover; the tab icon covers it until
            # then.
            tab.hover()
            tab.get_by_label("Close", exact=True).click()
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

        renamed_from_menu = False
        if modality == "context_menu":
            # Only tabs backed by a resource carry a Rename item.
            self.ctx_menu.open_on(tab.locator("p"))
            renamed_from_menu = self.ctx_menu.has_option("Rename", exact=False)
            if renamed_from_menu:
                self.ctx_menu.click_option("Rename", exact=False)
            else:
                self.ctx_menu.close()
        if not renamed_from_menu:
            tab.locator("p").first.dblclick()

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
        self.ctx_menu.action(tab, "Split horizontally", exact=False)

    def split_vertical(self, tab_name: str) -> None:
        """Split a leaf vertically via context menu.

        Args:
            tab_name: Name of the tab to split
        """
        tab = self.get_tab(tab_name)
        self.ctx_menu.action(tab, "Split vertically", exact=False)

    def focus(self, tab_name: str) -> None:
        """Focus on a leaf (maximize it) via context menu.

        Args:
            tab_name: Name of the tab to focus
        """
        self.close_left_toolbar()
        # Focus is a session action, not a panel write, so it must work on tabs
        # without a close button (a user who cannot write the panel).
        tab = self.get_read_only_tab(tab_name)
        tab.wait_for(state="visible", timeout=5000)
        tab.click()
        self.ctx_menu.action(tab.locator("p"), "Focus", exact=False)

    def show_visualization_toolbar(self) -> None:
        """Show the bottom Component toolbar via its nav toggle."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.pluto--location-bottom.pluto--visible"
        )
        if bottom_drawer.count() > 0 and bottom_drawer.is_visible():
            return

        self.page.get_by_role("menuitem", name="Component", exact=True).click()
        bottom_drawer.wait_for(state="visible", timeout=5000)

    def hide_visualization_toolbar(self) -> None:
        """Hide the bottom Component toolbar via its nav toggle."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.pluto--location-bottom.pluto--visible"
        )
        if bottom_drawer.count() == 0 or not bottom_drawer.is_visible():
            return

        self.page.get_by_role("menuitem", name="Component", exact=True).click()
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

    def click_role(self, role: AriaRole, name: str, *, exact: bool = False) -> None:
        """Click on an element by its ARIA role and accessible name.

        Args:
            role: The ARIA role (e.g., "button", "checkbox", "textbox").
            name: The accessible name of the element.
            exact: If True, match the name exactly.
        """
        self.page.get_by_role(role, name=name, exact=exact).click()

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

    def drop_files(self, paths: list[str], target: Locator | None = None) -> None:
        """Drop OS files or directories onto ``target`` (default: the mosaic).

        Synthetic ``DataTransfer`` drops carry no file-system entries, so the
        drag is dispatched through CDP, which produces a native drag whose
        dropped items resolve real ``FileSystemEntry`` objects.

        :param paths: Absolute paths of the files or directories to drop.
        :param target: Locator to drop onto; the mosaic when omitted.
        """
        if target is None:
            target = self.page.locator(".console-mosaic").first
        target.wait_for(state="visible", timeout=5000)
        box = target.bounding_box()
        if box is None:
            raise AssertionError("drop target has no bounding box")
        x = box["x"] + box["width"] / 2
        y = box["y"] + box["height"] / 2
        cdp = self.page.context.new_cdp_session(self.page)
        try:
            # Copy, link, and move together present as effectAllowed "all",
            # matching a real OS file drag; a copy-only mask reads "copy" and
            # the app does not recognize the drag as a file drag.
            data = {"items": [], "files": paths, "dragOperationsMask": 19}

            def send(event_type: str) -> None:
                cdp.send(
                    "Input.dispatchDragEvent",
                    {"type": event_type, "x": x, "y": y, "data": data},
                )

            send("dragEnter")
            send("dragOver")
            # The drop is accepted only once React commits the file-drag state,
            # signalled by the mosaic's drag shield appearing.
            self.page.locator(".pluto-mosaic__shield").first.wait_for(
                state="visible", timeout=5000
            )
            send("dragOver")
            send("drop")
        finally:
            cdp.detach()

    def show_toolbar(self, shortcut_key: str, item_prefix: str) -> None:
        """Show a navigation toolbar using keyboard shortcut.

        Args:
            shortcut_key: The keyboard shortcut (e.g., "d", "u", "r").
            item_prefix: The ID prefix of items in the panel (e.g., "rack:", "role:").
        """
        items = self.page.locator(f"div[id^='{item_prefix}']")
        if items.count() > 0 and items.first.is_visible():
            return
        # Right after login the nav item may still be hidden (its permission
        # query is loading), which swallows the shortcut, so retry the press.
        for attempt in range(3):
            self.press_key(shortcut_key)
            try:
                items.first.wait_for(state="visible", timeout=3000)
                return
            except PlaywrightTimeoutError:
                if attempt == 2:
                    raise

    def get_list_item(self, selector: str, name: str) -> Locator:
        """Get a list item locator by CSS selector filtered by text content."""
        return self.page.locator(selector).filter(has_text=name).first

    def deselect_all_items(self, container: Locator | Page, item_selector: str) -> None:
        """Deselect all checked items by dispatching click on their checkbox labels."""
        checked = container.locator(
            f"{item_selector}:has(input.pluto-input__checkbox-input:checked"
            ":not([aria-label='Favorite']))"
        )
        for _ in range(10):
            if checked.count() == 0:
                break
            checked.first.locator(
                ".pluto-input__checkbox:not(:has(input[aria-label='Favorite']))"
            ).dispatch_event("click")

    def select_items(
        self, names: list[str], get_item_fn: Callable[[str], Locator]
    ) -> Locator:
        """Select multiple items via their checkbox labels, return the last item."""
        last_item = None
        for name in names:
            item = get_item_fn(name)
            item.wait_for(state="visible", timeout=5000)
            # List items also carry a favorite checkbox; the unlabeled one is
            # the selection control.
            checkbox = item.locator(
                ".pluto-input__checkbox:not(:has(input[aria-label='Favorite']))"
            ).first
            if not checkbox.locator("input").is_checked():
                checkbox.dispatch_event("click")
            last_item = item
        assert last_item is not None
        return last_item

    def ctrl_select_items(
        self, names: list[str], get_item_fn: Callable[[str], Locator]
    ) -> Locator:
        """Multi-select items via Ctrl+Click, return the last item."""
        first = get_item_fn(names[0])
        first.wait_for(state="visible", timeout=5000)
        first.click()
        for name in names[1:]:
            item = get_item_fn(name)
            item.wait_for(state="visible", timeout=5000)
            item.click(modifiers=["ControlOrMeta"])
        return get_item_fn(names[-1])

    def locator_exists(self, locator: Locator) -> bool:
        """Check if a locator is visible within 5 seconds."""
        try:
            locator.wait_for(state="visible", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

    def confirm_delete(self) -> None:
        """Confirm an already-open delete confirmation modal.

        Waits for the modal to appear, clicks the Delete button, and waits
        for the modal to close. Use this when the delete action has already
        been triggered (e.g., via context menu or icon click).
        """
        modal = self.page.locator(self.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        self.notifications.close_all()
        modal.get_by_role("button", name="Delete", exact=True).click()
        modal.wait_for(state="hidden", timeout=5000)

    def delete_with_confirmation(self, item: Locator) -> None:
        """Delete an item via context menu with confirmation modal."""
        self.ctx_menu.action(item, "Delete")
        self.confirm_delete()

    def rename_with_modal(self, item: Locator, new_name: str) -> None:
        """Rename an item via context menu and modal dialog.

        Triggers "Rename" from the context menu, fills the Name input
        in the resulting modal, and clicks Save.

        Args:
            item: The Locator for the element to rename.
            new_name: The new name to set.
        """
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Rename")
        modal = self.page.locator(self.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        modal.locator("input[placeholder='Name']").fill(new_name)
        modal.get_by_role("button", name="Save", exact=True).click(timeout=5000)
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
        close_btn = (
            self.page.locator(self.MODAL_SELECTOR)
            .get_by_role("button", name="Close", exact=True)
            .first
        )
        close_btn.click()
        self.page.locator(selector).wait_for(state="hidden", timeout=5000)
