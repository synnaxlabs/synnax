#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator

import synnax as sy
from console.layout import LayoutClient
from console.page import ConsolePage

DATA_ROW_SELECTOR = ".pluto-table__row:not(.pluto-table__col-resizer)"


class Table(ConsolePage):
    """Table page management interface"""

    page_type: str = "Table"
    pluto_label: str = ".pluto-table"

    def __init__(
        self,
        layout: LayoutClient,
        client: sy.Synnax,
        page_name: str,
        *,
        pane_locator: Locator,
    ) -> None:
        """Initialize a Table page wrapper (see ConsolePage.__init__ for details)."""
        super().__init__(layout, client, page_name, pane_locator=pane_locator)

    def set_cell_channel(self, channel_name: str, row: int = 0, col: int = 0) -> None:
        """Set a cell to display a channel's telemetry value.

        Args:
            channel_name: Name of the channel to display
            row: Row index (0-based)
            col: Column index (0-based)
        """
        self._get_cell(row, col).click()
        self.set_toolbar_variant("Value")
        self.page.get_by_text("Telemetry").click()
        self.layout.click_btn("Input Channel")
        self.layout.select_from_dropdown(channel_name)

    def get_cell_channel(self, row: int = 0, col: int = 0) -> str:
        """Get the channel name displayed in a cell.

        Args:
            row: Row index (0-based)
            col: Column index (0-based)

        Returns:
            The channel name or empty string if not set
        """
        self._select_cell(row, col)
        self.page.get_by_text("Telemetry").click()
        channel_btn = (
            self.page.locator("text=Input Channel")
            .locator("..")
            .locator("button")
            .first
        )
        return channel_btn.inner_text().strip()

    def get_cell_text(self, row: int = 0, col: int = 0) -> str:
        """Get the text content of a text cell.

        Args:
            row: Row index (0-based)
            col: Column index (0-based)

        Returns:
            The text value of the cell
        """
        self._select_cell(row, col)
        text_input = self.page.locator("text=Text").locator("..").locator("input").first
        return text_input.input_value().strip()

    def has_text(self, text: str, row: int = 0, col: int = 0) -> bool:
        """Check if a text cell contains the given text.

        Args:
            text: Text to check for
            row: Row index (0-based)
            col: Column index (0-based)

        Returns:
            True if the cell text matches
        """
        return self.get_cell_text(row, col) == text

    def has_channel(self, channel_name: str, row: int = 0, col: int = 0) -> bool:
        """Check if a channel is shown in a cell.

        Args:
            channel_name: Name of the channel to check
            row: Row index (0-based)
            col: Column index (0-based)

        Returns:
            True if the channel is displayed in the cell
        """
        return channel_name in self.get_cell_channel(row, col)

    def add_row(self) -> None:
        """Add a new row to the table by clicking the add-row button."""
        control = self.page.locator(".pluto-table-frame__add-row").first
        control.wait_for(state="visible", timeout=5000)
        control.locator("button").last.click()

    def add_column(self) -> None:
        """Add a new column to the table by clicking the add-column button."""
        control = self.page.locator(".pluto-table-frame__add-col").first
        control.wait_for(state="visible", timeout=5000)
        control.locator("button").last.click()

    def delete_row(self, row: int, col: int = 0) -> None:
        """Delete a row via context menu on a cell.

        Args:
            row: Row index (0-based)
            col: Column index (0-based) of the cell to right-click
        """
        cell = self._get_cell(row, col)
        self.ctx_menu.action(cell, f"Delete row {row + 1}")

    def delete_column(self, col: int, row: int = 0) -> None:
        """Delete a column via context menu on a cell.

        Args:
            col: Column index (0-based) of the column to delete
            row: Row index (0-based) of the cell to right-click
        """
        cell = self._get_cell(row, col)
        letter = chr(ord("A") + col)
        self.ctx_menu.action(cell, f"Delete column {letter}")

    def set_redline(self, row: int, col: int, lower: float, upper: float) -> None:
        """Configure redline bounds on a value cell.

        The cell must already be set to "Value" variant with a channel configured.

        Args:
            row: Row index (0-based)
            col: Column index (0-based)
            lower: Lower redline bound
            upper: Upper redline bound
        """
        self._select_cell(row, col)
        self.page.get_by_text("Redline").click()
        self.layout.fill_input_field("Lower", str(lower))
        self.layout.fill_input_field("Upper", str(upper))

    def get_redline(self, row: int, col: int) -> tuple[str, str]:
        """Get the current redline bounds from a value cell.

        Args:
            row: Row index (0-based)
            col: Column index (0-based)

        Returns:
            Tuple of (lower_bound, upper_bound) as strings
        """
        self._select_cell(row, col)
        self.page.get_by_text("Redline").click()
        lower = self.layout.get_input_field("Lower")
        upper = self.layout.get_input_field("Upper")
        return (lower, upper)

    def _select_cell(self, row: int, col: int) -> None:
        """Focus the tab, click a cell, and open the visualization toolbar."""
        self.layout.get_tab(self.page_name).click()
        self._get_cell(row, col).click()
        self.layout.show_visualization_toolbar()

    def _get_cell(self, row: int, col: int) -> Locator:
        """Get a locator for a specific cell in the table.

        Args:
            row: Row index (0-based)
            col: Column index (0-based)

        Returns:
            Locator for the cell element
        """
        cells = self.page.locator(".pluto-table__cell")
        cell_index = row * self.get_column_count() + col
        return cells.nth(cell_index)

    def get_row_count(self) -> int:
        """Get the number of data rows in the table (excludes the column resizer row)."""
        self.page.locator(DATA_ROW_SELECTOR).first.wait_for(
            state="visible", timeout=5000
        )
        return self.page.locator(DATA_ROW_SELECTOR).count()

    def get_column_count(self) -> int:
        """Get the number of data columns in the table (excludes the row resizer cell)."""
        data_row = self.page.locator(DATA_ROW_SELECTOR).first
        data_row.wait_for(state="visible", timeout=5000)
        return data_row.locator(".pluto-table__cell").count()

    def select_cell(self, row: int, col: int) -> None:
        """Single-click select a cell."""
        self._get_cell(row, col).click()

    def shift_select_cell(self, row: int, col: int) -> None:
        """Shift-click to extend the selection through this cell."""
        self._get_cell(row, col).click(modifiers=["Shift"])

    def ctrl_select_cell(self, row: int, col: int) -> None:
        """Ctrl/Cmd-click to toggle this cell in the selection."""
        self._get_cell(row, col).click(modifiers=["ControlOrMeta"])

    def select_row_via_header(self, row: int) -> None:
        """Click the row indicator (left header) to select every cell in the row."""
        self.page.locator(f'td[id="resizer-y-{row}"]').click()

    def select_col_via_header(self, col: int) -> None:
        """Click the column indicator (top header) to select every cell in the column."""
        self.page.locator(f'td[id="resizer-x-{col}"]').click()

    def add_row_above(self, row: int, col: int = 0) -> None:
        """Insert a row above the row containing the cell at (row, col)."""
        self.ctx_menu.action(self._get_cell(row, col), "Add row above")

    def add_row_below(self, row: int, col: int = 0) -> None:
        """Insert a row below the row containing the cell at (row, col)."""
        self.ctx_menu.action(self._get_cell(row, col), "Add row below")

    def add_col_left(self, col: int, row: int = 0) -> None:
        """Insert a column to the left of the cell at (row, col)."""
        self.ctx_menu.action(self._get_cell(row, col), "Add column left")

    def add_col_right(self, col: int, row: int = 0) -> None:
        """Insert a column to the right of the cell at (row, col)."""
        self.ctx_menu.action(self._get_cell(row, col), "Add column right")

    def set_cell_text(self, row: int, col: int, text: str) -> None:
        """Replace the text content of a text-variant cell."""
        cell = self._get_cell(row, col)
        cell.dblclick()
        self.page.keyboard.type(text)
        self.page.keyboard.press("Enter")

    def copy(self) -> None:
        """Cmd/Ctrl + C — copy current selection to clipboard."""
        self.layout.press_key("ControlOrMeta+c")

    def paste(self) -> None:
        """Cmd/Ctrl + V — paste clipboard at current selection anchor."""
        self.layout.press_key("ControlOrMeta+v")

    def delete_selected(self) -> None:
        """Delete key — clear selected cells, removing fully-selected rows/cols."""
        self.layout.press_key("Delete")

    def undo(self) -> None:
        """Cmd/Ctrl + Z — pop the last entry off the undo stack."""
        self.layout.press_key("ControlOrMeta+z")

    def redo(self) -> None:
        """Cmd/Ctrl + Shift + Z — re-apply the most recently undone entry."""
        self.layout.press_key("ControlOrMeta+Shift+z")

    def toggle_editing(self) -> None:
        """Toggle the editable state via the cell context menu."""
        cell = self._get_cell(0, 0)
        self.ctx_menu.open_on(cell)
        label = (
            "Disable editing"
            if self.ctx_menu.has_option("Disable editing")
            else "Enable editing"
        )
        self.ctx_menu.click_option(label)

    def drag_col_resizer(self, col: int, dx: float) -> None:
        """Drag the right edge of a column's resize handle by `dx` pixels.

        Positive `dx` widens the column; negative narrows it.
        """
        handle = self.page.locator(f'td[id="resizer-x-{col}"] button')
        box = handle.bounding_box()
        if box is None:
            raise ValueError(f"Could not locate resize handle for column {col}")
        cx = box["x"] + box["width"] / 2
        cy = box["y"] + box["height"] / 2
        self.page.mouse.move(cx, cy)
        self.page.mouse.down()
        self.page.mouse.move(cx + dx, cy, steps=10)
        self.page.mouse.up()

    def drag_row_resizer(self, row: int, dy: float) -> None:
        """Drag the bottom edge of a row's resize handle by `dy` pixels."""
        handle = self.page.locator(f'td[id="resizer-y-{row}"] button')
        box = handle.bounding_box()
        if box is None:
            raise ValueError(f"Could not locate resize handle for row {row}")
        cx = box["x"] + box["width"] / 2
        cy = box["y"] + box["height"] / 2
        self.page.mouse.move(cx, cy)
        self.page.mouse.down()
        self.page.mouse.move(cx, cy + dy, steps=10)
        self.page.mouse.up()

    def get_column_width(self, col: int) -> float:
        """Read the rendered pixel width of a column."""
        indicator = self.page.locator(f'td[id="resizer-x-{col}"]')
        return float(indicator.evaluate("el => el.offsetWidth"))

    def get_row_height(self, row: int) -> float:
        """Read the rendered pixel height of a row's indicator."""
        indicator = self.page.locator(f'td[id="resizer-y-{row}"]')
        return float(indicator.evaluate("el => el.offsetHeight"))

    SIZE_LABELS = ("XL", "L", "M", "S", "XS")

    def set_toolbar_variant(self, variant: str) -> None:
        """Open the toolbar's Variant dropdown and pick the named option."""
        self.layout.show_visualization_toolbar()
        self.layout.click_btn("Variant")
        self.layout.select_from_dropdown(variant)

    def get_toolbar_variant(self) -> str:
        """Read the toolbar's Variant dropdown value. Empty string when the
        selected cells disagree on variant."""
        self.layout.show_visualization_toolbar()
        return self.layout.get_dropdown_value("Variant")

    def set_toolbar_size(self, label: str) -> None:
        """Click one of the toolbar's Size buttons (XL/L/M/S/XS)."""
        self.layout.show_visualization_toolbar()
        self.page.get_by_text(label, exact=True).first.click()

    def get_toolbar_size(self) -> str | None:
        """Return the selected Size button label, or None when no size is
        active (multi-cell anchor has no level or selected cells disagree)."""
        self.layout.show_visualization_toolbar()
        try:
            return self.layout.get_selected_button(list(self.SIZE_LABELS))
        except RuntimeError:
            return None

    def get_toolbar_cell_count(self) -> int:
        """Read the multi-cell selection count from the toolbar breadcrumb.

        Returns the N from the "N cells" segment when 2+ cells are
        selected, 1 when a single cell is selected (the segment shows a
        position like "A1"), or 0 when no cell is selected.
        """
        self.layout.show_visualization_toolbar()
        segments = self.page.locator(".pluto-breadcrumb__segment")
        if segments.count() < 2:
            return 0
        text = segments.nth(1).inner_text().strip()
        if text.endswith("cells"):
            return int(text.split()[0])
        return 1

    def get_color_swatch_count(self) -> int:
        """Count the swatches in the toolbar's "Selection colors" group.

        Each distinct color across the selection contributes one swatch,
        so this is the number of color groups the multi-cell form is
        rendering. Returns 0 when the group is absent (single cell or
        no cells with a color prop).
        """
        self.layout.show_visualization_toolbar()
        label = self.page.get_by_text("Selection colors", exact=True).first
        if label.count() == 0:
            return 0
        group = label.locator("..")
        return group.locator(".pluto-color-swatch").count()
