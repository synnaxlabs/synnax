#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING

import synnax as sy
from playwright.sync_api import Page

from .console_page import ConsolePage

if TYPE_CHECKING:
    from ..console import Console


class Channels(ConsolePage):
    """Console channels management interface"""

    def __init__(self, page: Page, console: "Console"):
        super().__init__(page, console)

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
        self.console.command_palette("Create a Channel")

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
            self.console._select_from_dropdown("Data Type", data_type_str)

            # Set index
            self.console._select_from_dropdown("Index", index)

        # Create the channel
        self.page.get_by_role("button", name="Create", exact=True).click()
        return True
