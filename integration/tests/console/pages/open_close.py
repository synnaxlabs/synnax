#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from console.console import PageType


class OpenClose(ConsoleCase):
    """
    Test creating and closing pages
    """

    def run(self) -> None:
        """
        Test Opening and closing pages
        """
        console = self.console

        pages: list[PageType] = [
            "Schematic",
            "Line Plot",
            "Log",
            "Table",
            "NI Analog Read Task",
            "NI Analog Write Task",
            "NI Digital Read Task",
            "NI Digital Write Task",
            "LabJack Read Task",
            "LabJack Write Task",
            "OPC UA Read Task",
            "OPC UA Write Task",
        ]

        pages_renamed: list[tuple[PageType, str]] = [
            ("Schematic", "S_Name"),
            ("Line Plot", "L_Name"),
            ("Log", "Log_Name"),
            ("Table", "Table_Name"),
            ("NI Analog Read Task", "AI"),
            ("NI Analog Write Task", "AO"),
            ("NI Digital Read Task", "DI"),
            ("NI Digital Write Task", "DO"),
            ("LabJack Read Task", "LabJack R"),
            ("LabJack Write Task", "LabJack O"),
            ("OPC UA Read Task", "OPC Read"),
            ("OPC UA Write Task", "OPC Write"),
        ]

        self.log("(1/4) Create pages by cmd palette (Default names)")
        for p in pages:
            console._create_page_by_command_palette(p)
        for p in pages:
            console.close_page(p)

        self.log("(2/4) Create pages by cmd palette (Custom names)")
        for page_type, page_name in pages_renamed:
            console._create_page_by_command_palette(page_type, page_name)
        for page_type, page_name in pages_renamed:
            console.close_page(page_name)

        self.log("(3/4) Create pages by (+) button (Default names)")
        for p in pages:
            console._create_page_by_new_page_button(p)
        for p in pages:
            console.close_page(p)

        self.log("(4/4) Create pages by (+) button (Custom names)")
        for page_type, page_name in pages_renamed:
            console._create_page_by_new_page_button(page_type, page_name)
        for page_type, page_name in pages_renamed:
            console.close_page(page_name)

        # Close "Get Started" if it's still open (may have been closed by workspace selection)
        get_started_tab = console._get_tab_locator("Get Started")
        if get_started_tab.count() > 0:
            console.close_page("Get Started")

        # Should see "New Component" if all pages closed successfully
        pass_condition = self.page.get_by_text("New Component").count() > 0
        assert (
            pass_condition
        ), "Some pages were not closed - 'New Component' screen not visible"
