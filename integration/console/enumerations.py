#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from collections.abc import Callable

from playwright.sync_api import Locator, TimeoutError as PlaywrightTimeoutError, expect

import synnax as sy
from console.base import ResourceClient
from console.color import pick as pick_color
from console.layout import LayoutClient
from synnax import enumeration

_HEADING_SELECTOR = ".console-enumeration-settings__heading"
_ITEM_SELECTOR = ".console-enumeration-settings__item"
_SEARCH_PLACEHOLDER = "Search enumerations"
_POLL_MS = 100
_CORE_TIMEOUT_S = 5


class EnumerationClient(ResourceClient):
    """Console client for the Enumerations page of Settings. The Console sends an edit
    a short time after it is made, so a method that takes the name of the enumeration
    waits until the Core holds the edit."""

    def __init__(self, *, layout: LayoutClient, client: sy.Synnax) -> None:
        super().__init__(layout)
        self.client = client

    def create(self, name: str, *, entries: list[str]) -> enumeration.Enumeration:
        """Create an enumeration, then add entries in the form that opens under it.

        :param name: The name of the new enumeration.
        :param entries: The names of the entries to add, in order.
        :returns: The enumeration as the Core holds it.
        """
        self._show_page()
        item = self.layout.open_settings_create("Create enumeration")
        item.get_by_placeholder("Enumeration name", exact=True).fill(name)
        item.get_by_role("button", name="Create", exact=True).click()
        item.wait_for(state="hidden", timeout=5000)
        self.wait_for(name, lambda _: True, f"The Core did not create {name}")
        self.search(name)
        self._add_entries(entries)
        return self.wait_for(
            name,
            lambda e: len(e.entries) == len(entries),
            f"The Core did not add the entries of {name}",
        )

    def add_entry(self, enumeration_name: str, name: str) -> None:
        """Add an entry with the add button on the heading of an enumeration.

        :param enumeration_name: The name of the enumeration.
        :param name: The name of the new entry.
        """
        self.search(enumeration_name)
        self.page.get_by_role(
            "button", name=f"Add an entry to {enumeration_name}", exact=True
        ).click()
        self._add_entries([name])

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename an enumeration from the menu of its heading."""
        self.layout.rename_in_place(self.heading(old_name), new_name)
        self.wait_for(new_name, lambda _: True, f"The Core did not rename {old_name}")

    def rename_entry(
        self, enumeration_name: str, *, old_name: str, new_name: str
    ) -> None:
        """Rename an entry of a shown enumeration from the menu of the entry."""
        self.layout.rename_in_place(self.entry(old_name), new_name)
        self.wait_for(
            enumeration_name,
            lambda e: any(x.name == new_name for x in e.entries),
            f"The Core did not rename {old_name}",
        )

    def set_value(self, entry_name: str, value: int) -> None:
        """Type a value into an entry of a shown enumeration and commit it."""
        field = self.entry(entry_name).get_by_label("Value", exact=True)
        field.fill(str(value))
        field.press("Enter")

    def toggle_abnormal(self, entry_name: str) -> None:
        """Flip the Abnormal switch of an entry of a shown enumeration."""
        self.entry(entry_name).get_by_label("Abnormal", exact=True).click()

    def set_color(self, entry_name: str, hex_color: str) -> None:
        """Pick a fixed color for an entry of a shown enumeration."""
        swatch = self.entry(entry_name).get_by_label("Entry color", exact=True)
        pick_color(self.page, swatch, hex_color=hex_color)

    def remove_entry(self, entry_name: str) -> None:
        """Remove an entry of a shown enumeration from the menu of the entry."""
        self.ctx_menu.action(self.entry(entry_name), "Remove")
        expect(self.entry(entry_name)).to_have_count(0, timeout=5000)

    def restore(self, name: str) -> None:
        """Restore a built-in enumeration from the menu of its heading."""
        self.ctx_menu.action(self.heading(name), "Restore default")

    def delete(self, name: str) -> None:
        """Delete an enumeration from the menu of its heading and confirm it."""
        self.layout.delete_with_confirmation(self.heading(name))
        self._wait_gone(name)

    def replace(self, name: str, *, with_name: str) -> None:
        """Delete an enumeration with "Replace with" in its delete dialog, which moves
        each user to the enumeration named with_name."""
        self.ctx_menu.action(self.heading(name), "Delete")
        modal = self.page.locator(self.layout.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        modal.get_by_role("button", name="Replace with", exact=True).click()
        self.page.get_by_placeholder("Search enumerations...", exact=True).fill(
            with_name
        )
        self.page.get_by_role("option", name=with_name, exact=True).click()
        modal.get_by_role("button", name="Replace", exact=True).click()
        modal.wait_for(state="hidden", timeout=5000)
        self._wait_gone(name)

    def menu_options(self, name: str, options: list[str]) -> list[str]:
        """Open the menu of a heading and return which of the options it shows."""
        self.ctx_menu.open_on(self.heading(name))
        shown = [o for o in options if self.ctx_menu.has_option(o)]
        self.ctx_menu.close()
        return shown

    def search(self, term: str) -> None:
        """Show the Enumerations page with only the enumerations that match term."""
        self._show_page()
        self.page.get_by_placeholder(_SEARCH_PLACEHOLDER, exact=True).fill(term)

    def heading(self, name: str) -> Locator:
        """The heading of the shown enumeration named name."""
        return self.page.locator(_HEADING_SELECTOR).filter(
            has=self.page.get_by_text(name, exact=True)
        )

    def entry(self, name: str) -> Locator:
        """The line of the shown entry named name."""
        return self.page.locator(_ITEM_SELECTOR).and_(
            self.page.get_by_role("option", name=name, exact=True)
        )

    def wait_for(
        self,
        name: str,
        check: Callable[[enumeration.Enumeration], bool],
        message: str,
    ) -> enumeration.Enumeration:
        """Wait until the Core holds an enumeration named name that passes check.

        :returns: The enumeration as the Core holds it.
        :raises PlaywrightTimeoutError: If the Core holds none in time.
        """
        deadline = time.monotonic() + _CORE_TIMEOUT_S
        while True:
            found = self.client.enumerations.retrieve(names=[name])
            if len(found) == 1 and check(found[0]):
                return found[0]
            if time.monotonic() >= deadline:
                raise PlaywrightTimeoutError(message)
            self.page.wait_for_timeout(_POLL_MS)

    def _wait_gone(self, name: str) -> None:
        """Wait until neither the page nor the Core holds the enumeration."""
        expect(self.heading(name)).to_have_count(0, timeout=5000)
        deadline = time.monotonic() + _CORE_TIMEOUT_S
        while self.client.enumerations.retrieve(names=[name]):
            if time.monotonic() >= deadline:
                raise PlaywrightTimeoutError(f"The Core did not delete {name}")
            self.page.wait_for_timeout(_POLL_MS)

    def _add_entries(self, names: list[str]) -> None:
        """Type each name into the open new entry form, then close the form."""
        field = self.page.get_by_placeholder("Entry name", exact=True)
        for name in names:
            field.fill(name)
            field.press("Enter")
            expect(field).to_have_value("", timeout=5000)
        field.press("Escape")
        expect(field).to_have_count(0, timeout=5000)

    def _show_page(self) -> None:
        """Show the Enumerations page of Settings."""
        nav = self.layout.open_settings()
        nav.get_by_role("menuitem", name="Enumerations", exact=True).click()
        self.page.get_by_placeholder(_SEARCH_PLACEHOLDER, exact=True).wait_for(
            state="visible", timeout=5000
        )
