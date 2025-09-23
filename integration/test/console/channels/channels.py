#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Page

import synnax as sy

from ..console_page import ConsolePage


class Channels(ConsolePage):
    """Console channels management interface"""

    def __init__(self, page: Page):
        super().__init__(page)

    def create(
        self,
        name: str,
        virtual: bool = False,
        is_index: bool = False,
        data_type: sy.CrudeDataType = sy.DataType.TIMESTAMP,
        index: str = "",
    ) -> bool:
        """
        Create a channel via console UI.
        Returns True if channel was created, False if it already exists.
        """
        # Open command palette and create channel
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        self.page.wait_for_selector("text=Create a Channel", timeout=5000)
        self.page.get_by_text("Create a Channel").click()

        # Fill channel name
        name_input = self.page.locator("text=Name").locator("..").locator("input").first
        name_input.fill(name)

        # Set virtual if needed
        if virtual:
            self.page.get_by_text("Virtual").click()

        # Configure as index or regular channel
        if is_index:
            is_index_toggle = (
                self.page.locator("text=Is Index")
                .locator("..")
                .locator("input[type='checkbox']")
                .first
            )
            is_index_toggle.click()
        else:
            if index == "":
                raise ValueError("Index must be provided if is_index is False")

            # Set data type
            data_type_str = str(sy.DataType(data_type))
            self._select_from_dropdown("Data Type", data_type_str)

            # Set index
            self._select_from_dropdown("Index", index)

        # Create the channel
        self.page.get_by_role("button", name="Create", exact=True).click()
        return True

    def _select_from_dropdown(self, input_field: str, input_text: str) -> None:
        """Helper method for dropdown selection"""
        channel_button = (
            self.page.locator(f"text={input_field}")
            .locator("..")
            .locator("button")
            .first
        )
        channel_button.click()
        search_input = self.page.locator("input[placeholder*='Search']")
        search_input.press("Control+a")
        search_input.type(input_text)
        self.page.wait_for_timeout(300)

        # Iterate through dropdown items
        item_found = False
        item_selector = self.page.locator(".pluto-list__item").all()
        for item in item_selector:
            if item.is_visible() and input_text in item.inner_text().strip().lower():
                item.click()
                item_found = True
                break

        if not item_found:
            raise RuntimeError(f"Could not find channel '{input_text}' in dropdown")