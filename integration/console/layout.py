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
from pathlib import Path
from typing import Literal

from playwright.sync_api import (
    Locator,
    Page,
    TimeoutError as PlaywrightTimeoutError,
    expect,
)

import synnax as sy
from console.context_menu import ContextMenu
from console.notifications import NotificationsClient, click_past
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


_SPAN_UNITS = {
    "d": sy.TimeSpan.DAY,
    "h": sy.TimeSpan.HOUR,
    "min": sy.TimeSpan.MINUTE,
    "s": sy.TimeSpan.SECOND,
    "ms": sy.TimeSpan.MILLISECOND,
    "μs": sy.TimeSpan.MICROSECOND,
    "ns": sy.TimeSpan.NANOSECOND,
}
_SPAN_PART = re.compile(r"(\d+(?:\.\d+)?)(min|ms|μs|ns|d|h|s)")


class LayoutClient:
    """
    Layout and UI primitives for Console automation.

    This is the PRIMARY abstraction layer over Playwright. All UI operations
    (command palette, forms, keyboard, navigation) should go through this client.
    """

    MODAL_SELECTOR = "div.lyra-dialog__dialog.lyra--modal.lyra--visible"
    SETTINGS_NAV_SELECTOR = "nav[aria-label='Settings']"
    SETTINGS_CREATE_SELECTOR = ".console-settings-table__create"
    # Text.Editable carries the class whether or not it is editing; the attribute is
    # what marks the element that currently takes keystrokes.
    EDITABLE_SELECTOR = ".lyra-text--editable[contenteditable='true']"
    # Focusing a tab collapses the mosaic to a single overlaid leaf instead of
    # opening a modal dialog.
    FOCUS_SELECTOR = ".pluto-panel-mosaic__overlaid-leaf"
    # Scoped to the mosaic: the top nav renders a second tab strip for panels.
    TAB_STRIP_SELECTOR = (
        ".console-mosaic .pluto-mosaic__leaf > .lyra-tabs > .lyra-tabs__selector"
    )
    TAB_SELECTOR = f"{TAB_STRIP_SELECTOR} > .lyra-tabs__tab"

    def __init__(self, *, page: Page, results_dir: Path):
        self.page = page
        self.results_dir = results_dir
        self.ctx_menu = ContextMenu(self.page)
        self.notifications = NotificationsClient(self.page)
        self.tree = Tree(self)

    @property
    def dialog(self) -> Locator:
        """The currently open dialog.

        A dialog mounts hidden until its frame places it, and still exposes its role;
        the open-state class excludes it until then.
        """
        return self.page.locator("[role='dialog'].lyra--visible")

    def form_section(self, title: str) -> Locator:
        """The form section with the given title."""
        header = self.page.locator(".lyra-form-section__header").get_by_text(
            title, exact=True
        )
        return self.page.locator(".lyra-form-section").filter(has=header)

    def command_palette(self, command: str) -> None:
        """Execute a command via the command palette."""
        self._palette(query=command, input_text=f">{command}")

    def choose_import_file(self, path: str | list[str]) -> None:
        """Send ``path`` through the "Import components" palette file chooser.

        Only drives the chooser; callers assert the import's outcome on their
        own surface (project tree, Arc panel, notifications).

        :param path: Path, or list of paths, of the JSON file(s) to import.
        """
        with self.page.expect_file_chooser() as fc_info:
            self.command_palette("Import components")
        fc_info.value.set_files(path)

    def search_palette(self, query: str) -> None:
        """Search for a resource via the command palette (without > prefix)."""
        self._palette(query=query, input_text=query)

    def _palette(self, *, query: str, input_text: str) -> None:
        palette_btn = self.page.locator(".console-palette button").first
        palette_btn.wait_for(state="visible", timeout=5000)
        palette_btn.click(timeout=5000)
        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.wait_for(state="visible", timeout=5000)
        palette_input.press("ControlOrMeta+a")
        palette_input.type(input_text, timeout=5000)
        # A command shows once its visibility query answers, without a new search.
        target = self.dialog.get_by_role("option").filter(has_text=query).first
        try:
            target.wait_for(state="visible", timeout=10000)
        except PlaywrightTimeoutError as e:
            options = self.dialog.get_by_role("option").all_text_contents()
            raise RuntimeError(
                f"Palette: could not find '{query}'. "
                f"Input value: '{palette_input.input_value()}'. "
                f"Available options: {options}"
            ) from e
        target.click(timeout=5000)

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

        :param match: Substring to search for in the notification message.
        :returns: True if a matching error was found, False otherwise.
        """
        for notification in self.notifications.check():
            if match.lower() in notification.message.lower():
                self.notifications.close(notification_index=0)
                return True
        return False

    # Resource toolbar toggles live in the left navbar's main content section. The
    # end section holds the Component toggle.
    _RESOURCE_NAV_ITEMS = (
        ".lyra-navbar.lyra--location-left "
        ".lyra-navbar__content:not(.lyra--end) button.console-main-nav__item"
    )

    def show_resource_toolbar(self, name: str) -> None:
        """Show a resource toolbar by its nav item's name (e.g. "Tasks")."""
        nav_drawer = self.page.locator(
            ".console-nav__drawer.lyra--visible:not(.lyra--location-bottom)"
        )
        item = self.page.get_by_role("menuitem", name=name, exact=True)
        selected = "lyra--selected" in (item.get_attribute("class") or "")
        if selected and nav_drawer.count() > 0 and nav_drawer.first.is_visible():
            return
        if not selected:
            item.click(timeout=5000)
        nav_drawer.wait_for(state="visible", timeout=5000)

    def close_left_toolbar(self) -> None:
        """Close any open side nav drawer (left/right, not bottom visualization
        toolbar).
        """
        nav_drawer = self.page.locator(
            ".console-nav__drawer.lyra--visible:not(.lyra--location-bottom)"
        )
        if nav_drawer.count() == 0 or not nav_drawer.first.is_visible():
            return
        active_nav_btn = self.page.locator(
            f"{self._RESOURCE_NAV_ITEMS}.lyra--selected"
        ).first
        if active_nav_btn.count() == 0:
            return
        drawer_class = nav_drawer.first.get_attribute("class") or ""
        is_expanded = "lyra--expanded" in drawer_class
        if is_expanded:
            # First click: collapse from expanded to anchored
            active_nav_btn.click()

            try:
                self.page.locator(
                    ".console-nav__drawer.lyra--visible.lyra--expanded:not(.lyra--location-bottom)"
                ).wait_for(state="hidden", timeout=2000)
            except PlaywrightTimeoutError:
                # Retry
                active_nav_btn.click()
                sy.sleep(0.2)

            anchored_drawer = self.page.locator(
                ".console-nav__drawer.lyra--visible:not(.lyra--expanded):not(.lyra--location-bottom)"
            )
            if anchored_drawer.count() > 0 and anchored_drawer.first.is_visible():
                # Re-find the selected button to close anchored drawer
                selected_btn = self.page.locator(
                    f"{self._RESOURCE_NAV_ITEMS}.lyra--selected"
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
            .locator("input, textarea")
            .first
        )
        input_field.wait_for(state="attached", timeout=5000)
        input_field.fill(value)

    def get_input_field(self, input_label: str) -> str:
        """Get the value of an input field by label."""
        input_field = (
            self.page.locator(f"text={input_label}")
            .locator("..")
            .locator("input, textarea")
            .first
        )
        input_field.wait_for(state="attached", timeout=5000)
        return input_field.input_value(timeout=2000)

    def get_time_span_field(self, input_label: str) -> sy.TimeSpan:
        """Get the span a duration field shows, such as "2ms 300μs"."""
        text = self.get_input_field(input_label)
        span = sy.TimeSpan(0)
        for part in text.split():
            match = _SPAN_PART.fullmatch(part)
            if match is None:
                raise ValueError(f"{input_label} shows {text!r}, not a duration")
            span += sy.TimeSpan(float(match[1]) * _SPAN_UNITS[match[2]])
        return span

    def click_btn(self, button_label: str) -> None:
        """Click a button by label."""
        button = (
            self.page.locator(f"text={button_label}")
            .locator("..")
            .locator("button")
            .first
        )
        click_past(button)

    def click_checkbox(self, checkbox_label: str) -> None:
        """Click a checkbox by label."""
        checkbox = (
            self.page.locator(f"text={checkbox_label}")
            .locator("..")
            .locator("input[type='checkbox']")
            .first
        )
        click_past(checkbox)

    def get_toggle(self, toggle_label: str) -> bool:
        """Get the value of a toggle by label."""
        toggle = (
            self.page.locator(f"text={toggle_label}")
            .locator("..")
            .locator("input[type='checkbox']")
            .first
        )
        return toggle.is_checked()

    def _dropdown_button(self, dropdown_label: str) -> Locator:
        return (
            self.page.locator(f"text={dropdown_label}")
            .locator("..")
            .locator("button")
            .first
        )

    def _dropdown_selection(self, dropdown_label: str) -> Locator:
        # The trigger can also show content after the selection, such as a status badge.
        return self._dropdown_button(dropdown_label).locator(
            ":scope > .lyra-select__label"
        )

    def get_dropdown_value(self, dropdown_label: str) -> str:
        """Get the current value of a dropdown by label."""
        selection = self._dropdown_selection(dropdown_label)
        selection.wait_for(state="attached", timeout=5000)
        return selection.inner_text().strip()

    def wait_for_dropdown_value(self, dropdown_label: str, value: str) -> None:
        """Wait until a dropdown, found by label, shows the given value."""
        expect(self._dropdown_selection(dropdown_label)).to_have_text(
            value, timeout=5000
        )

    def get_selected_button(
        self, button_options: list[str], *, scope: Locator | None = None
    ) -> str:
        """Get the currently selected button from a button group (no label).

        :param scope: Where the button group lives. Defaults to the whole page.
        """
        parent = self.page if scope is None else scope
        for option in button_options:
            text_el = parent.get_by_text(option, exact=True).first
            if text_el.count() == 0:
                continue
            button = text_el.locator("xpath=ancestor-or-self::button[1]").first
            if button.count() == 0:
                continue
            button.wait_for(state="attached", timeout=5000)
            class_name = button.get_attribute("class") or ""
            if "lyra--selected" in class_name:
                return option

        raise RuntimeError(f"No selected button found from options: {button_options}")

    def select_labels(self, labels: list[str], *, scope: Locator | None = None) -> None:
        """Pick labels from an empty "Labels" dropdown.

        :param labels: The label names to select.
        :param scope: Where the trigger lives. Defaults to the whole page.
        """
        parent = self.page if scope is None else scope
        trigger = parent.get_by_role("button", name="Labels", exact=True)
        # Only its parts take a click, and an empty trigger has only its placeholder.
        click_past(trigger.locator(".lyra-select-multiple-trigger-placeholder"))
        for name in labels:
            self.select_from_dropdown(name, exact=True)
        self.press_escape()

    def select_from_dropdown(
        self,
        text: str,
        *,
        placeholder: str | None = None,
        exact: bool = False,
    ) -> None:
        """Select an item from an open dropdown.

        :param text: Visible text of the item to select.
        :param placeholder: Search input placeholder to filter with before selecting.
        :param exact: Match the whole item text, ignoring case, instead of a substring.
        :raises RuntimeError: If the item does not show within 5 seconds.
        """
        name = re.compile(rf"^{re.escape(text)}$", re.IGNORECASE) if exact else text
        target = self.dialog.get_by_role("option", name=name)
        generic = self.dialog.locator("input[placeholder*='Search']")
        specific = (
            self.dialog.locator(f"input[placeholder*='{placeholder}']")
            if placeholder is not None
            else generic
        )
        # The dropdown renders its search input and options together, so either one
        # showing means it is open.
        generic.or_(target).first.wait_for(state="visible", timeout=5000)
        search_input = specific if specific.count() > 0 else generic
        if search_input.count() > 0 and search_input.input_value() != text:
            search_input.fill(text)
        item = target.first
        try:
            item.wait_for(state="visible", timeout=5000)
        except PlaywrightTimeoutError as e:
            items = self.dialog.get_by_role("option").all_text_contents()
            raise RuntimeError(
                f"Could not find item '{text}' in dropdown. Available items: {items}"
            ) from e
        click_past(item)

    def click(self, selector: str | Locator) -> None:
        """Click an element by text selector or Locator.

        :param selector: Either a text string to search for, or a Playwright Locator
        """
        if isinstance(selector, str):
            element = self.page.get_by_text(selector, exact=True).first
            element.click(timeout=5000)
        else:
            with self._bring_to_front(selector) as el:
                el.click(timeout=5000, force=True)

        sy.sleep(0.1)

    def meta_click(self, selector: str | Locator) -> None:
        """Click an element with platform-appropriate modifier key held.

        :param selector: Either a text string to search for, or a Playwright Locator
        """
        if isinstance(selector, str):
            element = self.page.get_by_text(selector, exact=True).first
            element.click(timeout=5000, modifiers=["ControlOrMeta"])
        else:
            with self._bring_to_front(selector) as el:
                el.click(timeout=5000, modifiers=["ControlOrMeta"])

        sy.sleep(0.1)

    @contextmanager
    def _bring_to_front(self, element: Locator) -> Generator[Locator]:
        """Context manager that temporarily brings an element to the front.

        Sets z-index to 9999 to ensure the element is clickable even if other
        elements are overlapping it. Restores the original z-index on exit.

        TODO: This is a workaround for overlapping elements in the Console UI.
        Once the underlying z-index bug is fixed, this method should be removed.

        :param element: The Playwright Locator to bring to front
        :returns: A context that yields the same element, with z-index set to 9999
        """
        original_z_index = element.evaluate("el => el.style.zIndex || 'auto'")
        element.evaluate("el => el.style.zIndex = '9999'")
        try:
            yield element
        finally:
            element.evaluate(f"el => el.style.zIndex = '{original_z_index}'")

    def get_tab(self, name: str | re.Pattern[str]) -> Locator:
        """Get a tab locator by its name.

        :param name: The name/title of the tab to find, or a pattern it matches
        :returns: Locator for the tab element
        """
        pattern = name if isinstance(name, re.Pattern) else f"^{re.escape(name)}$"
        return (
            self.page.locator(self.TAB_SELECTOR)
            .filter(has_text=re.compile(pattern))
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

        :param name: The name the resource had when it was deleted
        :returns: Locator for the tombstone element
        """
        return self.page.get_by_role("group", name=f"{name} was deleted", exact=True)

    def restore_tombstone(self, name: str) -> None:
        """Click Restore on a deleted resource's tombstone.

        :param name: The name the resource had when it was deleted
        """
        tombstone = self.get_tombstone(name)
        tombstone.get_by_role("button", name="Restore", exact=True).click()

    def close_tombstone(self, name: str) -> None:
        """Click Close on a deleted resource's tombstone.

        :param name: The name the resource had when it was deleted
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
            ".console-panel-selector button:has(.lyra-icon--add)"
        ).first
        add_btn.wait_for(state="visible", timeout=5000)
        add_btn.click()
        self.page.locator(self.TAB_STRIP_SELECTOR).first.wait_for(
            state="visible", timeout=10000
        )

    def wait_for_tab(self, name: str) -> None:
        """Wait for a tab to be visible.

        :param name: The name/title of the tab to wait for.
        """
        self.get_tab(name).wait_for(state="visible", timeout=5000)

    def open_settings(self) -> Locator:
        """Open the Settings tab, or show it if it is already open.

        :returns: The section list of Settings.
        """
        nav = self.page.locator(self.SETTINGS_NAV_SELECTOR)
        if not nav.is_visible():
            self.command_palette("Open settings")
            nav.wait_for(state="visible", timeout=5000)
        return nav

    def open_settings_create(self, button: str) -> Locator:
        """Open the create item of the shown Settings page.

        :param button: The name of the page's create button, such as "Create label".
        :returns: The create item.
        """
        self.page.get_by_role("button", name=button, exact=True).click()
        item = self.page.locator(self.SETTINGS_CREATE_SELECTOR)
        item.wait_for(state="visible", timeout=5000)
        return item

    def submit_settings_create(self, item: Locator, name_field: Locator) -> None:
        """Confirm a filled create item, then close it.

        :param item: The create item from open_settings_create.
        :param name_field: A field of the item that the create empties.
        """
        item.get_by_role("button", name="Create", exact=True).click()
        # The item stays open and starts empty again once the Core has the new one.
        expect(name_field).to_have_value("", timeout=5000)
        item.get_by_role("button", name="Cancel", exact=True).click()
        item.wait_for(state="hidden", timeout=5000)

    @staticmethod
    def _tab_name_text(tab: Locator) -> Locator:
        """Return the text of the tab's name. A name in parts, such as a saved view's,
        ends with the part that a rename edits."""
        return tab.locator("p").last

    def close_tab(self, name: str | re.Pattern[str]) -> None:
        """Close a tab using a randomly selected modality.

        Randomly chooses between:
        - Click close button (X)
        - Context menu -> Close

        :param name: Name of the tab to close, or a pattern it matches
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
            self.ctx_menu.action(self._tab_name_text(tab), "Close", exact=False)

        if self.page.get_by_text("Lose Unsaved Changes").count() > 0:
            self.page.get_by_role("button", name="Confirm").click()

    def rename_tab(self, *, old_name: str, new_name: str) -> None:
        """Rename a tab using a randomly selected modality.

        Randomly chooses between:
        - Double-click on tab name
        - Context menu -> Rename

        :param old_name: Current name of the tab
        :param new_name: New name for the tab
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
            self.ctx_menu.open_on(self._tab_name_text(tab))
            renamed_from_menu = self.ctx_menu.has_option("Rename", exact=False)
            if renamed_from_menu:
                self.ctx_menu.click_option("Rename", exact=False)
            else:
                self.ctx_menu.close()
        if not renamed_from_menu:
            self._tab_name_text(tab).dblclick()

        # The tab name uses Text.Editable which becomes contentEditable (not an input)
        editable_text = tab.locator("p[contenteditable='true']").first
        try:
            editable_text.wait_for(state="visible", timeout=2000)
        except PlaywrightTimeoutError:
            # Fallback to more general selector
            editable_text = tab.locator(
                ".lyra-text--editable[contenteditable='true']"
            ).first
            editable_text.wait_for(state="visible", timeout=2000)

        self.select_all_and_type(new_name)
        self.press_enter()

        sy.sleep(0.3)
        self.get_tab(new_name).wait_for(state="visible", timeout=10000)

    def split_horizontal(self, tab_name: str) -> None:
        """Split a leaf horizontally via context menu.

        :param tab_name: Name of the tab to split
        """
        tab = self.get_tab(tab_name)
        self.ctx_menu.action(tab, "Split horizontally", exact=False)

    def split_vertical(self, tab_name: str) -> None:
        """Split a leaf vertically via context menu.

        :param tab_name: Name of the tab to split
        """
        tab = self.get_tab(tab_name)
        self.ctx_menu.action(tab, "Split vertically", exact=False)

    def focus(self, tab_name: str) -> None:
        """Focus on a leaf (maximize it) via context menu.

        :param tab_name: Name of the tab to focus
        """
        self.close_left_toolbar()
        # Focus is a session action, not a panel write, so it must work on tabs
        # without a close button (a user who cannot write the panel).
        tab = self.get_read_only_tab(tab_name)
        tab.wait_for(state="visible", timeout=5000)
        tab.click()
        self.ctx_menu.action(self._tab_name_text(tab), "Focus", exact=False)

    def show_visualization_toolbar(self) -> Locator:
        """Show the bottom Component toolbar via its nav toggle.

        :returns: The toolbar's drawer.
        """
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.lyra--location-bottom.lyra--visible"
        )
        if bottom_drawer.count() > 0 and bottom_drawer.is_visible():
            return bottom_drawer

        self.page.get_by_role("menuitem", name="Component", exact=True).click()
        bottom_drawer.wait_for(state="visible", timeout=5000)
        return bottom_drawer

    def hide_visualization_toolbar(self) -> None:
        """Hide the bottom Component toolbar via its nav toggle."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.lyra--location-bottom.lyra--visible"
        )
        if bottom_drawer.count() == 0 or not bottom_drawer.is_visible():
            return

        self.page.get_by_role("menuitem", name="Component", exact=True).click()
        bottom_drawer.wait_for(state="hidden", timeout=5000)

    def get_visualization_toolbar_title(self) -> str:
        """Get the title from the visualization toolbar header."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.lyra--location-bottom.lyra--visible"
        )
        # Use combined selector to handle different page type structures
        header = bottom_drawer.locator(
            "header .lyra-breadcrumb__segment, header .lyra-header__text"
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

        :param locator: The Playwright Locator to wait for.
        """
        locator.wait_for(state="visible", timeout=5000)

    def wait_for_hidden(self, locator: Locator) -> None:
        """Wait for a locator to become hidden.

        :param locator: The Playwright Locator to wait for.
        """
        locator.wait_for(state="hidden", timeout=5000)

    def press_key(self, key: str) -> None:
        """Press a keyboard key.

        :param key: The key to press (e.g., "Enter", "Escape", "ControlOrMeta+a").
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

        :param text: The text to type.
        """
        self.page.keyboard.type(text)

    def get_by_text(self, text: str, *, exact: bool = False) -> Locator:
        """Get a locator for an element containing the specified text.

        :param text: The text to search for.
        :param exact: If True, match the exact text. If False, match substring.
        :returns: A Playwright Locator for the element.
        """
        return self.page.get_by_text(text, exact=exact)

    def click_role(self, role: AriaRole, name: str, *, exact: bool = False) -> None:
        """Click on an element by its ARIA role and accessible name.

        :param role: The ARIA role (e.g., "button", "checkbox", "textbox").
        :param name: The accessible name of the element.
        :param exact: If True, match the name exactly.
        """
        self.page.get_by_role(role, name=name, exact=exact).click()

    def locator(self, selector: str) -> Locator:
        """Create a locator for the given CSS selector.

        :param selector: CSS selector string.
        :returns: A Playwright Locator for the element(s).
        """
        return self.page.locator(selector)

    def wait_for_selector_visible(self, selector: str) -> Locator:
        """Wait for a selector to become visible and return its locator.

        :param selector: CSS selector string.
        :returns: A Playwright Locator for the visible element.
        """
        loc = self.page.locator(selector)
        loc.wait_for(state="visible", timeout=5000)
        return loc

    def wait_for_selector_hidden(self, selector: str) -> None:
        """Wait for a selector to become hidden.

        :param selector: CSS selector string.
        """
        self.page.locator(selector).wait_for(state="hidden", timeout=5000)

    def sleep(self, ms: int) -> None:
        """Wait for a specified number of milliseconds.

        :param ms: The number of milliseconds to wait.
        """
        self.page.wait_for_timeout(ms)

    def read_clipboard(self) -> str:
        """Read text from the clipboard.

        :returns: The clipboard text.
        """
        return str(self.page.evaluate("navigator.clipboard.readText()"))

    def drag(self, source: Locator, target: Locator) -> None:
        """Drag ``source`` and drop it on the center of ``target``.

        Use this instead of ``Locator.drag_to``. When the first press of ``drag_to`` is
        blocked, it presses again without a release, and the drag breaks.

        :param source: The element to drag.
        :param target: The element to drop onto.
        """
        # A hover retries until it reaches the source, and it never presses.
        source.hover()
        self.page.mouse.down()
        target.wait_for(state="visible")
        # A row scrolled out of its list still counts as visible.
        target.scroll_into_view_if_needed()
        box = target.bounding_box()
        if box is None:
            raise AssertionError("drag target has no bounding box")
        x = box["x"] + box["width"] / 2
        y = box["y"] + box["height"] / 2
        self.page.mouse.move(x, y, steps=10)
        self.page.mouse.up()

    def drop_files(self, paths: list[str], *, target: Locator | None = None) -> None:
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

    def get_list_item(self, selector: str, *, name: str) -> Locator:
        """Get a list item locator by CSS selector filtered by text content."""
        return self.page.locator(selector).filter(has_text=name).first

    def deselect_all_items(
        self, container: Locator | Page, *, item_selector: str
    ) -> None:
        """Deselect all checked items by dispatching click on their checkbox labels."""
        checked = container.locator(
            f"{item_selector}:has(input.lyra-input__checkbox-input:checked"
            ":not([aria-label='Favorite']))"
        )
        # Wait out each unchecking before the next. Clicking again while the row is
        # still checked re-selects the box that was just cleared.
        while (remaining := checked.count()) > 0:
            checked.first.locator(
                ".lyra-input__checkbox:not(:has(input[aria-label='Favorite']))"
            ).dispatch_event("click")
            expect(checked).to_have_count(remaining - 1, timeout=5000)

    def select_items(
        self, names: list[str], *, get_item_fn: Callable[[str], Locator]
    ) -> Locator:
        """Select multiple items via their checkbox labels, return the last item."""
        last_item = None
        for name in names:
            item = get_item_fn(name)
            item.wait_for(state="visible", timeout=5000)
            # List items also carry a favorite checkbox; the unlabeled one is
            # the selection control.
            checkbox = item.locator(
                ".lyra-input__checkbox:not(:has(input[aria-label='Favorite']))"
            ).first
            if not checkbox.locator("input").is_checked():
                checkbox.dispatch_event("click")
            last_item = item
        assert last_item is not None
        return last_item

    def ctrl_select_items(
        self, names: list[str], *, get_item_fn: Callable[[str], Locator]
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
        click_past(modal.get_by_role("button", name="Delete", exact=True))
        modal.wait_for(state="hidden", timeout=5000)

    def delete_with_confirmation(self, item: Locator) -> None:
        """Delete an item via context menu with confirmation modal."""
        self.ctx_menu.action(item, "Delete")
        self.confirm_delete()

    def rename_in_place(self, item: Locator, new_name: str) -> None:
        """Rename an item via context menu. The name edits in place.

        Triggers "Rename" from the context menu, which turns the item's name into an
        editable text element, then types the new name and commits it.

        :param item: The Locator for the element to rename.
        :param new_name: The new name to set.
        """
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Rename")
        editable = item.locator(self.EDITABLE_SELECTOR).first
        editable.wait_for(state="visible", timeout=5000)
        self.select_all_and_type(new_name)
        self.press_enter()
        editable.wait_for(state="hidden", timeout=5000)

    def open_modal(self, command: str, *, selector: str) -> None:
        """Open a modal via command palette.

        :param command: The command to execute in the palette.
        :param selector: CSS selector for the modal to wait for.
        """
        self.command_palette(command)
        self.page.locator(selector).wait_for(state="visible", timeout=5000)

    def close_modal(self, selector: str) -> None:
        """Close a modal via close button.

        :param selector: CSS selector for the modal to wait for hidden.
        """
        close_btn = (
            self.page.locator(self.MODAL_SELECTOR)
            .get_by_role("button", name="Close", exact=True)
            .first
        )
        close_btn.click()
        self.page.locator(selector).wait_for(state="hidden", timeout=5000)
