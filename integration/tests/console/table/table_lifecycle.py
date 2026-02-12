#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from console.case import ConsoleCase
from console.table import Table
from framework.utils import assert_link_format, get_fixture_path, get_random_name


class TableLifecycle(ConsoleCase):
    """Test table lifecycle operations."""

    suffix: str
    idx_name: str
    data_name: str
    main_table_name: str | None
    main_table_link: str
    ctx_table_name: str | None

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self.main_table_name = None
        self.ctx_table_name = None

        ctx_table = self.console.workspace.create_table(
            f"Context Menu Test {self.suffix}"
        )
        self.ctx_table_name = ctx_table.page_name
        ctx_table.close()

    def teardown(self) -> None:
        names_to_cleanup = [
            self.ctx_table_name,
            self.main_table_name,
        ]
        for name in names_to_cleanup:
            if name and self.console.workspace.page_exists(name):
                self.console.workspace.delete_page(name)
        super().teardown()

    def setup_channels(self) -> None:
        """Create test channels for value cell telemetry."""
        self.idx_name = f"table_test_idx_{self.suffix}"
        self.data_name = f"table_test_data_{self.suffix}"

        index_ch = self.client.channels.create(
            name=self.idx_name,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name=self.data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        """Run all table lifecycle tests."""
        self.setup_channels()

        table = self.console.workspace.create_table(f"Table Test {self.suffix}")

        # Visualization
        self.test_add_rows_and_columns(table)
        self.test_delete_rows_and_columns(table)
        self.test_add_redlines(table)

        # Save link + name, then close
        self.main_table_link = table.copy_link()
        self.main_table_name = table.page_name
        table.close()
        assert not table.is_open, "Table should be closed"

        # Resources Toolbar
        self.test_open_table_from_resources()
        self.test_drag_table_onto_mosaic()

        # Search and Command Palette
        self.test_open_table_from_search()
        self.test_import_table_from_file()

        # Context menu operations
        self.test_ctx_copy_link()
        self.test_ctx_export_json()
        self.test_ctx_delete()

        # Cleanup channels
        self.client.channels.delete([self.idx_name, self.data_name])

    def test_add_rows_and_columns(self, table: Table) -> None:
        """Test adding rows and columns to a table."""
        self.log("Testing add rows and columns")

        initial_rows = table.get_row_count()
        initial_cols = table.get_column_count()

        table.add_row()
        assert (
            table.get_row_count() == initial_rows + 1
        ), f"Expected {initial_rows + 1} rows after add, got {table.get_row_count()}"

        table.add_column()
        assert (
            table.get_column_count() == initial_cols + 1
        ), f"Expected {initial_cols + 1} cols after add, got {table.get_column_count()}"

    def test_delete_rows_and_columns(self, table: Table) -> None:
        """Test deleting rows and columns from a table."""
        self.log("Testing delete rows and columns")

        rows_before = table.get_row_count()
        cols_before = table.get_column_count()

        # Add a row and column so we can safely delete them
        table.add_row()
        table.add_column()
        assert table.get_row_count() == rows_before + 1
        assert table.get_column_count() == cols_before + 1

        # Delete the last row
        table.delete_row(table.get_row_count() - 1)
        assert (
            table.get_row_count() == rows_before
        ), f"Expected {rows_before} rows after delete, got {table.get_row_count()}"

        # Delete the last column
        table.delete_column(table.get_column_count() - 1)
        assert (
            table.get_column_count() == cols_before
        ), f"Expected {cols_before} cols after delete, got {table.get_column_count()}"

    def test_add_redlines(self, table: Table) -> None:
        """Test setting a channel on a value cell and adding redlines."""
        self.log("Testing add redlines to value cell")

        table.set_cell_channel(self.data_name, row=1, col=0)
        assert table.has_channel(
            self.data_name, row=1, col=0
        ), f"Cell [1,0] should have channel '{self.data_name}'"

        table.set_redline(row=1, col=0, lower=10.0, upper=90.0)
        lower, upper = table.get_redline(row=1, col=0)
        assert lower == "10", f"Lower bound should be '10', got '{lower}'"
        assert upper == "90", f"Upper bound should be '90', got '{upper}'"

    def test_open_table_from_resources(self) -> None:
        """Test opening a table by double-clicking in the workspace resources toolbar."""
        assert self.main_table_name is not None
        self.log("Testing open table from resources toolbar")

        table = self.console.workspace.open_table(self.main_table_name)
        assert (
            table.is_pane_visible
        ), f"Table '{self.main_table_name}' pane not visible after opening"

        opened_link = table.copy_link()
        assert (
            opened_link == self.main_table_link
        ), f"Link mismatch: expected {self.main_table_link}, got {opened_link}"
        table.close()

    def test_drag_table_onto_mosaic(self) -> None:
        """Test dragging a table from the resources toolbar onto the mosaic."""
        assert self.main_table_name is not None
        self.log("Testing drag table onto mosaic")

        table = self.console.workspace.drag_table_to_mosaic(self.main_table_name)
        assert (
            table.is_pane_visible
        ), f"Table '{self.main_table_name}' pane not visible after drag"

        opened_link = table.copy_link()
        assert (
            opened_link == self.main_table_link
        ), f"Link mismatch: expected {self.main_table_link}, got {opened_link}"
        table.close()

    def test_open_table_from_search(self) -> None:
        """Test opening a table by searching its name in the command palette."""
        assert self.main_table_name is not None
        self.log("Testing open table from search palette")

        table = self.console.workspace.open_from_search(Table, self.main_table_name)
        assert (
            table.is_pane_visible
        ), f"Table '{self.main_table_name}' pane not visible after search"

        opened_link = table.copy_link()
        assert (
            opened_link == self.main_table_link
        ), f"Link mismatch: expected {self.main_table_link}, got {opened_link}"
        table.close()

    def test_import_table_from_file(self) -> None:
        """Test importing a table from a JSON file."""
        self.log("Testing import table from file")
        json_path = get_fixture_path("ImportSpace/Metrics Table.json")
        imported_name = f"Imported Table {self.suffix}"
        self.console.workspace.import_page(json_path, imported_name)

        assert self.console.workspace.page_exists(
            imported_name
        ), f"Imported table '{imported_name}' should appear in workspace"

        table = Table.from_open_page(self.console.layout, self.client, imported_name)
        assert (
            table.get_row_count() == 2
        ), f"Expected 2 rows, got {table.get_row_count()}"
        assert (
            table.get_column_count() == 2
        ), f"Expected 2 columns, got {table.get_column_count()}"
        table.close()
        self.console.workspace.delete_page(imported_name)

    def test_ctx_copy_link(self) -> None:
        """Test copying a link to a table via context menu."""
        assert self.ctx_table_name is not None
        self.log("Testing copy link via context menu")
        link = self.console.workspace.copy_page_link(self.ctx_table_name)
        assert_link_format(link, "table")

    def test_ctx_export_json(self) -> None:
        """Test exporting a table as JSON via context menu."""
        assert self.ctx_table_name is not None
        self.log("Testing export table via context menu")
        exported = self.console.workspace.export_page(self.ctx_table_name)
        assert "key" in exported, "Exported JSON should contain 'key'"
        assert len(exported["key"]) == 36, "Table key should be a UUID"

    def test_ctx_delete(self) -> None:
        """Test deleting a table via context menu."""
        assert self.ctx_table_name is not None
        self.log("Testing delete table via context menu")
        self.console.workspace.delete_page(self.ctx_table_name)
        self.ctx_table_name = None
