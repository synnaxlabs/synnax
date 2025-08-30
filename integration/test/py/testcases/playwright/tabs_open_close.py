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

        sequence_name = "Case_Sensitive_Name"
        self.open_page("Line Plot")
        self.open_page("Schematic")
        self.open_page("log Log")
        self.open_page("Table")
        self.open_page("NI Analog Read Task")
        self.open_page("NI Analog Write Task") 
        self.open_page("NI Digital Read Task")
        self.open_page("NI Digital Write Task")
        self.open_page("LabJack Read Task")
        self.open_page("LabJack Write Task")
        self.open_page("OPC UA Read Task")
        self.open_page("OPC UA Write Task")
        self.open_page("Control", [sequence_name])
        self.close_page("Get Started")
        self.close_page("Line Plot")
        self.close_page("Schematic")
        self.close_page("Log")
        self.close_page("Table")
        self.close_page("NI Analog Read Task")
        self.close_page("NI Analog Write Task")
        self.close_page("NI Digital Read Task")
        self.close_page("NI Digital Write Task")
        self.close_page("LabJack Read Task")
        self.close_page("LabJack Write Task")
        self.close_page("OPC UA Read Task")
        self.close_page("OPC UA Write Task")
        self.close_page(sequence_name)