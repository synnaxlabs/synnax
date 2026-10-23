#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator, Page


def pick(page: Page, swatch: Locator, hex_color: str) -> None:
    """Opens the color picker of a swatch, enters a color, and closes the picker.

    :param swatch: The swatch that opens the picker.
    :param hex_color: The color as hex digits, with or without a leading "#".
    """
    swatch.click()
    picker = page.locator(".pluto-color-picker")
    picker.wait_for(state="visible", timeout=2000)
    hex_input = picker.get_by_label("Hex", exact=True)
    hex_input.click(click_count=3)
    hex_input.fill(hex_color.lstrip("#"))
    page.keyboard.press("Enter")
    # Escape in a text box only leaves the box, so leave it first.
    hex_input.blur()
    page.keyboard.press("Escape")
    picker.wait_for(state="hidden", timeout=2000)


def pick_auto(page: Page, swatch: Locator) -> None:
    """Opens the color picker of an optional color's swatch, picks Auto, and closes
    the picker.

    :param swatch: The swatch that opens the picker.
    """
    swatch.click()
    picker = page.locator(".pluto-color-picker")
    picker.wait_for(state="visible", timeout=2000)
    picker.get_by_label("Auto", exact=True).click()
    page.keyboard.press("Escape")
    picker.wait_for(state="hidden", timeout=2000)
