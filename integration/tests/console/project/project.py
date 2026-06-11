#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import shutil

from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.case import ConsoleCase
from console.log import Log
from console.plot import Plot
from console.schematic.schematic import Schematic
from console.table import Table
from framework.utils import get_fixture_path
from x import get_synnax_version

EXPECTED_PAGES = ["Metrics Plot", "Metrics Schematic", "Metrics Log", "Metrics Table"]


class Project(ConsoleCase):
    """Test project operations."""

    _cleanup_projects: list[str]

    def setup(self) -> None:
        super().setup()
        self._cleanup_projects = []

    def teardown(self) -> None:
        for name in self._cleanup_projects:
            try:
                self.console.project.delete(name)
            except PlaywrightTimeoutError:
                pass
        super().teardown()

    def run(self) -> None:
        self.test_version_visible_in_navbar()

        # Project Navigation
        self.console.project.create("ProjectA")
        self._cleanup_projects.append("ProjectA")
        self.console.project.create("ProjectB")
        self._cleanup_projects.append("ProjectB")
        self.test_switch_projects_in_resources()
        self.test_rename_project()
        self.test_clear_project_from_selector()
        self.test_delete_project()

        # Import Pages
        self.console.project.create("ImportSpace")
        self._cleanup_projects.append("ImportSpace")
        self.test_import_line_plot()
        self.test_import_schematic()
        self.test_import_log()
        self.test_import_table()

        # Export/Import
        self.test_export_project()
        self.test_import_project()

    def test_version_visible_in_navbar(self) -> None:
        """Test that the correct version is displayed in the navbar."""
        self.log("Testing version badge visible in navbar")
        expected = f"v{get_synnax_version()}"
        displayed = self.console.layout.get_version()
        self.log(f"Version badge displays: {displayed}, expected prefix: {expected}")
        assert displayed.startswith(expected), (
            f"Version badge '{displayed}' does not start with expected '{expected}'"
        )

    def test_switch_projects_in_resources(self) -> None:
        """Test switching between projects by double-clicking in resources toolbar."""
        self.log("Testing switch projects in resources view")

        self.console.project.select("ProjectA")
        assert (
            self.page.get_by_role("button").filter(has_text="ProjectA").is_visible()
        ), "ProjectA should be active after selection"

        self.console.project.select("ProjectB")
        assert (
            self.page.get_by_role("button").filter(has_text="ProjectB").is_visible()
        ), "ProjectB should be active after selection"

    def test_rename_project(self) -> None:
        """Test renaming a project via context menu and verify synchronization."""
        self.log("Testing rename project with synchronization")
        self.console.project.select("ProjectA")
        self.console.project.rename(old_name="ProjectA", new_name="RenamedProject")

        assert self.console.project.exists("RenamedProject"), (
            "Project should be renamed in Resources Toolbar"
        )

        project_selector = self.page.get_by_role("button").filter(
            has_text="RenamedProject"
        )
        project_selector.wait_for(state="visible", timeout=5000)
        assert project_selector.is_visible(), (
            "Project Selector should show renamed project"
        )

        self.console.project.rename(old_name="RenamedProject", new_name="ProjectA")
        self.console.layout.close_left_toolbar()

    def test_import_line_plot(self) -> None:
        """Test importing a line plot from a JSON file."""
        self.log("Testing import line plot")
        json_path = get_fixture_path("ImportSpace/Metrics Plot.json")
        self.console.project.import_page(json_path, "Metrics Plot")

        assert self.console.project.page_exists("Metrics Plot"), (
            "Imported line plot should appear in project resources"
        )

        plot = Plot.from_open_page(self.console.layout, self.client, "Metrics Plot")
        labels = plot.get_line_labels()
        expected = [
            "sy_node_1_metrics_cpu_percentage",
            "sy_node_1_metrics_mem_percentage",
        ]
        assert labels == expected, f"Expected line labels {expected}, got {labels}"

        self.console.layout.close_tab("Metrics Plot")

    def test_import_schematic(self) -> None:
        """Test importing a schematic from a JSON file."""
        self.log("Testing import schematic")
        json_path = get_fixture_path("ImportSpace/Metrics Schematic.json")
        self.console.project.import_page(json_path, "Metrics Schematic")

        assert self.console.project.page_exists("Metrics Schematic"), (
            "Imported schematic should appear in project resources"
        )

        schematic = Schematic.from_open_page(
            self.console.layout, self.client, "Metrics Schematic"
        )
        assert schematic.get_symbol_count() == 2, (
            "Imported schematic should have 2 gauge symbols"
        )

        self.console.layout.close_tab("Metrics Schematic")

    def test_import_log(self) -> None:
        """Test importing a log from a JSON file."""
        self.log("Testing import log")
        json_path = get_fixture_path("ImportSpace/Metrics Log.json")
        self.console.project.import_page(json_path, "Metrics Log")

        assert self.console.project.page_exists("Metrics Log"), (
            "Imported log should appear in project resources"
        )

        log = Log.from_open_page(self.console.layout, self.client, "Metrics Log")
        assert log.has_channel("sy_node_1_metrics_cpu_percentage"), (
            "Imported log should have sy_node_1_metrics_cpu_percentage channel"
        )

        self.console.layout.close_tab("Metrics Log")

    def test_import_table(self) -> None:
        """Test importing a table from a JSON file."""
        self.log("Testing import table")
        json_path = get_fixture_path("ImportSpace/Metrics Table.json")
        self.console.project.import_page(json_path, "Metrics Table")

        assert self.console.project.page_exists("Metrics Table"), (
            "Imported table should appear in project resources"
        )

        table = Table.from_open_page(self.console.layout, self.client, "Metrics Table")
        assert table.get_row_count() == 2, (
            f"Expected 2 rows, got {table.get_row_count()}"
        )
        assert table.get_column_count() == 2, (
            f"Expected 2 columns, got {table.get_column_count()}"
        )
        assert table.has_text("CPU %", 0, 0), "Cell [0,0] should have text 'CPU %'"
        assert table.has_text("Mem %", 0, 1), "Cell [0,1] should have text 'Mem %'"
        assert table.has_channel("sy_node_1_metrics_cpu_percentage", 1, 0), (
            "Cell [1,0] should have sy_node_1_metrics_cpu_percentage"
        )
        assert table.has_channel("sy_node_1_metrics_mem_percentage", 1, 1), (
            "Cell [1,1] should have sy_node_1_metrics_mem_percentage"
        )

        self.console.layout.close_tab("Metrics Table")

    def test_export_project(self) -> None:
        """Test exporting a project through the real Export action."""
        self.log("Testing export project")

        # Re-open all imported pages so they exist in Redux state
        # (import tests close tabs, which removes layouts from Redux)
        for p in EXPECTED_PAGES:
            self.console.project.open_page(p)

        self._export_dir = self.console.project.export_project("ImportSpace")

        for p in EXPECTED_PAGES:
            self.console.layout.close_tab(p)

        with open(os.path.join(self._export_dir, "LAYOUT.json"), "r") as f:
            layout_data = json.load(f)
        layouts = {l["name"]: l["type"] for l in layout_data["layouts"].values()}
        expected = {
            "Metrics Plot": "lineplot",
            "Metrics Schematic": "schematic",
            "Metrics Log": "log",
            "Metrics Table": "table",
        }
        for page_name, page_type in expected.items():
            assert page_name in layouts, f"Export should contain layout '{page_name}'"
            assert layouts[page_name] == page_type, (
                f"Layout '{page_name}' should be type {page_type}, got "
                f"{layouts[page_name]}"
            )
            assert os.path.exists(
                os.path.join(self._export_dir, f"{page_name}.json")
            ), f"Export should contain {page_name}.json"

    def test_import_project(self) -> None:
        """Test importing a project through the real "Import a project" command."""
        self.log("Testing import project via command palette")

        # Rename the export directory so the imported project has a
        # distinct name from the original ImportSpace project.
        imported_dir = os.path.join(os.path.dirname(self._export_dir), "ImportedSpace")
        if os.path.isdir(imported_dir):
            shutil.rmtree(imported_dir)
        os.rename(self._export_dir, imported_dir)

        self.console.project.import_project_from_directory(imported_dir)

        for tab_name in EXPECTED_PAGES:
            tab = self.console.layout.get_tab(tab_name)
            assert tab.is_visible(), f"Imported project should have tab '{tab_name}'"

        self.console.project.delete("ImportedSpace")

    def test_clear_project_from_selector(self) -> None:
        """Test clearing projects from the selector (switching to no project)."""
        self.log("Testing clear project from selector")

        self.console.project.select("ProjectA")

        project_selector = self.page.get_by_role("button").filter(has_text="ProjectA")
        project_selector.click(timeout=5000)
        self.page.get_by_role("button", name="Clear").click(timeout=5000)

        self.page.get_by_role("button", name="No project").wait_for(
            state="visible", timeout=5000
        )
        assert self.page.get_by_role("button", name="No project").is_visible(), (
            "No project should be active after clearing"
        )

        self.console.layout.close_left_toolbar()

    def test_delete_project(self) -> None:
        """Test deleting a project via context menu."""
        self.log("Testing delete project")

        self.console.project.delete("ProjectB")
        self._cleanup_projects.remove("ProjectB")
        self.console.project.delete("ProjectA")
        self._cleanup_projects.remove("ProjectA")
