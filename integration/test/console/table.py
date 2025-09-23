#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Dict, List, Optional, Union

from playwright.sync_api import Page

from .console_page import ConsolePage

if TYPE_CHECKING:
    from .console import Console


class Table(ConsolePage):
    """Table page management interface"""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "Table"
        self.pluto_label = ".pluto-table"

    def add_column(self) -> None:
        pass

    def remove_column(self) -> None:
        pass

    def get_row_count(self) -> int:
        return 0

    def edit_cell(self, row: int, column: str, value: Any) -> None:
        pass
