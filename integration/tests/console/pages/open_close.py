#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.case import ConsoleCase
from console.workspace import PageType
from framework.utils import get_random_name


class OpenClose(ConsoleCase):
    """
    Test creating and closing pages
    """

    _cleanup_pages: list[str]

    def setup(self) -> None:
        super().setup()
        self._cleanup_pages = []

    def teardown(self) -> None:
        for name in self._cleanup_pages:
            try:
                self.console.workspace.delete_page(name)
            except PlaywrightTimeoutError:
                pass
        super().teardown()

    def run(self) -> None:
        """
        Test Opening and closing pages
        """
        console = self.console
        suffix = get_random_name()

        pages_renamed: list[tuple[PageType, str]] = [
            ("Schematic", f"Sch_{suffix}"),
            ("Line Plot", f"LinePlt_{suffix}"),
            ("Log", f"LogPg_{suffix}"),
            ("Table", f"TablePg_{suffix}"),
            ("NI Analog Read Task", f"NIAR_{suffix}"),
            ("NI Analog Write Task", f"NIAW_{suffix}"),
            ("NI Digital Read Task", f"NIDR_{suffix}"),
            ("NI Digital Write Task", f"NIDW_{suffix}"),
            ("LabJack Read Task", f"LJRead_{suffix}"),
            ("LabJack Write Task", f"LJWrite_{suffix}"),
            ("OPC UA Read Task", f"OPCRead_{suffix}"),
            ("OPC UA Write Task", f"OPCWrite_{suffix}"),
        ]

        self.log("(1/2) Create pages by cmd palette")
        for page_type, page_name in pages_renamed:
            console.workspace.create_page_by_command_palette(page_type, page_name)
            self._cleanup_pages.append(page_name)
        for page_type, page_name in pages_renamed:
            console.workspace.close_page(page_name)

        self.log("(2/2) Create pages by (+) button")
        for page_type, page_name in pages_renamed:
            console.workspace.create_page_by_new_page_button(page_type, page_name)
            self._cleanup_pages.append(page_name)
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
