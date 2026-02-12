#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from playwright.sync_api import Locator

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
        self.layout.show_visualization_toolbar()
        self.layout.click_btn("Variant")
        self.layout.select_from_dropdown("Value")
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
        add_row_btn = self.page.locator(".console-table__add-row").first
        add_row_btn.wait_for(state="visible", timeout=5000)
        add_row_btn.click()

    def add_column(self) -> None:
        """Add a new column to the table by clicking the add-column button."""
        add_col_btn = self.page.locator(".console-table__add-col").first
        add_col_btn.wait_for(state="visible", timeout=5000)
        add_col_btn.click()

    def delete_row(self, row: int, col: int = 0) -> None:
        """Delete a row via context menu on a cell.

        Args:
            row: Row index (0-based)
            col: Column index (0-based) of the cell to right-click
        """
        cell = self._get_cell(row, col)
        self.ctx_menu.action(cell, "Delete row")

    def delete_column(self, col: int, row: int = 0) -> None:
        """Delete a column via context menu on a cell.

        Args:
            col: Column index (0-based) of the column to delete
            row: Row index (0-based) of the cell to right-click
        """
        cell = self._get_cell(row, col)
        self.ctx_menu.action(cell, "Delete column")

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
