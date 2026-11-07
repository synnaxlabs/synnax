#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from console.page import PageType
from framework.hardware_case import HardwareCase
from x import random_name


class OpenClose(HardwareCase, ConsoleCase):
    """
    Test creating and closing pages
    """

    _page_names: list[str]
    _task_names: list[str]

    def setup(self) -> None:
        super().setup()
        self._page_names = []
        self._task_names = []
        self._pages_deleted = False

    def run(self) -> None:
        """
        Test Opening and closing pages
        """
        console = self.console
        suffix = random_name()

        PROJECT_PAGES: list[tuple[PageType, str]] = [
            ("schematic", f"Sch_{suffix}"),
            ("lineplot", f"LinePlt_{suffix}"),
            ("log", f"LogPg_{suffix}"),
            ("table", f"TablePg_{suffix}"),
        ]

        TASK_PAGES: list[tuple[PageType, str]] = [
            ("ni_analog_read", f"NIAR_{suffix}"),
            ("ni_analog_write", f"NIAW_{suffix}"),
            ("ni_digital_read", f"NIDR_{suffix}"),
            ("ni_digital_write", f"NIDW_{suffix}"),
            ("labjack_read", f"LJRead_{suffix}"),
            ("labjack_write", f"LJWrite_{suffix}"),
            ("opc_read", f"OPCRead_{suffix}"),
            ("opc_write", f"OPCWrite_{suffix}"),
        ]

        self.log("(1/2) Create pages by cmd palette")
        all_pages = PROJECT_PAGES + TASK_PAGES
        self._task_names = [name for _, name in TASK_PAGES]

        for page_type, page_name in all_pages:
            console.pages.create_by_command_palette(page_type, page_name)
            if page_type in ("schematic", "lineplot", "log", "table"):
                self._page_names.append(page_name)
        for _, page_name in all_pages:
            console.layout.close_tab(page_name)

        self.log("(2/2) Create pages by (+) button")
        for page_type, page_name in all_pages:
            console.pages.create_by_new_page_button(page_type, page_name)
            if page_type in ("schematic", "lineplot", "log", "table"):
                self._page_names.append(page_name)
        for _, page_name in all_pages:
            console.layout.close_tab(page_name)

        remaining = console.layout.tab_names()
        assert not remaining, f"Some pages were not closed: {remaining}"

        console.pages.delete_many(self._page_names)
        self._pages_deleted = True

    def teardown(self) -> None:
        if not self._pages_deleted:
            self._cleanup_pages.extend(self._page_names)
        # Each task page saves its task to the Core on creation.
        with self._try_to("find created tasks"):
            if self._task_names:
                self.track_test_tasks(
                    self.client.tasks.retrieve(names=self._task_names)
                )
        super().teardown()
