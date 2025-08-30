#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from testcases.playwright.playwright import Playwright

class Tabs_Open_Close(Playwright):
    """
    Open all pages in the "New Component" window and close them
    """

    def run(self) -> None:

        pages=[
            "Line Plot",
            "Schematic",
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

        self.open_page("Control", ["Case_Sensitive_Name"])
        for p in pages:
            self.open_page(p)

        self.close_page("Case_Sensitive_Name")
        for p in pages:
            self.close_page(p)
        