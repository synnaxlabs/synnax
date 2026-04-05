#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test: import a workspace on old version, verify after migration."""

from abc import abstractmethod

from console.case import ConsoleCase
from framework.utils import get_fixture_path

WORKSPACE_NAME = "mig_workspace"
FIXTURE_DIR = "ImportSpace"

PAGES = [
    ("Metrics Plot", f"{FIXTURE_DIR}/Metrics Plot.json"),
    ("Metrics Schematic", f"{FIXTURE_DIR}/Metrics Schematic.json"),
    ("Metrics Log", f"{FIXTURE_DIR}/Metrics Log.json"),
    ("Metrics Table", f"{FIXTURE_DIR}/Metrics Table.json"),
]


class WorkspaceMigration(ConsoleCase):
    """Base class defining the migration test contract for workspaces.

    Subclasses must implement each test method — setup creates the state,
    verify checks it after migration.
    """

    def run(self) -> None:
        self.test_workspace()
        self.test_page()

    @abstractmethod
    def test_workspace(self) -> None: ...

    @abstractmethod
    def test_page(self) -> None: ...


class WorkspacesSetup(WorkspaceMigration):
    """Create a workspace and import page fixtures."""

    def test_workspace(self) -> None:
        self.log("Testing: Create workspace")
        self.console.workspace.create(WORKSPACE_NAME)

    def test_page(self) -> None:
        self.log("Testing: Import pages")
        for name, fixture in PAGES:
            self.log(f"Importing {name}")
            json_path = get_fixture_path(fixture)
            self.console.workspace.import_page(json_path, name)
            assert self.console.workspace.page_exists(
                name
            ), f"Page '{name}' not found after import"
            self.console.layout.close_tab(name)


class WorkspacesVerify(WorkspaceMigration):
    """Verify workspace and all pages survive migration."""

    def test_workspace(self) -> None:
        self.log("Testing: Workspace exists with all pages")
        self.console.workspace.select(WORKSPACE_NAME)
        for name, _ in PAGES:
            assert self.console.workspace.page_exists(
                name
            ), f"Page '{name}' not found after migration"

    def test_page(self) -> None:
        self.log("Testing: All pages render after migration")
        for name, _ in PAGES:
            self.console.workspace.open_page(name)
            tab = self.console.layout.get_tab(name)
            assert tab.is_visible(), f"Page '{name}' tab is not visible after opening"
            self.console.layout.close_tab(name)
