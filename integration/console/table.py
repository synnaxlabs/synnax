#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from .console import Console
from .page import ConsolePage


class Table(ConsolePage):
    """Table page management interface"""

    page_type: str = "Table"
    pluto_label: str = ".pluto-table"

    def __init__(
        self,
        console: Console,
        page_name: str,
        *,
        _skip_create: bool = False,
    ) -> None:
        """
        Initialize a Table page.

        Args:
            console: Console instance
            page_name: Name for the page
            _skip_create: Internal flag to skip page creation (used by factory methods)
        """
        super().__init__(console, page_name, _skip_create=_skip_create)

    def set_cell_channel(self, channel_name: str, row: int = 0, col: int = 0) -> None:
        """Set a cell to display a channel's telemetry value.

        Args:
            channel_name: Name of the channel to display
            row: Row index (0-based)
            col: Column index (0-based)
        """
        self._click_cell(row, col)
        self.layout.show_visualization_toolbar()
        self.console.click_btn("Variant")
        self.console.select_from_dropdown("Value")
        self.page.get_by_text("Telemetry").click()
        self.console.click_btn("Input Channel")
        self.console.select_from_dropdown(channel_name)

    def get_cell_channel(self, row: int = 0, col: int = 0) -> str:
        """Get the channel name displayed in a cell.

        Args:
            row: Row index (0-based)
            col: Column index (0-based)

        Returns:
            The channel name or empty string if not set
        """
        self.layout.get_tab(self.page_name).click()
        self._click_cell(row, col)
        self.layout.show_visualization_toolbar()
        self.page.get_by_text("Telemetry").click()
        channel_btn = (
            self.page.locator("text=Input Channel")
            .locator("..")
            .locator("button")
            .first
        )
        channel_text = channel_btn.inner_text().strip()
        return channel_text

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

    def _click_cell(self, row: int, col: int) -> None:
        """Click on a specific cell in the table."""
        cells = self.page.locator(".pluto-table__cell")
        cell_index = row * self._get_column_count() + col
        cells.nth(cell_index).click()

    def _get_column_count(self) -> int:
        """Get the number of columns in the table."""
        first_row = self.page.locator(".pluto-table__row").first
        return first_row.locator(".pluto-table__cell").count()
