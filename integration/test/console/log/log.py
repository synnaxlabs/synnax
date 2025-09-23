#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, List, Optional

from playwright.sync_api import Page

from ..console_page import ConsolePage

if TYPE_CHECKING:
    from ..console import Console


class Log(ConsolePage):
    """Log page management interface"""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "Log"
        self.pluto_label = ".pluto-log"

    def clear(self) -> None:
        """Clear all log entries."""
        pass

    def get_entries(self, level: Optional[str] = None) -> List[str]:
        return []
