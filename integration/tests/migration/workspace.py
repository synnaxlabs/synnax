#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test: import a workspace on old version, verify after migration."""

import json
from abc import abstractmethod
from typing import Any

from console.case import ConsoleCase
from framework.utils import get_fixture_path

WORKSPACE_NAME = "mig_workspace"
FIXTURE_DIR = "ImportSpace"
EXPECTED_PAGES = ["Metrics Plot", "Metrics Schematic", "Metrics Log", "Metrics Table"]


def _load_workspace_fixture() -> dict[str, Any]:
    """Build an import-ready workspace data dict from the ImportSpace fixtures.

    Returns a dict with 'layout' and 'components' keys matching the format
    expected by WorkspaceClient.import_workspace.
    """
    layout_path = get_fixture_path(f"{FIXTURE_DIR}/LAYOUT.json")
    with open(layout_path) as f:
        layout = json.load(f)

    components: dict[str, Any] = {}
    for key, entry in layout.get("layouts", {}).items():
        name = entry.get("name")
        entry_type = entry.get("type")
        if entry_type in ("main", None):
            continue
        try:
            page_path = get_fixture_path(f"{FIXTURE_DIR}/{name}.json")
        except FileNotFoundError:
            continue
        with open(page_path) as f:
            component = json.load(f)
        component["type"] = entry_type
        components[key] = component

    return {"layout": layout, "components": components}


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
    """Import a full workspace from fixtures."""

    def test_workspace(self) -> None:
        self.log("Testing: Import workspace")
        data = _load_workspace_fixture()
        self.console.workspace.import_workspace(WORKSPACE_NAME, data)

    def test_page(self) -> None:
        for name in EXPECTED_PAGES:
            assert self.console.workspace.page_exists(
                name
            ), f"Page '{name}' not found after import"


class WorkspacesVerify(WorkspaceMigration):
    """Verify workspace and all pages survive migration."""

    def test_workspace(self) -> None:
        self.log("Testing: Workspace exists with all pages")
        self.console.workspace.select(WORKSPACE_NAME)
        for name in EXPECTED_PAGES:
            assert self.console.workspace.page_exists(
                name
            ), f"Page '{name}' not found after migration"

    def test_page(self) -> None:
        self.log("Testing: All pages render after migration")
        for name in EXPECTED_PAGES:
            self.console.workspace.open_page(name)
            tab = self.console.layout.get_tab(name)
            assert tab.is_visible(), f"Page '{name}' tab is not visible after opening"
            self.console.layout.close_tab(name)
