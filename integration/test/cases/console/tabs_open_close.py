#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from test.console.console_case import ConsoleCase


class Tabs_Open_Close(ConsoleCase):
    """
    Open all pages in the "New Component" window and close them
    """

    def run(self) -> None:

        console = self.console
        print(console)

        self._log_message("(1/2) Creating pages by command palette")
        page_names = [
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

        names = []
        for page_name in page_names:
            console.create_page(page_name[0], page_name[1])
            names.append(page_name[1])
        for name in names:
            console.close_page(name)

        self._log_message("(2/2) Creating pages by manual add")
        pages = [
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

        console.open_page("Control", ["Control Sequence"])
        for p in pages:
            console.open_page(p)

        console.close_page("Control Sequence")
        for p in pages:
            console.close_page(p)

        console.close_page("Get Started")

        # Should see "New Component" if all pages closed successfully
        if self.page.get_by_text("New Component").count() > 0:
            self._log_message("All pages closed - 'New Component' screen visible")
        else:
            self._log_message(
                "FAILED: Pages still be open - 'New Component' screen not visible"
            )
            self.fail()

    # KEEP THIS HERE
    def open_page(self, page_name: str, inputs_items: list[str] = []) -> None:

        """
        This differs from create_page in that it uses the manual
        New Page (+) button instead of the command palette.
        """
        self.page.locator(".pluto-icon--add").first.click()  # (+)
        self.page.get_by_role("button", name=page_name).first.click()
        # Apply inputs
        for i in inputs_items:
            self.page.get_by_role("textbox", name="Name").fill(i)
            self.page.get_by_role("textbox", name="Name").press("ControlOrMeta+Enter")
