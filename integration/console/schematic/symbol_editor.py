#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Symbol Editor modal client for creating and editing custom symbols."""

from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.layout import LayoutClient


class SymbolEditor:
    """Client for interacting with the Symbol Editor modal."""

    def __init__(self, layout: LayoutClient):
        self.page = layout.page
        self.layout = layout

    @property
    def modal(self) -> Locator:
        """Get the symbol editor modal locator."""
        return self.page.locator(".console-schematic__symbol-create-layout")

    @property
    def drop_zone(self) -> Locator:
        """Get the SVG drop zone locator."""
        return self.page.get_by_text(
            "Click to select an SVG file or drag and drop it here"
        )

    def wait_for_open(self) -> None:
        """Wait for the editor modal to open."""
        try:
            self.drop_zone.wait_for(state="visible", timeout=3000)
        except PlaywrightTimeoutError:
            self.wait_for_form_visible()

    def wait_for_form_visible(self) -> None:
        """Wait for the form fields to appear (after SVG upload)."""
        self.page.locator("input[placeholder='Symbol Name']").wait_for(
            state="visible", timeout=5000
        )

    def wait_for_closed(self) -> None:
        """Wait for the editor modal to close."""
        self.modal.wait_for(state="hidden", timeout=5000)

    def set_name(self, name: str) -> None:
        """Set the symbol name."""
        name_input = self.page.locator("input[placeholder='Symbol Name']")
        name_input.fill(name)

    def upload_svg(self, svg_path: str) -> None:
        """Upload SVG file by directly calling the component's content handler.

        This must be called before setting the name or other properties,
        as the form only appears after an SVG is uploaded.

        Args:
            svg_path: Path to the SVG file to upload.
        """
        import os

        with open(svg_path, "r", encoding="utf-8") as f:
            svg_content = f.read()

        filename = os.path.basename(svg_path)
        name_without_ext = os.path.splitext(filename)[0]
        proper_name = " ".join(
            word.capitalize()
            for word in name_without_ext.replace("_", " ").replace("-", " ").split()
        )

        # SY-3670
        self.page.evaluate(
            """([svgContent, properName]) => {
                // Access React fiber to call FileDrop's onContentsChange prop directly
                const dropZone = document.querySelector('.console-file-drop');
                if (!dropZone) throw new Error('Drop zone not found');

                const fiberKey = Object.keys(dropZone).find(k => k.startsWith('__reactFiber$'));
                if (!fiberKey) throw new Error('React fiber not found');

                let fiber = dropZone[fiberKey];
                let onContentsChange = null;

                while (fiber) {
                    if (fiber.memoizedProps?.onContentsChange) {
                        onContentsChange = fiber.memoizedProps.onContentsChange;
                        break;
                    }
                    fiber = fiber.return;
                }

                if (!onContentsChange) throw new Error('onContentsChange not found');
                onContentsChange(svgContent, properName);
            }""",
            [svg_content, proper_name],
        )
        self.wait_for_form_visible()

    def _region_rows(self) -> Locator:
        """Locate the region rows (the only list items carrying color swatches)."""
        return self.page.locator(".pluto-list__item").filter(
            has=self.page.locator(".pluto-color-swatch")
        )

    def region_count(self) -> int:
        """Return the number of detected regions."""
        return self._region_rows().count()

    def _set_region_color(
        self, swatch_index: int, hex_color: str, region_index: int
    ) -> None:
        swatch = (
            self._region_rows()
            .nth(region_index)
            .locator(".pluto-color-swatch")
            .nth(swatch_index)
        )
        swatch.click()
        color_picker = self.page.locator(".pluto-color-picker-container")
        color_picker.wait_for(state="visible", timeout=2000)
        hex_input = color_picker.locator(".sketch-picker input").first
        hex_input.click(click_count=3)
        hex_input.type(hex_color.replace("#", ""))
        self.page.keyboard.press("Enter")
        self.page.keyboard.press("Escape")
        color_picker.wait_for(state="hidden", timeout=2000)

    def set_region_stroke_color(self, hex_color: str, region_index: int = 0) -> None:
        """Set the stroke color for a region (the first swatch in the row).

        Args:
            hex_color: Hex color string (e.g., "#FF0000" or "FF0000").
            region_index: Index of the region (0-based).
        """
        self._set_region_color(0, hex_color, region_index)

    def set_region_fill_color(self, hex_color: str, region_index: int = 0) -> None:
        """Set the fill color for a region (the second swatch in the row).

        Args:
            hex_color: Hex color string (e.g., "#FF0000" or "FF0000").
            region_index: Index of the region (0-based).
        """
        self._set_region_color(1, hex_color, region_index)

    def get_preview_fill(self, selector: str) -> str:
        """Return the computed fill (e.g. "rgb(255, 0, 0)") of a rendered preview shape.

        Args:
            selector: A CSS selector resolved within the rendered preview SVG.
        """
        el = self.page.locator(f".console-preview svg {selector}").first
        el.wait_for(state="attached", timeout=5000)
        return str(el.evaluate("(el) => getComputedStyle(el).fill"))

    def assert_preview_fill(
        self, selector: str, expected_rgb: str, timeout: int = 5000
    ) -> None:
        """Wait until a rendered preview shape resolves to the expected computed fill.

        Args:
            selector: A CSS selector resolved within the rendered preview SVG.
            expected_rgb: The expected computed fill, e.g. "rgb(255, 0, 0)".
        """
        self.page.wait_for_function(
            """([sel, exp]) => {
                const el = document.querySelector('.console-preview svg ' + sel);
                return el != null && getComputedStyle(el).fill === exp;
            }""",
            arg=[selector, expected_rgb],
            timeout=timeout,
        )

    def assert_preview_width(self, expected: float, timeout: int = 5000) -> None:
        """Wait until the rendered preview SVG reaches the expected width attribute."""
        self.page.wait_for_function(
            """(exp) => {
                const svg = document.querySelector('.console-preview svg');
                if (svg == null) return false;
                return Math.abs(parseFloat(svg.getAttribute('width')) - exp) < 0.5;
            }""",
            arg=expected,
            timeout=timeout,
        )

    def add_handle(self) -> None:
        """Add a new connection handle."""
        handles_header = self.page.locator(".pluto-header").filter(has_text="Handles")
        add_handle_btn = handles_header.locator("button.pluto-btn--outlined").first
        add_handle_btn.click()

    def set_default_scale(self, scale_percent: int) -> None:
        """Set the default scale percentage for the symbol.

        Args:
            scale_percent: Scale percentage (e.g., 100 for 100%, 150 for 150%).
                           Valid range is 5 to 1001.
        """
        properties_header = self.page.locator(".pluto-header").filter(
            has_text="Properties"
        )
        properties_section = properties_header.locator("..").locator("..")
        scale_input = properties_section.locator("input[type='text']").first
        scale_input.click(click_count=3)
        scale_input.fill(str(scale_percent))
        self.page.keyboard.press("Enter")

    def set_state(self, state: str) -> None:
        """Set the symbol state type (Static or Actuator).

        Args:
            state: The state type - "Static" or "Actuator"
        """
        static_btn = self.modal.get_by_text("Static", exact=True)
        static_btn.wait_for(state="visible", timeout=2000)
        static_btn.click()
        self.layout.select_from_dropdown(state, placeholder="variants")

    def save(self) -> None:
        """Click Save/Create button to save the symbol."""
        save_btn = self.page.get_by_role("button", name="Create", exact=True)
        if save_btn.count() == 0:
            save_btn = self.page.get_by_role("button", name="Save", exact=True)
        save_btn.dispatch_event("click")
        self.wait_for_closed()
