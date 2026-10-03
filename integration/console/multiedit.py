#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The Colors section that the schematic and table toolbars show for a selection of
several elements."""

from playwright.sync_api import Locator, Page

from console import color


class Colors:
    """The Colors section of a multi-element toolbar form."""

    def __init__(self, page: Page) -> None:
        self.page = page
        self.section = page.locator(".console-multiedit-colors")

    def field(self, label: str) -> Locator:
        """The hex text input of the Stroke, Fill, or Text control."""
        return self.section.locator(f'input[aria-label="{label}"]')

    def set(self, label: str, hex_digits: str) -> None:
        """Type a hex color, without the leading #, into a control."""
        self.field(label).fill(hex_digits)

    def selection_swatches(self) -> Locator:
        """One swatch per color the selection stores, in the Selection control."""
        return self.section.locator(
            ".pluto-input__item:has(> label:text-is('Selection')) .pluto-color-swatch"
        )

    def selection_colors(self) -> list[str]:
        """The CSS color of each Selection swatch, such as "rgba(255, 0, 0, 1)"."""
        return [
            swatch.evaluate("e => e.style.getPropertyValue('--pluto-swatch-color')")
            for swatch in self.selection_swatches().all()
        ]

    def recolor(self, swatch: Locator, hex_digits: str) -> None:
        """Pick a new color for every element that holds the color of `swatch`."""
        color.pick(self.page, swatch, hex_digits)
