#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from console.workspace import PageType


class OpenClose(ConsoleCase):
    """
    Test creating and closing pages
    """

    def run(self) -> None:
        """
        Test Opening and closing pages
        """
        console = self.console

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

        self.log("(1/2) Create pages by cmd palette")
        for page_type, page_name in pages_renamed:
            console.workspace.create_page_by_command_palette(page_type, page_name)
        for page_type, page_name in pages_renamed:
            console.workspace.close_page(page_name)

        self.log("(2/2) Create pages by (+) button")
        for page_type, page_name in pages_renamed:
            console.workspace.create_page_by_new_page_button(page_type, page_name)
        for page_type, page_name in pages_renamed:
            console.workspace.close_page(page_name)

        # Close "Get Started" if it's still open (may have been closed by workspace selection)
        get_started_tab = console.layout.get_tab("Get Started")
        if get_started_tab.count() > 0:
            console.workspace.close_page("Get Started")

        # Should see "New Component" if all pages closed successfully
        pass_condition = self.page.get_by_text("New Component").count() > 0
        assert (
            pass_condition
        ), "Some pages were not closed - 'New Component' screen not visible"
