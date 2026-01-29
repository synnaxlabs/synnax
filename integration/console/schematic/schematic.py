#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import json
from typing import TYPE_CHECKING, Any, Literal

import synnax as sy

from framework.utils import get_results_path

from ..page import ConsolePage

if TYPE_CHECKING:
    from console.console import Console
from .symbol import (
    Symbol,
    box_bottom,
    box_center_x,
    box_center_y,
    box_left,
    box_right,
    box_top,
)

PropertyDict = dict[str, float | str | bool]

SCHEMATIC_VERSION = "5.0.0"

AlignmentType = Literal[
    "vertical",
    "horizontal",
    "left",
    "right",
    "top",
    "bottom",
]

ALIGNMENT_EXTRACTORS = {
    "left": box_left,
    "right": box_right,
    "top": box_top,
    "bottom": box_bottom,
    "horizontal": box_center_x,
    "vertical": box_center_y,
}

DistributionType = Literal[
    "horizontal",
    "vertical",
]

RotationType = Literal[
    "clockwise",
    "counterclockwise",
]


class Schematic(ConsolePage):
    """Schematic page management interface"""

    page_type: str = "Schematic"
    pluto_label: str = ".react-flow__pane"

    @classmethod
    def open_from_search(cls, console: "Console", name: str) -> "Schematic":
        """Open an existing schematic by searching its name in the command palette.

        Args:
            console: Console instance.
            name: Name of the schematic to search for and open.

        Returns:
            Schematic instance for the opened schematic.
        """
        console.search_palette(name)

        schematic_pane = console.page.locator(cls.pluto_label)
        schematic_pane.first.wait_for(state="visible", timeout=5000)

        schematic = cls(console, name, _skip_create=True)
        schematic.pane_locator = schematic_pane.first
        return schematic

    def __init__(
        self,
        console: "Console",
        page_name: str,
        *,
        _skip_create: bool = False,
    ):
        """Initialize a Schematic page."""
        super().__init__(console, page_name, _skip_create=_skip_create)

    def create_symbol(self, symbol: Symbol) -> Symbol:
        """Add a symbol to the schematic and configure it.

        Args:
            symbol: Symbol instance to add to the schematic

        Returns:
            The configured symbol instance

        Example:
            valve = Valve(
                label="Pressure Valve",
                state_channel="press_vlv_state",
                command_channel="press_vlv_cmd"
            )
            configured_valve = schematic.create_symbol(valve)
            configured_valve.move(delta_x=-90, delta_y=-100)
        """

        symbol.create(self.page, self.console)
        return symbol

    def copy_link(self) -> str:
        """Copy link to the schematic via the toolbar link button.

        Returns:
            The copied link from clipboard (empty string if clipboard access fails)
        """
        self.console.notifications.close_all()
        self.console.layout.show_visualization_toolbar()
        link_button = self.page.locator(".pluto-icon--link").locator("..")
        link_button.click(timeout=5000)

        try:
            link: str = str(self.page.evaluate("navigator.clipboard.readText()"))
            return link
        except Exception as e:
            if "Timeout" in type(e).__name__:
                return ""
            raise RuntimeError(f"Error copying schematic link: {e}") from e

    def export_json(self) -> dict[str, Any]:
        """Export the schematic as a JSON file via the toolbar export button.

        The file is saved to the tests/results directory with the schematic name.

        Returns:
            The exported JSON content as a dictionary.
        """
        self.console.layout.show_visualization_toolbar()
        export_button = self.page.locator(".pluto-icon--export").locator("..")
        self.page.evaluate("delete window.showSaveFilePicker")

        with self.page.expect_download(timeout=5000) as download_info:
            export_button.click()

        download = download_info.value
        save_path = get_results_path(f"{self.page_name}.json")
        download.save_as(save_path)
        with open(save_path, "r") as f:
            result: dict[str, Any] = json.load(f)
            return result

    @staticmethod
    def assert_exported_json(exported: dict[str, Any]) -> None:
        """Assert that the exported JSON has a valid structure.

        Validates:
        - Root 'key' is a valid UUID format
        - Version matches SCHEMATIC_VERSION
        - Required keys exist: nodes, edges, props, viewport

        Args:
            exported: The exported JSON dictionary to validate.
        """
        assert "key" in exported, "Exported JSON should contain 'key'"
        try:
            uuid.UUID(exported["key"])
        except ValueError:
            raise AssertionError(
                f"Schematic key should be a valid UUID, got '{exported['key']}'"
            )

        assert "version" in exported, "Exported JSON should contain 'version'"
        assert exported["version"] == SCHEMATIC_VERSION, (
            f"Schematic version should be '{SCHEMATIC_VERSION}', "
            f"got '{exported['version']}'"
        )

        required_keys = ["nodes", "edges", "props", "viewport"]
        for key in required_keys:
            assert key in exported, f"Exported JSON should contain '{key}'"

    def get_node_count(self) -> int:
        """Get the number of nodes on the schematic canvas.

        Returns:
            Number of nodes currently on the canvas.
        """
        canvas = self.page.locator(".react-flow__pane")
        nodes = canvas.locator(".react-flow__node")
        return nodes.count()

    def wait_for_node(self, timeout: int = 5000) -> None:
        """Wait for at least one node to appear on the schematic canvas.

        Args:
            timeout: Maximum time to wait in milliseconds.
        """
        canvas = self.page.locator(".react-flow__pane")
        node = canvas.locator(".react-flow__node").last
        node.wait_for(state="visible", timeout=timeout)

    def get_control_legend_entries(self) -> list[str]:
        """Get list of writer names from the control legend.

        Returns:
            List of writer names currently shown in the legend.
            Returns empty list if legend is not visible.
        """
        legend = self.page.locator(".pluto-legend")
        if legend.count() == 0 or not legend.is_visible():
            return []

        entries = legend.locator(".pluto-legend-entry")
        return [entry.inner_text().strip() for entry in entries.all()]

    def align(
        self,
        symbols: list[Symbol],
        alignment: AlignmentType,
        tolerance: float | None = None,
    ) -> None:
        """
        Align multiple symbols using the schematic alignment controls.

        Args:
            symbols: List of symbols to align (must have at least 2 symbols)
            alignment: The alignment type to apply
            tolerance: Maximum allowed difference in pixels for assertion
                - Default: 3.0 for edge alignments (left/right/top/bottom)
                - Default: 15.0 for center alignments (horizontal/vertical)

        Raises:
            ValueError: If fewer than 2 symbols are provided
            AssertionError: If symbols are not properly aligned within tolerance after alignment
        """
        if len(symbols) < 2:
            raise ValueError("At least 2 symbols are required for alignment")

        alignment_icon_map = {
            "vertical": "pluto-icon--align-y-center",
            "horizontal": "pluto-icon--align-x-center",
            "left": "pluto-icon--align-left",
            "right": "pluto-icon--align-right",
            "top": "pluto-icon--align-top",
            "bottom": "pluto-icon--align-bottom",
        }

        icon_label = alignment_icon_map[alignment]

        symbols[0].click()
        for symbol in symbols[1:]:
            symbol.meta_click()

        alignment_button = self.page.locator(f"button svg[aria-label='{icon_label}']")
        alignment_button.wait_for(state="visible", timeout=2000)
        alignment_button.locator("..").click()

        # Deselect all symbols
        for symbol in symbols:
            symbol.meta_click()

    def distribute(
        self,
        symbols: list[Symbol],
        distribution: DistributionType,
        tolerance: float | None = None,
    ) -> None:
        """
        Distribute multiple symbols evenly using the schematic distribution controls.

        Args:
            symbols: List of symbols to distribute (must have at least 3 symbols)
            distribution: The distribution type to apply ('horizontal' or 'vertical')
            tolerance: Maximum allowed difference in spacing for assertion (default: 3.0 for horizontal, 20.0 for vertical)

        Raises:
            ValueError: If fewer than 3 symbols are provided (otherwise, nothing to distribute)
            AssertionError: If symbols are not evenly distributed within tolerance after distribution
        """
        if len(symbols) < 3:
            raise ValueError("At least 3 symbols are required for distribution")

        icon_label = (
            "pluto-icon--distribute-x"
            if distribution == "horizontal"
            else "pluto-icon--distribute-y"
        )

        symbols[0].click()
        for symbol in symbols[1:]:
            symbol.meta_click()

        # Click the distribution button
        distribution_button = self.page.locator(
            f"button svg[aria-label='{icon_label}']"
        )
        distribution_button.wait_for(state="visible", timeout=2000)
        distribution_button.locator("..").click()
        for symbol in symbols:
            symbol.meta_click()

    def rotate(
        self,
        symbols: list[Symbol],
        direction: RotationType,
        group: bool = False,
    ) -> None:
        """
        Rotate multiple symbols using the schematic rotation controls.

        Args:
            symbols: List of symbols to rotate (must have at least 1 symbol)
            direction: The rotation direction ('clockwise' or 'counterclockwise')
            group: If True, rotate symbols around their group center. If False, rotate each individually.

        Raises:
            ValueError: If no symbols are provided
        """
        if len(symbols) < 1:
            raise ValueError("At least 1 symbol is required for rotation")

        # Map direction names to icon aria-labels
        if group:
            rotation_icon_map = {
                "clockwise": "pluto-icon--rotate-around-center-cw",
                "counterclockwise": "pluto-icon--rotate-around-center-ccw",
            }
        else:
            rotation_icon_map = {
                "clockwise": "pluto-icon--rotate-group-cw",
                "counterclockwise": "pluto-icon--rotate-group-ccw",
            }

        icon_label = rotation_icon_map[direction]

        # Select all symbols
        symbols[0].click()
        for symbol in symbols[1:]:
            symbol.meta_click()

        # Click the rotation button
        rotation_button = self.page.locator(f"button svg[aria-label='{icon_label}']")
        rotation_button.wait_for(state="visible", timeout=2000)
        rotation_button.locator("..").click()

        # Wait for rotation animation/processing
        self.page.wait_for_timeout(100)

        # Deselect all symbols
        for symbol in symbols:
            symbol.meta_click()

    def connect_symbols(
        self,
        source_symbol: Symbol,
        source_handle: str,
        target_symbol: Symbol,
        target_handle: str,
    ) -> None:
        """Connect two symbols by dragging from source handle to target handle."""
        source_x, source_y = self.find_symbol_handle(source_symbol, source_handle)
        target_x, target_y = self.find_symbol_handle(target_symbol, target_handle)

        self.page.mouse.move(source_x, source_y)
        self.page.mouse.down()
        self.page.mouse.move(target_x, target_y, steps=10)
        self.page.mouse.up()

    def find_symbol_handle(self, symbol: Symbol, handle: str) -> tuple[float, float]:
        """Calculate the coordinates of a symbol's connection handle."""
        symbol_box = symbol.locator.bounding_box()
        if not symbol_box:
            raise RuntimeError(f"Could not get bounding box for symbol {symbol.label}")

        x, y, w, h = (
            symbol_box["x"],
            symbol_box["y"],
            symbol_box["width"],
            symbol_box["height"],
        )

        handle_positions = {
            "left": (x, y + h / 2),
            "right": (x + w, y + h / 2),
            "top": (x + w / 2, y),
            "bottom": (x + w / 2, y + h),
        }

        if handle not in handle_positions:
            valid_handles = ", ".join(handle_positions.keys())
            raise ValueError(
                f"Invalid handle: {handle}. Must be one of: {valid_handles}"
            )

        return handle_positions[handle]

    def set_authority(self, authority: int) -> None:
        """Set the control authority for the schematic page."""
        if authority > 255 or authority < 0:
            raise ValueError(
                f"Control Authority must be between 0 and 255, got {authority}"
            )
        self.console.notifications.close_all()
        self.console.click("Control")
        self.console.fill_input_field("Control Authority", str(authority))

    def set_properties(
        self,
        control_authority: int | None = None,
        show_control_legend: bool | None = None,
    ) -> None:
        """Set schematic properties."""
        self.console.notifications.close_all()
        self.console.click("Control")

        if control_authority is not None:
            if control_authority < 0 or control_authority > 255:
                raise ValueError(
                    f"Control Authority must be between 0 and 255, got {control_authority}"
                )
            self.console.fill_input_field("Control Authority", str(control_authority))
            self.page.keyboard.press("Enter")

        if show_control_legend is not None:
            legend_toggle = (
                self.page.locator("text=Show Control State Legend")
                .locator("..")
                .locator("input[type='checkbox']")
            )
            if legend_toggle.count() > 0:
                current_state = legend_toggle.is_checked()
                if current_state != show_control_legend:
                    legend_toggle.click()

    def get_control_status(self) -> bool:
        """Get whether control is currently acquired for this schematic."""
        control_button = (
            self.page.locator(".console-controls button")
            .filter(has=self.page.locator("svg.pluto-icon--circle"))
            .first
        )

        if control_button.count() > 0:
            class_attr = control_button.get_attribute("class") or ""
            has_filled = "pluto-btn--filled" in class_attr
            return has_filled

        return False

    def acquire_control(self) -> None:
        """Acquire control of the schematic if not already acquired."""
        if not self.get_control_status():
            control_button = (
                self.page.locator(".console-controls button.pluto-btn--outlined")
                .filter(has=self.page.locator("svg.pluto-icon--circle"))
                .first
            )
            if control_button.count() > 0:
                control_button.click()
                self.page.wait_for_selector(
                    ".console-controls button.pluto-btn--filled", timeout=2000
                )
            sy.sleep(0.1)  # Wait for Core update

    def release_control(self) -> None:
        """Release control of the schematic if currently acquired."""
        if self.get_control_status():
            control_button = (
                self.page.locator(".console-controls button.pluto-btn--filled")
                .filter(has=self.page.locator("svg.pluto-icon--circle"))
                .first
            )
            if control_button.count() > 0:
                control_button.click()
                self.page.wait_for_selector(
                    ".console-controls button.pluto-btn--outlined", timeout=1000
                )
            sy.sleep(0.1)  # Wait for Core update

    def get_edit_status(self) -> bool:
        """Get whether edit is currently enabled for this schematic."""
        edit_button = (
            self.page.locator(".console-controls button")
            .filter(has=self.page.locator("svg.pluto-icon--edit"))
            .first
        )

        if edit_button.count() == 0:
            edit_button = (
                self.page.locator(".console-controls button")
                .filter(has=self.page.locator("svg.pluto-icon--edit-off"))
                .first
            )

        if edit_button.count() > 0:
            class_attr = edit_button.get_attribute("class") or ""
            has_filled = "pluto-btn--filled" in class_attr
            return has_filled

        return False

    def enable_edit(self) -> None:
        """Enable edit for the schematic if not already enabled."""
        if not self.get_edit_status():
            edit_button = (
                self.page.locator(".console-controls button.pluto-btn--outlined")
                .filter(has=self.page.locator("svg.pluto-icon--edit"))
                .first
            )
            if edit_button.count() > 0:
                edit_button.click()
                self.page.wait_for_selector(
                    ".console-controls button.pluto-btn--filled", timeout=2000
                )
        sy.sleep(0.1)

    def disable_edit(self) -> None:
        """Disable edit for the schematic if currently enabled."""
        if self.get_edit_status():
            edit_button = (
                self.page.locator(".console-controls button.pluto-btn--filled")
                .filter(has=self.page.locator("svg.pluto-icon--edit"))
                .first
            )
            if edit_button.count() > 0:
                edit_button.click()
                self.page.wait_for_selector(
                    ".console-controls button.pluto-btn--outlined", timeout=2000
                )
        sy.sleep(0.1)

    def get_properties(self) -> tuple[int, bool]:
        """Get the current properties of the schematic.

        Returns:
            Tuple of (control_authority, show_control_legend)
        """
        self.console.notifications.close_all()
        self.console.click("Control")

        control_authority = int(self.console.get_input_field("Control Authority"))

        try:
            show_control_legend = self.console.get_toggle("Show Control State Legend")
        except Exception as e:
            if "Timeout" in type(e).__name__:
                show_control_legend = True
            else:
                raise RuntimeError(
                    f"Error getting show control legend toggle: {e}"
                ) from e

        return (control_authority, show_control_legend)

    @property
    def control_legend_visible(self) -> bool:
        """Check if the control state legend is visible."""
        legend = self.page.locator(".pluto-legend")
        return legend.count() > 0 and legend.is_visible()
