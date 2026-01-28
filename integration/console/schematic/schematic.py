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
import math
from typing import TYPE_CHECKING, Any, Literal

import synnax as sy
from playwright.sync_api import FloatRect

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
    def open_from_search(
        cls, client: sy.Synnax, console: "Console", name: str
    ) -> "Schematic":
        """Open an existing schematic by searching its name in the command palette.

        Args:
            client: Synnax client instance.
            console: Console instance.
            name: Name of the schematic to search for and open.

        Returns:
            Schematic instance for the opened schematic.
        """
        console.search_palette(name)

        schematic_pane = console.page.locator(cls.pluto_label)
        schematic_pane.first.wait_for(state="visible", timeout=5000)

        schematic = cls.__new__(cls)
        schematic.client = client
        schematic.console = console
        schematic.page = console.page
        schematic.page_name = name
        schematic.pane_locator = schematic_pane.first
        return schematic

    def __init__(self, client: sy.Synnax, console: "Console", page_name: str):
        """Initialize a Schematic page."""
        super().__init__(client, console, page_name)

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
        except Exception:
            return ""

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

    def assert_control_legend_contains(self, writer_name: str) -> None:
        """Assert that the control legend shows the specified writer.

        Args:
            writer_name: Name of the writer to check for in the legend.

        Raises:
            AssertionError: If writer is not in the legend.
        """
        entries = self.get_control_legend_entries()
        assert writer_name in entries, (
            f"Expected writer '{writer_name}' in control legend!\n"
            f"Actual entries: {entries}"
        )

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

        self.assert_alignment(symbols, alignment, tolerance)

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

        self.assert_distribution(symbols, distribution, tolerance)

    def rotate(
        self,
        symbols: list[Symbol],
        direction: RotationType,
        group: bool = False,
        tolerance: float = 5.0,
    ) -> None:
        """
        Rotate multiple symbols using the schematic rotation controls.

        Args:
            symbols: List of symbols to rotate (must have at least 1 symbol)
            direction: The rotation direction ('clockwise' or 'counterclockwise')
            group: If True, rotate symbols around their group center. If False, rotate each individually.
            tolerance: Maximum allowed difference in pixels for assertion (default: 10.0)

        Raises:
            ValueError: If no symbols are provided
            AssertionError: If rotation was not applied correctly
        """
        if len(symbols) < 1:
            raise ValueError("At least 1 symbol is required for rotation")

        # Capture initial positions before rotation
        initial_positions = [symbol.position for symbol in symbols]

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

        self.assert_rotation(symbols, initial_positions, direction, group, tolerance)

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

    def assert_setpoint(
        self, setpoint_symbol: Symbol, channel_name: str, value: float
    ) -> None:
        """Assert that setting the setpoint value results in the expected value in the Core."""
        setpoint_symbol.set_value(value)
        sy.sleep(0.2)  # Wait for the value to be set
        actual_value = self.get_value(channel_name)
        assert (
            actual_value == value
        ), f"Setpoint value mismatch!\nActual: {actual_value}\nExpected: {value}"

    def assert_symbol_properties(
        self, symbol: Symbol, expected_props: PropertyDict
    ) -> None:
        actual_props = symbol.get_properties()
        assert (
            actual_props == expected_props
        ), f"Props mismatch!\nActual: {actual_props}\nExpected: {expected_props}"

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
        except Exception:
            show_control_legend = True  # Default if not found

        return (control_authority, show_control_legend)

    def assert_properties(
        self, control_authority: int = 1, show_control_legend: bool = True
    ) -> None:
        """Assert the schematic properties match expected values."""
        if control_authority < 0 or control_authority > 255:
            raise ValueError(
                f"Control Authority must be between 0 and 255, got {control_authority}"
            )

        actual_authority, actual_legend = self.get_properties()

        assert actual_authority == control_authority, (
            f"Control Authority mismatch!\n"
            f"Actual: {actual_authority}\n"
            f"Expected: {control_authority}"
        )

        assert actual_legend == show_control_legend, (
            f"Show Control Legend mismatch!\n"
            f"Actual: {actual_legend}\n"
            f"Expected: {show_control_legend}"
        )

    def assert_control_status(self, expected: bool) -> None:
        """Assert the control status matches the expected value."""
        actual = self.get_control_status()
        assert actual == expected, (
            f"Control status mismatch!\n" f"Actual: {actual}\n" f"Expected: {expected}"
        )

    def assert_control_legend_visible(self, expected: bool) -> None:
        """Assert the control state legend visibility matches the expected value."""
        legend = self.page.locator(".pluto-legend")
        is_visible = legend.count() > 0 and legend.is_visible()
        assert is_visible == expected, (
            f"Control legend visibility mismatch!\n"
            f"Actual: {is_visible}\n"
            f"Expected: {expected}"
        )

    def assert_edit_status(self, expected: bool) -> None:
        """Assert the edit status matches the expected value."""
        actual = self.get_edit_status()
        assert actual == expected, (
            f"Edit status mismatch!\n" f"Actual: {actual}\n" f"Expected: {expected}"
        )

    def assert_alignment(
        self,
        symbols: list[Symbol],
        alignment: AlignmentType,
        tolerance: float | None = None,
    ) -> None:
        """
        Assert that all symbols are aligned along the specified axis.

        Args:
            symbols: List of symbols to check for alignment
            alignment: The alignment axis to check
                - 'left', 'right', 'top', 'bottom': edge alignment (uses bounding box edges)
                - 'horizontal': horizontal center alignment (uses handle midpoints)
                - 'vertical': vertical center alignment (uses handle midpoints)
            tolerance: Maximum allowed difference in pixels
                - Default: 3.0 for edge alignments (left/right/top/bottom)
                - Default: 15.0 for center alignments (horizontal/vertical) due to handle-based logic

        Raises:
            AssertionError: If symbols are not properly aligned within tolerance
        """
        if len(symbols) < 2:
            raise ValueError("At least 2 symbols are required for alignment assertion")

        positions = [symbol.position for symbol in symbols]

        if tolerance is None:
            if alignment in ["horizontal", "vertical"]:
                tolerance = 15.0  # Larger tolerance for handle-based center alignment
            else:
                tolerance = 3.0  # Strict tolerance for edge alignment

        extractor = ALIGNMENT_EXTRACTORS[alignment]
        first_coord = extractor(positions[0])

        # Check that all symbols are aligned within tolerance
        for i, pos in enumerate(positions):
            coord = extractor(pos)
            diff = abs(coord - first_coord)
            assert diff <= tolerance, (
                f"Symbol {i} ('{symbols[i].label}') is not aligned on {alignment}!\n"
                f"Expected: {first_coord} (±{tolerance})\n"
                f"Actual: {coord}\n"
                f"Difference: {diff}\n"
                f"All symbols: {[s.label for s in symbols]}"
            )

    def assert_distribution(
        self,
        symbols: list[Symbol],
        distribution: DistributionType,
        tolerance: float | None = None,
    ) -> None:
        """
        Assert that symbols are evenly distributed along the specified axis.

        Args:
            symbols: List of symbols to check for distribution (must have at least 3 symbols)
            distribution: The distribution axis to check ('horizontal' or 'vertical')
            tolerance: Maximum allowed difference in spacing (default: 3.0 for horizontal, 20.0 for vertical)

        Raises:
            ValueError: If fewer than 3 symbols are provided
            AssertionError: If symbols are not evenly distributed within tolerance
        """
        if len(symbols) < 3:
            raise ValueError(
                "At least 3 symbols are required for distribution assertion"
            )

        # Use different default tolerances for horizontal vs vertical
        # Vertical needs higher tolerance due to varying symbol decorations (labels, control chips)
        if tolerance is None:
            tolerance = 3.0 if distribution == "horizontal" else 20.0

        positions = [symbol.position for symbol in symbols]

        if distribution == "horizontal":
            # Sort by left edge position
            sorted_data = sorted(zip(symbols, positions), key=lambda x: box_left(x[1]))
            sorted_symbols = [item[0] for item in sorted_data]
            sorted_positions = [item[1] for item in sorted_data]

            # Calculate gaps between consecutive symbols (right edge to left edge)
            gaps = []
            for i in range(len(sorted_positions) - 1):
                current_right = box_right(sorted_positions[i])
                next_left = box_left(sorted_positions[i + 1])
                gap = next_left - current_right
                gaps.append(gap)

            # Check that all gaps are equal within tolerance
            first_gap = gaps[0]
            for i, gap in enumerate(gaps):
                diff = abs(gap - first_gap)
                assert diff <= tolerance, (
                    f"Horizontal gap {i} is not equal to first gap!\n"
                    f"Expected gap: {first_gap} (±{tolerance})\n"
                    f"Actual gap: {gap}\n"
                    f"Difference: {diff}\n"
                    f"Gap between '{sorted_symbols[i].label}' and '{sorted_symbols[i + 1].label}'\n"
                    f"All symbols (left to right): {[s.label for s in sorted_symbols]}"
                )

        else:  # vertical
            # Sort by top edge position
            sorted_data = sorted(zip(symbols, positions), key=lambda x: box_top(x[1]))
            sorted_symbols = [item[0] for item in sorted_data]
            sorted_positions = [item[1] for item in sorted_data]

            # Calculate gaps between consecutive symbols (bottom edge to top edge)
            gaps = []
            for i in range(len(sorted_positions) - 1):
                current_bottom = box_bottom(sorted_positions[i])
                next_top = box_top(sorted_positions[i + 1])
                gap = next_top - current_bottom
                gaps.append(gap)

            # Check that all gaps are equal within tolerance
            first_gap = gaps[0]
            for i, gap in enumerate(gaps):
                diff = abs(gap - first_gap)
                assert diff <= tolerance, (
                    f"Vertical gap {i} is not equal to first gap!\n"
                    f"Expected gap: {first_gap} (±{tolerance})\n"
                    f"Actual gap: {gap}\n"
                    f"Difference: {diff}\n"
                    f"Gap between '{sorted_symbols[i].label}' and '{sorted_symbols[i + 1].label}'\n"
                    f"All symbols (top to bottom): {[s.label for s in sorted_symbols]}"
                )

    def assert_rotation(
        self,
        symbols: list[Symbol],
        initial_positions: list[FloatRect],
        direction: RotationType,
        group: bool = False,
        tolerance: float = 3.0,
    ) -> None:
        """
        Assert that symbols have been rotated correctly.

        For individual rotation (group=False):
            - Verifies that symbol dimensions have changed (rotation affects visual appearance)
            - For symbols with rotation capability, the visual rotation should be apparent

        For group rotation (group=True):
            - Verifies that symbols have moved to new positions around the group center
            - Checks that the spatial arrangement has changed according to 90-degree rotation

        Args:
            symbols: List of symbols that were rotated
            initial_positions: List of Box objects before rotation (from symbol.position)
            direction: The rotation direction ('clockwise' or 'counterclockwise')
            group: If True, expects group rotation. If False, expects individual rotation.
            tolerance: Maximum allowed difference in pixels for position comparisons (default: 3.0)

        Raises:
            ValueError: If the number of symbols doesn't match initial_positions
            AssertionError: If rotation was not applied correctly
        """
        if len(symbols) != len(initial_positions):
            raise ValueError(
                f"Number of symbols ({len(symbols)}) must match "
                f"number of initial positions ({len(initial_positions)})"
            )

        current_positions = [symbol.position for symbol in symbols]

        self._assert_individual_rotation_dimensions(
            symbols, initial_positions, current_positions, tolerance
        )

        if group:
            self._assert_group_rotation_transform(
                symbols, initial_positions, current_positions, direction, tolerance
            )
        else:
            self._assert_individual_rotation_ordering(
                symbols, initial_positions, current_positions
            )

    def _assert_individual_rotation_dimensions(
        self,
        symbols: list[Symbol],
        initial_positions: list[FloatRect],
        current_positions: list[FloatRect],
        tolerance: float,
    ) -> None:
        """Assert that symbol dimensions changed correctly after individual rotation.

        For rotatable symbols: width and height should be swapped.
        For non-rotatable symbols: dimensions should remain unchanged.
        """
        for i, symbol in enumerate(symbols):
            initial_pos = initial_positions[i]
            current_pos = current_positions[i]

            if symbol.rotatable:
                # For rotatable symbols, dimensions should be swapped
                width_diff = abs(current_pos["width"] - initial_pos["height"])
                height_diff = abs(current_pos["height"] - initial_pos["width"])

                assert width_diff <= tolerance and height_diff <= tolerance, (
                    f"Symbol {i} ({type(symbol).__name__}) dimensions not swapped after rotation!\n"
                    f"Initial: width={initial_pos['width']:.1f}, height={initial_pos['height']:.1f}\n"
                    f"Current: width={current_pos['width']:.1f}, height={current_pos['height']:.1f}\n"
                    f"Expected: width={initial_pos['height']:.1f}, height={initial_pos['width']:.1f}\n"
                    f"Difference: width_diff={width_diff:.1f}, height_diff={height_diff:.1f}\n"
                    f"Tolerance: {tolerance}px"
                )
            else:
                # For non-rotatable symbols, dimensions should remain the same
                width_diff = abs(current_pos["width"] - initial_pos["width"])
                height_diff = abs(current_pos["height"] - initial_pos["height"])

                assert width_diff <= tolerance and height_diff <= tolerance, (
                    f"Symbol {i} ({type(symbol).__name__}) dimensions changed after rotation attempt!\n"
                    f"This symbol type cannot be rotated.\n"
                    f"Initial: width={initial_pos['width']:.1f}, height={initial_pos['height']:.1f}\n"
                    f"Current: width={current_pos['width']:.1f}, height={current_pos['height']:.1f}\n"
                    f"Expected: width={initial_pos['width']:.1f}, height={initial_pos['height']:.1f}\n"
                    f"Difference: width_diff={width_diff:.1f}, height_diff={height_diff:.1f}\n"
                    f"Tolerance: {tolerance}px"
                )

    def _assert_individual_rotation_ordering(
        self,
        symbols: list[Symbol],
        initial_positions: list[FloatRect],
        current_positions: list[FloatRect],
    ) -> None:
        """Assert that relative ordering is preserved after individual rotation.

        Only checks ordering for symbols that are clearly separated (>15px).
        Symbols close together are allowed to swap due to center position shifts during rotation.
        """
        ordering_separation_threshold = 15.0  # pixels

        def check_axis_ordering(alignment: AlignmentType) -> None:
            extractor = ALIGNMENT_EXTRACTORS[alignment]
            violations: list[dict[str, int | float]] = []
            for i in range(len(symbols)):
                for j in range(i + 1, len(symbols)):
                    initial_i_val = extractor(initial_positions[i])
                    initial_j_val = extractor(initial_positions[j])
                    current_i_val = extractor(current_positions[i])
                    current_j_val = extractor(current_positions[j])

                    initial_diff = initial_j_val - initial_i_val
                    current_diff = current_j_val - current_i_val

                    if abs(initial_diff) > ordering_separation_threshold:
                        if (initial_diff > 0 and current_diff < 0) or (
                            initial_diff < 0 and current_diff > 0
                        ):
                            violations.append(
                                {
                                    "i": i,
                                    "j": j,
                                    "initial_i": initial_i_val,
                                    "initial_j": initial_j_val,
                                    "current_i": current_i_val,
                                    "current_j": current_j_val,
                                }
                            )

            if violations:
                msg = f"{alignment.capitalize()} ordering changed for well-separated symbols!\n"
                for v in violations:
                    i_idx = int(v["i"])
                    j_idx = int(v["j"])
                    msg += f"  Symbols {i_idx} ({type(symbols[i_idx]).__name__}) and {j_idx} ({type(symbols[j_idx]).__name__}) swapped:\n"
                    msg += f"    Initial: {v['initial_i']:.1f} vs {v['initial_j']:.1f} (diff: {abs(float(v['initial_j']) - float(v['initial_i'])):.1f}px)\n"
                    msg += f"    Current: {v['current_i']:.1f} vs {v['current_j']:.1f} (diff: {abs(float(v['current_j']) - float(v['current_i'])):.1f}px)\n"
                raise AssertionError(msg)

        check_axis_ordering("horizontal")
        check_axis_ordering("vertical")

    def _assert_group_rotation_transform(
        self,
        symbols: list[Symbol],
        initial_positions: list[FloatRect],
        current_positions: list[FloatRect],
        direction: RotationType,
        tolerance: float,
    ) -> None:
        """Assert that group rotation correctly transformed symbol positions.

        Uses angular displacement to verify rotation. Each symbol should have
        rotated by approximately ±90 degrees around the group center.
        """
        # Calculate initial center
        initial_center_x = sum(box_center_x(pos) for pos in initial_positions) / len(
            initial_positions
        )
        initial_center_y = sum(box_center_y(pos) for pos in initial_positions) / len(
            initial_positions
        )

        # Calculate current center
        current_center_x = sum(box_center_x(pos) for pos in current_positions) / len(
            current_positions
        )
        current_center_y = sum(box_center_y(pos) for pos in current_positions) / len(
            current_positions
        )

        expected_angle = 90.0 if direction == "clockwise" else -90.0
        angular_tolerance = 17.0

        for i in range(len(symbols)):

            init_x = box_center_x(initial_positions[i]) - initial_center_x
            init_y = box_center_y(initial_positions[i]) - initial_center_y

            curr_x = box_center_x(current_positions[i]) - current_center_x
            curr_y = box_center_y(current_positions[i]) - current_center_y

            initial_angle = math.degrees(math.atan2(init_y, init_x))
            current_angle = math.degrees(math.atan2(curr_y, curr_x))

            angular_displacement = current_angle - initial_angle
            if angular_displacement > 180:
                angular_displacement -= 360
            elif angular_displacement < -180:
                angular_displacement += 360

            angle_diff = abs(angular_displacement - expected_angle)

            assert angle_diff <= angular_tolerance, (
                f"Symbol {i} ({type(symbols[i]).__name__}) not rotated correctly in group!\n"
                f"Direction: {direction}\n"
                f"Initial angle: {initial_angle:.1f}°\n"
                f"Current angle: {current_angle:.1f}°\n"
                f"Angular displacement: {angular_displacement:.1f}°\n"
                f"Expected: {expected_angle:.1f}° ± {angular_tolerance:.1f}°\n"
                f"Difference: {angle_diff:.1f}°\n"
                f"Initial center: ({initial_center_x:.1f}, {initial_center_y:.1f})\n"
                f"Current center: ({current_center_x:.1f}, {current_center_y:.1f})"
            )
