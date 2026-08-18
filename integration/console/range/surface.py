#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator

from console.base import ResourceClient

FAVORITE_ACTIONS = ("Add to favorites", "Favorite")
UNFAVORITE_ACTIONS = ("Remove from favorites", "Unfavorite")
CREATE_MODAL_SELECTOR = ".console-range-create-layout"
NAME_INPUT_PLACEHOLDER = "Name"


class Surface(ResourceClient):
    """Shared context-menu and create-modal helpers for range surfaces."""

    def _any_text_locator(self, parent: Locator, texts: tuple[str, ...]) -> Locator:
        """Build a locator matching any of the given text options."""
        loc = parent.get_by_text(texts[0], exact=True)
        for text in texts[1:]:
            loc = loc.or_(parent.get_by_text(text, exact=True))
        return loc

    def _click_visible_option(self, texts: tuple[str, ...]) -> None:
        """Click whichever text variant is visible in the open context menu."""
        menu = self.layout.page.get_by_role("menu")
        for text in texts:
            btn = menu.get_by_text(text, exact=True)
            if btn.count() > 0 and btn.is_visible():
                self.ctx_menu.click_option(text)
                return
        self.ctx_menu.click_option(texts[0])

    def _ctx_action_any(self, element: Locator, texts: tuple[str, ...]) -> None:
        """Right-click element and click the first visible text variant."""
        self.ctx_menu.open_on(element)
        self._click_visible_option(texts)

    def _pick_stage_from_dropdown(self, stage_button: Locator, stage: str) -> None:
        """Click a stage button and select a stage from the dropdown."""
        stage_button.click()
        dropdown = self.layout.page.locator(".pluto-list__item").filter(has_text=stage)
        dropdown.click(timeout=2000)
        dropdown.wait_for(state="hidden", timeout=2000)

    def fill_create_modal(
        self,
        name: str,
        *,
        persisted: bool = True,
        parent: str | None = None,
        labels: list[str] | None = None,
        stage: str | None = None,
    ) -> None:
        """Fill and submit the range creation modal.

        Assumes the modal is already open and visible.
        """
        modal = self.layout.page.locator(CREATE_MODAL_SELECTOR)
        # Scope to the modal: the range overview page has a name input with the
        # same placeholder.
        name_input = modal.locator(f"input[placeholder='{NAME_INPUT_PLACEHOLDER}']")
        name_input.fill(name)

        if stage is not None:
            stage_button = (
                modal.locator("button")
                .filter(has_text="To do")
                .or_(modal.locator("button").filter(has_text="In progress"))
                .or_(modal.locator("button").filter(has_text="Completed"))
                .first
            )
            self._pick_stage_from_dropdown(stage_button, stage)

        if parent is not None:
            parent_button = modal.locator("button").filter(has_text="Select range")
            parent_button.click()
            # Scope to the picker dropdown: the explorer view has a search
            # input with the same placeholder.
            search_input = self.layout.page.locator(
                ".pluto-dialog__dialog.pluto--visible "
                "input[placeholder='Search ranges...']"
            )
            search_input.fill(parent)
            self.layout.page.locator(".pluto-range__list-item").filter(
                has_text=parent
            ).click(timeout=5000)

        if labels is not None:
            self.layout.select_labels(labels)

        if persisted:
            save_button = modal.get_by_role("button", name="Save to Core")
        else:
            save_button = modal.get_by_role("button", name="Save locally")

        save_button.click(timeout=2000)
        modal.wait_for(state="hidden", timeout=5000)
