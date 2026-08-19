#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Drop OS files and directories onto the mosaic.

The mosaic's file-drop handler imports every dropped JSON file through the
server, imports a dropped directory as a project, and reports a file that
fails on its own. The drop is synthesized through CDP so the dropped items
resolve real file-system entries, exactly like an OS drag.
"""

import json
import os
import shutil
from contextlib import ExitStack

from console.case import ConsoleCase
from console.log import Log
from console.page import ConsolePage
from console.schematic.schematic import Schematic
from console.table import Table
from framework.run_dir import resolve_results_path
from framework.utils import get_core_fixture_path, named_envelope_copy
from x import random_name


class FileDrop(ConsoleCase):
    """Drop goldens and a project bundle onto the mosaic and verify."""

    _cleanup_projects: list[str]

    def setup(self) -> None:
        super().setup()
        self._cleanup_projects = []

    def teardown(self) -> None:
        # Deleted server-side: a UI delete of the active project drops the
        # Console to the project selector, stranding the base page cleanup.
        for name in self._cleanup_projects:
            try:
                matches = self.client.projects.retrieve(search_term=name)
                self.client.projects.delete(
                    [p.key for p in matches if p.name == name]
                )
            except Exception as e:
                self.log(f"Failed to delete project {name}: {e}")
        super().teardown()

    def run(self) -> None:
        self.suffix = random_name()
        self.test_drop_single_file()
        self.test_drop_multiple_files()
        self.test_drop_non_json_file()
        self.test_drop_project_directory()

    def test_drop_single_file(self) -> None:
        """Drop one golden envelope and verify its page opens."""
        self.log("Dropping a single golden file")
        name = f"drop_schematic_{self.suffix}"
        path = get_core_fixture_path("schematic/versions/testdata/import_v7.json")
        with named_envelope_copy(path, name) as tmp_path:
            self.console.layout.drop_files([tmp_path])
            self.console.layout.wait_for_tab(name)
        self._cleanup_pages.append(name)
        page = Schematic.from_open_page(self.console.layout, self.client, name)
        assert page.is_pane_visible, "dropped schematic pane not visible"
        page.close()

    def test_drop_multiple_files(self) -> None:
        """Drop two goldens in one drag and verify both open as one batch."""
        self.log("Dropping multiple golden files")
        goldens: list[tuple[str, str, type[ConsolePage]]] = [
            ("log/versions/testdata/import_v2.json", f"drop_log_{self.suffix}", Log),
            (
                "table/versions/testdata/import_v1.json",
                f"drop_table_{self.suffix}",
                Table,
            ),
        ]
        with ExitStack() as stack:
            paths = [
                stack.enter_context(
                    named_envelope_copy(get_core_fixture_path(relative_path), name)
                )
                for relative_path, name, _ in goldens
            ]
            self.console.layout.drop_files(paths)
            for _, name, _ in goldens:
                self.console.layout.wait_for_tab(name)
        for _, name, page_class in goldens:
            self._cleanup_pages.append(name)
            # Both tabs open as one batch with the last one selected, so each
            # page is brought to the front before its pane is checked.
            self.console.layout.get_tab(name).click()
            page = page_class.from_open_page(self.console.layout, self.client, name)
            assert page.is_pane_visible, f"dropped {name} pane not visible"
            page.close()

    def test_drop_non_json_file(self) -> None:
        """Drop a non-JSON file and assert the failure surfaces."""
        self.log("Dropping a non-JSON file")
        file_name = f"drop_notes_{self.suffix}.txt"
        path = resolve_results_path(file_name)
        with open(path, "w", encoding="utf-8") as f:
            f.write("not an envelope")
        self.console.layout.drop_files([path])
        if not self.console.notifications.wait_for(f"Failed to import {file_name}"):
            raise AssertionError(
                "Dropping a non-JSON file did not surface a failure notification"
            )
        self.console.notifications.close_all()

    def test_drop_project_directory(self) -> None:
        """Drop an exported project bundle directory and verify it imports."""
        self.log("Dropping a project bundle directory")
        # The drop leaves the imported project active, where the base page
        # cleanup cannot see the earlier drops, so they are flushed first.
        self.console.pages.delete_many(self._cleanup_pages)
        self._cleanup_pages.clear()
        source = f"DropSource_{self.suffix}"
        self.console.project.create(source)
        self._cleanup_projects.append(source)
        page_name = f"drop_proj_page_{self.suffix}"
        page = self.console.pages.create(Schematic, page_name)
        export_dir = self.console.project.export(source)

        imported = f"DropImported_{self.suffix}"
        imported_dir = os.path.join(os.path.dirname(export_dir), imported)
        if os.path.isdir(imported_dir):
            shutil.rmtree(imported_dir)
        os.rename(export_dir, imported_dir)
        manifest_path = os.path.join(imported_dir, "manifest.json")
        with open(manifest_path, "r", encoding="utf-8") as f:
            manifest = json.load(f)
        manifest["name"] = imported
        with open(manifest_path, "w", encoding="utf-8") as f:
            json.dump(manifest, f)

        page.close()
        self.console.layout.drop_files([imported_dir])
        self.console.project.wait_for_active(imported)
        self._cleanup_projects.append(imported)
        tab = self.console.layout.get_tab(page_name)
        assert tab.is_visible(), "imported project should reopen its page tab"
