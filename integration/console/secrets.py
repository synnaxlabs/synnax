#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from playwright.sync_api import Locator, TimeoutError as PlaywrightTimeoutError, expect

from console.base import ResourceClient
from console.layout import LayoutClient

_ITEM_SELECTOR = ".console-secret-settings__item"


class _Page(ResourceClient):
    """A record table of Settings that lists secrets or certificates."""

    def __init__(self, layout: LayoutClient, *, section: str, noun: str) -> None:
        super().__init__(layout)
        self._section = section
        self._search_placeholder = f"Search {noun}"

    def exists(self, name: str) -> bool:
        """Check if the page lists an item by name."""
        return self._find_item(name) is not None

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename an item in place.

        :param old_name: The current name of the item.
        :param new_name: The new name for the item.
        """
        self.layout.rename_in_place(self._get_item(old_name), new_name)
        self._get_item(new_name)

    def delete(self, name: str) -> None:
        """Delete an item after confirmation."""
        self.layout.delete_with_confirmation(self._get_item(name))
        expect(self._items(name)).to_have_count(0, timeout=5000)

    def is_selected(self, name: str) -> bool:
        """Check if the item is selected, without searching the page first."""
        item = self._items(name).first
        item.wait_for(state="visible", timeout=5000)
        return item.get_attribute("aria-selected") == "true"

    def show(self) -> None:
        """Show the page in Settings."""
        nav = self.layout.open_settings()
        nav.get_by_role("menuitem", name=self._section, exact=True).click()
        self._search_input().wait_for(state="visible", timeout=5000)

    def _search_input(self) -> Locator:
        return self.page.get_by_placeholder(self._search_placeholder, exact=True)

    def _items(self, name: str) -> Locator:
        return self.page.locator(_ITEM_SELECTOR).and_(
            self.page.get_by_role("option", name=name, exact=True)
        )

    def _find_item(self, name: str) -> Locator | None:
        """Search the page for an item, or return None if it is not listed."""
        self.show()
        self._search_input().fill(name)
        item = self._items(name).first
        try:
            item.wait_for(state="visible", timeout=5000)
        except PlaywrightTimeoutError:
            return None
        return item

    def _get_item(self, name: str) -> Locator:
        """Like _find_item, but raises if the search does not list the item."""
        item = self._find_item(name)
        if item is None:
            raise ValueError(f"'{name}' not found on the {self._section} page")
        return item


class SecretClient(_Page):
    """Console client for the Secrets page of Settings."""

    def __init__(self, layout: LayoutClient) -> None:
        super().__init__(layout, section="Secrets", noun="secrets")

    def create(self, name: str, *, value: str) -> None:
        """Create a stored secret.

        :param name: The name of the secret.
        :param value: The value of the secret.
        """
        self.show()
        self.page.get_by_role("button", name="Create secret", exact=True).click()
        modal = self.page.locator(self.layout.MODAL_SELECTOR).first
        name_input = modal.get_by_label("Secret name", exact=True)
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)
        modal.get_by_placeholder("Value", exact=True).fill(value)
        modal.get_by_role("button", name="Create", exact=True).click()
        modal.wait_for(state="hidden", timeout=5000)
        self._get_item(name)

    def replace_value(self, name: str, *, value: str) -> None:
        """Replace the value of a stored secret."""
        self.ctx_menu.action(self._get_item(name), "Replace value")
        modal = self.page.locator(self.layout.MODAL_SELECTOR).first
        modal.get_by_placeholder("New value", exact=True).fill(value)
        modal.get_by_role("button", name="Replace", exact=True).click()
        modal.wait_for(state="hidden", timeout=5000)

    def used_by(self, name: str) -> list[str]:
        """Return the names of the resources that use a secret.

        Leaves the popover that lists them open.
        """
        item = self._get_item(name)
        self.ctx_menu.action(item, "Used by")
        popover = self._used_by_popover()
        names = popover.locator("button")
        # The list loads after the popover opens, and says so when it is empty.
        names.first.or_(popover.get_by_text("Nothing uses this secret.")).wait_for(
            state="visible", timeout=5000
        )
        return [n.strip() for n in names.all_inner_texts()]

    def open_used_by(self, *, secret: str, resource: str) -> None:
        """Click a resource in the popover that lists the uses of a secret."""
        if resource not in self.used_by(secret):
            raise ValueError(f"'{resource}' does not use secret '{secret}'")
        self._used_by_popover().get_by_text(resource, exact=True).click()

    def _used_by_popover(self) -> Locator:
        return self.page.locator(".console-secret-settings__popover")


class CertificateClient(_Page):
    """Console client for the Certificates page of Settings."""

    def __init__(self, layout: LayoutClient) -> None:
        super().__init__(layout, section="Certificates", noun="certificates")
