#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test: create a workspace with a single visualization on old version."""

from console.case import ConsoleCase
from framework.utils import get_fixture_path

WORKSPACE_NAME = "mig_workspace"
LINE_PLOT_NAME = "mig_line_plot"
FIXTURE_PATH = "ImportSpace/Metrics Plot.json"


class WorkspacesSetup(ConsoleCase):
    """Create a workspace and import a line plot fixture."""

    def run(self) -> None:
        self.test_create_workspace()
        self.test_import_page()

    def test_create_workspace(self) -> None:
        self.log("Testing: Create workspace")
        self.console.workspace.create(WORKSPACE_NAME)

    def test_import_page(self) -> None:
        self.log("Testing: Import line plot page")
        fixture_path = get_fixture_path(FIXTURE_PATH)
        self.console.workspace.import_page(fixture_path, LINE_PLOT_NAME)
        tab = self.console.layout.get_tab(LINE_PLOT_NAME)
        assert tab.is_visible(), "Imported line plot tab is not visible"


class WorkspacesVerify(ConsoleCase):
    """Verify workspace and line plot exist after migration."""

    def run(self) -> None:
        self.test_workspace_exists()
        self.test_page_renders()

    def test_workspace_exists(self) -> None:
        self.log("Testing: Workspace exists")
        self.console.workspace.select(WORKSPACE_NAME)
        assert self.console.workspace.page_exists(
            LINE_PLOT_NAME
        ), f"Page '{LINE_PLOT_NAME}' not found after migration"

    def test_page_renders(self) -> None:
        self.log("Testing: Page renders after migration")
        self.console.workspace.open_page(LINE_PLOT_NAME)
        tab = self.console.layout.get_tab(LINE_PLOT_NAME)
        assert tab.is_visible(), "Line plot tab is not visible after opening"
