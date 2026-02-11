#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

from console.case import ConsoleCase
from console.log import Log
from console.plot import Plot
from console.schematic.schematic import Schematic
from console.table import Table
from framework.utils import get_fixture_path, get_results_path, get_synnax_version

EXPECTED_PAGES = ["Metrics Plot", "Metrics Schematic", "Metrics Log", "Metrics Table"]


class Workspace(ConsoleCase):
    """Test workspace operations."""

    def run(self) -> None:
        self.test_version_visible_in_navbar()

        # Workspace Navigation
        self.console.workspace.create("WorkspaceA")
        self.console.workspace.create("WorkspaceB")
        self.test_switch_workspaces_in_resources()
        self.test_rename_workspace()
        self.test_clear_workspace_from_selector()
        self.test_delete_workspace()

        # Import Pages
        self.console.workspace.create("ImportSpace")
        self.test_import_line_plot()
        self.test_import_schematic()
        self.test_import_log()
        self.test_import_table()

        # Export/Import
        self.test_export_workspace()
        self.test_import_workspace()

    def test_version_visible_in_navbar(self) -> None:
        """Test that the correct version is displayed in the navbar."""
        self.log("Testing version badge visible in navbar")
        expected = f"v{get_synnax_version()}"
        displayed = self.console.layout.get_version()
        self.log(f"Version badge displays: {displayed}, expected prefix: {expected}")
        assert displayed.startswith(
            expected
        ), f"Version badge '{displayed}' does not start with expected '{expected}'"

    def test_switch_workspaces_in_resources(self) -> None:
        """Test switching between workspaces by double-clicking in resources toolbar."""
        self.log("Testing switch workspaces in resources view")

        self.console.workspace.select("WorkspaceA")
        assert (
            self.page.get_by_role("button").filter(has_text="WorkspaceA").is_visible()
        ), "WorkspaceA should be active after selection"

        self.console.workspace.select("WorkspaceB")
        assert (
            self.page.get_by_role("button").filter(has_text="WorkspaceB").is_visible()
        ), "WorkspaceB should be active after selection"

    def test_rename_workspace(self) -> None:
        """Test renaming a workspace via context menu and verify synchronization."""
        self.log("Testing rename workspace with synchronization")
        self.console.workspace.select("WorkspaceA")
        self.console.workspace.rename(
            old_name="WorkspaceA", new_name="RenamedWorkspace"
        )

        assert self.console.workspace.exists(
            "RenamedWorkspace"
        ), "Workspace should be renamed in Resources Toolbar"

        workspace_selector = self.page.get_by_role("button").filter(
            has_text="RenamedWorkspace"
        )
        workspace_selector.wait_for(state="visible", timeout=5000)
        assert (
            workspace_selector.is_visible()
        ), "Workspace Selector should show renamed workspace"

        self.console.workspace.rename(
            old_name="RenamedWorkspace", new_name="WorkspaceA"
        )
        self.console.layout.close_left_toolbar()

    def test_import_line_plot(self) -> None:
        """Test importing a line plot from a JSON file."""
        self.log("Testing import line plot")
        json_path = get_fixture_path("ImportSpace/Metrics Plot.json")
        self.console.workspace.import_page(json_path, "Metrics Plot")

        assert self.console.workspace.page_exists(
            "Metrics Plot"
        ), "Imported line plot should appear in workspace resources"

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
        self.console.workspace.import_page(json_path, "Metrics Schematic")

        assert self.console.workspace.page_exists(
            "Metrics Schematic"
        ), "Imported schematic should appear in workspace resources"

        schematic = Schematic.from_open_page(
            self.console.layout, self.client, "Metrics Schematic"
        )
        assert (
            schematic.get_symbol_count() == 2
        ), "Imported schematic should have 2 gauge symbols"
        props = schematic.get_properties()
        assert (
            props["control_authority"] == 134
        ), f"Expected control_authority 134, got {props['control_authority']}"
        assert (
            props["show_control_legend"] is False
        ), "Expected show_control_legend to be False"

        self.console.layout.close_tab("Metrics Schematic")

    def test_import_log(self) -> None:
        """Test importing a log from a JSON file."""
        self.log("Testing import log")
        json_path = get_fixture_path("ImportSpace/Metrics Log.json")
        self.console.workspace.import_page(json_path, "Metrics Log")

        assert self.console.workspace.page_exists(
            "Metrics Log"
        ), "Imported log should appear in workspace resources"

        log = Log.from_open_page(self.console.layout, self.client, "Metrics Log")
        assert log.has_channel(
            "sy_node_1_metrics_cpu_percentage"
        ), "Imported log should have sy_node_1_metrics_cpu_percentage channel"

        self.console.layout.close_tab("Metrics Log")

    def test_import_table(self) -> None:
        """Test importing a table from a JSON file."""
        self.log("Testing import table")
        json_path = get_fixture_path("ImportSpace/Metrics Table.json")
        self.console.workspace.import_page(json_path, "Metrics Table")

        assert self.console.workspace.page_exists(
            "Metrics Table"
        ), "Imported table should appear in workspace resources"

        table = Table.from_open_page(self.console.layout, self.client, "Metrics Table")
        assert (
            table.get_row_count() == 2
        ), f"Expected 2 rows, got {table.get_row_count()}"
        assert (
            table.get_column_count() == 2
        ), f"Expected 2 columns, got {table.get_column_count()}"
        assert table.has_text("CPU %", 0, 0), "Cell [0,0] should have text 'CPU %'"
        assert table.has_text("Mem %", 0, 1), "Cell [0,1] should have text 'Mem %'"
        assert table.has_channel(
            "sy_node_1_metrics_cpu_percentage", 1, 0
        ), "Cell [1,0] should have sy_node_1_metrics_cpu_percentage"
        assert table.has_channel(
            "sy_node_1_metrics_mem_percentage", 1, 1
        ), "Cell [1,1] should have sy_node_1_metrics_mem_percentage"

        self.console.layout.close_tab("Metrics Table")

    def test_export_workspace(self) -> None:
        """Test exporting a workspace via JS injection to results directory."""
        self.log("Testing export workspace")

        # Re-open all imported pages so they exist in Redux state
        # (import tests close tabs, which removes layouts from Redux)

        for p in EXPECTED_PAGES:
            self.console.workspace.open_page(p)

        result = self.console.workspace.export_workspace("ImportSpace")

        for p in EXPECTED_PAGES:
            self.console.layout.close_tab(p)

        layouts = {l["name"]: l["type"] for l in result["layout"]["layouts"].values()}
        component_types = {c["type"] for c in result["components"].values()}

        expected = {
            "Metrics Plot": "lineplot",
            "Metrics Schematic": "schematic",
            "Metrics Log": "log",
            "Metrics Table": "table",
        }
        for page_name, page_type in expected.items():
            assert page_name in layouts, f"Export should contain layout '{page_name}'"
            assert (
                page_type in component_types
            ), f"Export should contain {page_type} component"

    def test_import_workspace(self) -> None:
        """Test importing a workspace via command palette."""
        self.log("Testing import workspace via command palette")

        export_path = get_results_path("ImportSpace_export.json")
        with open(export_path, "r") as f:
            data = json.load(f)

        self.console.workspace.import_workspace("ImportSpace", data)
        for tab_name in EXPECTED_PAGES:
            tab = self.console.layout.get_tab(tab_name)
            assert tab.is_visible(), f"Imported workspace should have tab '{tab_name}'"

        self.console.workspace.delete("ImportSpace")

    def test_clear_workspace_from_selector(self) -> None:
        """Test clearing workspaces from the selector (switching to no workspace)."""
        self.log("Testing clear workspace from selector")

        self.console.workspace.select("WorkspaceA")

        workspace_selector = self.page.get_by_role("button").filter(
            has_text="WorkspaceA"
        )
        workspace_selector.click(timeout=5000)
        self.page.get_by_role("button", name="Clear").click(timeout=5000)

        self.page.get_by_role("button", name="No workspace").wait_for(
            state="visible", timeout=5000
        )
        assert self.page.get_by_role(
            "button", name="No workspace"
        ).is_visible(), "No workspace should be active after clearing"

        self.console.layout.close_left_toolbar()

    def test_delete_workspace(self) -> None:
        """Test deleting a workspace via context menu."""
        self.log("Testing delete workspace")

        self.console.workspace.delete("WorkspaceB")
        self.console.workspace.delete("WorkspaceA")
