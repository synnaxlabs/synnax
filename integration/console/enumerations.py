#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
import time
from collections.abc import Callable

from playwright.sync_api import Locator, TimeoutError as PlaywrightTimeoutError, expect

import synnax as sy
from console.base import ResourceClient
from console.color import pick as pick_color
from console.layout import LayoutClient
from synnax import enumeration

_ITEM_SELECTOR = ".console-enumeration-settings__item"
_NAME_LABEL = re.compile(r"^Name \d+$")
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
        """Create an enumeration, then add entries in the grid under it.

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
        self._add_entries(name, entries)
        return self.wait_for(
            name,
            lambda e: [x.name for x in e.entries] == entries,
            f"The Core did not add the entries of {name}",
        )

    def add_entry(self, enumeration_name: str, name: str) -> None:
        """Add an entry with the Add entry button under an enumeration.

        :param enumeration_name: The name of the enumeration.
        :param name: The name of the new entry.
        """
        self.search(enumeration_name)
        self._add_entries(enumeration_name, [name])

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename an enumeration from its menu."""
        self.layout.rename_in_place(self.item(old_name), new_name)
        self.wait_for(new_name, lambda _: True, f"The Core did not rename {old_name}")

    def rename_entry(
        self, enumeration_name: str, *, old_name: str, new_name: str
    ) -> None:
        """Rename an entry of a listed enumeration in its Name cell."""
        self._commit(self.cell(enumeration_name, "Name", old_name), new_name)
        self.wait_for(
            enumeration_name,
            lambda e: any(x.name == new_name for x in e.entries),
            f"The Core did not rename {old_name}",
        )

    def set_value(self, enumeration_name: str, entry_name: str, value: int) -> None:
        """Type a value into an entry of a listed enumeration and commit it."""
        self._commit(self.cell(enumeration_name, "Value", entry_name), str(value))

    def toggle_abnormal(self, enumeration_name: str, entry_name: str) -> None:
        """Flip the Abnormal switch of an entry of a listed enumeration."""
        self.cell(enumeration_name, "Abnormal", entry_name).click()

    def set_color(self, enumeration_name: str, entry_name: str, hex_color: str) -> None:
        """Pick a fixed color for an entry of a listed enumeration."""
        row = self._row(enumeration_name, entry_name)
        swatch = self.entries(enumeration_name).get_by_role(
            "button", name=f"Pick color {row}", exact=True
        )
        pick_color(self.page, swatch, hex_color=hex_color)

    def remove_entry(self, enumeration_name: str, entry_name: str) -> None:
        """Remove an entry of a listed enumeration with its Remove button."""
        names = self._names(enumeration_name)
        count = names.count()
        row = self._row(enumeration_name, entry_name)
        self.entries(enumeration_name).get_by_role(
            "button", name=f"Remove entry {row}", exact=True
        ).click()
        expect(names).to_have_count(count - 1, timeout=5000)

    def restore(self, name: str) -> None:
        """Restore a built-in enumeration from its menu."""
        self.ctx_menu.action(self.item(name), "Restore default")

    def delete(self, name: str) -> None:
        """Delete an enumeration from its menu and confirm it."""
        self.layout.delete_with_confirmation(self.item(name))
        self._wait_gone(name)

    def replace(self, name: str, *, with_name: str) -> None:
        """Delete an enumeration with "Replace with" in its delete dialog, which moves
        each user to the enumeration named with_name."""
        self.ctx_menu.action(self.item(name), "Delete")
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
        """Open the menu of an enumeration and return which of the options it shows."""
        self.ctx_menu.open_on(self.item(name))
        shown = [o for o in options if self.ctx_menu.has_option(o)]
        self.ctx_menu.close()
        return shown

    def search(self, term: str) -> None:
        """Show the Enumerations page with only the enumerations that match term."""
        self._show_page()
        self.page.get_by_placeholder(_SEARCH_PLACEHOLDER, exact=True).fill(term)

    def item(self, name: str) -> Locator:
        """The listed enumeration named name."""
        return self.page.locator(_ITEM_SELECTOR).and_(
            self.page.get_by_role("option", name=name, exact=True)
        )

    def entries(self, name: str) -> Locator:
        """Open the listed enumeration named name, if it is closed, and return the
        entries that show under it."""
        entries = self.page.get_by_role("region", name=f"{name} entries", exact=True)
        item = self.item(name)
        if item.get_attribute("aria-expanded") != "true":
            item.click()
        entries.wait_for(state="visible", timeout=5000)
        return entries

    def cell(self, enumeration_name: str, column: str, entry_name: str) -> Locator:
        """The cell of a column in the row of the listed entry named entry_name."""
        row = self._row(enumeration_name, entry_name)
        return self.entries(enumeration_name).get_by_label(
            f"{column} {row}", exact=True
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
        expect(self.item(name)).to_have_count(0, timeout=5000)
        deadline = time.monotonic() + _CORE_TIMEOUT_S
        while self.client.enumerations.retrieve(names=[name]):
            if time.monotonic() >= deadline:
                raise PlaywrightTimeoutError(f"The Core did not delete {name}")
            self.page.wait_for_timeout(_POLL_MS)

    def _names(self, enumeration_name: str) -> Locator:
        """The Name cells of the listed enumeration, in order."""
        return self.entries(enumeration_name).get_by_label(_NAME_LABEL)

    def _entry_names(self, enumeration_name: str) -> list[str]:
        names = self._names(enumeration_name)
        return [names.nth(i).input_value() for i in range(names.count())]

    def _row(self, enumeration_name: str, entry_name: str) -> int:
        """The row of the listed entry named entry_name, from 1."""
        return self._entry_names(enumeration_name).index(entry_name) + 1

    def _add_entries(self, enumeration_name: str, names: list[str]) -> None:
        """Add an entry for each name with the Add entry button, then name it."""
        entries = self.entries(enumeration_name)
        for name in names:
            row = self._names(enumeration_name).count() + 1
            entries.get_by_role("button", name="Add entry", exact=True).click()
            self._commit(entries.get_by_label(f"Name {row}", exact=True), name)

    @staticmethod
    def _commit(cell: Locator, text: str) -> None:
        """Type into a cell of the entries grid, then blur it to commit the text."""
        cell.fill(text)
        cell.blur()

    def _show_page(self) -> None:
        """Show the Enumerations page of Settings."""
        nav = self.layout.open_settings()
        nav.get_by_role("menuitem", name="Enumerations", exact=True).click()
        self.page.get_by_placeholder(_SEARCH_PLACEHOLDER, exact=True).wait_for(
            state="visible", timeout=5000
        )
