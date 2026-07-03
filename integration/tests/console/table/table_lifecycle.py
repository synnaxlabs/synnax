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
from framework.utils import assert_link_format, get_fixture_path
from x import random_name


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
        self.suffix = random_name()
        self.main_table_name = None
        self.ctx_table_name = None

        ctx_table = self.console.project.create_table(
            f"Context Menu Test {self.suffix}"
        )
        self.ctx_table_name = ctx_table.page_name
        ctx_table.close()

    def teardown(self) -> None:
        for name in [self.ctx_table_name, self.main_table_name]:
            if name:
                self._cleanup_pages.append(name)
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

        table = self.console.project.create_table(f"Table Test {self.suffix}")

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

        # Interaction: cell editing, selection, clipboard, keyboard shortcuts
        interaction_name = f"Interaction Test {self.suffix}"
        interaction = self.console.project.create_table(interaction_name)
        self._cleanup_pages.append(interaction.page_name)
        try:
            self.test_cell_text_editing(interaction)
            self.test_context_menu_insert(interaction)
            self.test_delete_clears_cell_variant(interaction)
            self.test_delete_removes_full_row(interaction)
            self.test_delete_removes_full_col(interaction)
            self.test_copy_paste_in_bounds(interaction)
            self.test_copy_paste_value_variant(interaction)
            self.test_paste_grows_table(interaction)
            self.test_undo_redo_add_row(interaction)
            self.test_undo_redo_text_edit(interaction)
            self.test_resize_col_via_drag(interaction)
            self.test_resize_row_via_drag(interaction)
            self.test_edit_toggle_gates_editing(interaction)
            self.test_multi_cell_variant_swap(interaction)
            self.test_multi_cell_color_grouping(interaction)
            self.test_multi_cell_level_bulk_apply(interaction)
            self.test_multi_cell_level_empty_when_disagreed(interaction)
        finally:
            interaction.close()

        # Cleanup channels
        self.client.channels.delete([self.idx_name, self.data_name])

    def test_add_rows_and_columns(self, table: Table) -> None:
        """Test adding rows and columns to a table."""
        self.log("Testing add rows and columns")

        initial_rows = table.get_row_count()
        initial_cols = table.get_column_count()

        table.add_row()
        assert table.get_row_count() == initial_rows + 1, (
            f"Expected {initial_rows + 1} rows after add, got {table.get_row_count()}"
        )

        table.add_column()
        assert table.get_column_count() == initial_cols + 1, (
            f"Expected {initial_cols + 1} cols after add, got {table.get_column_count()}"
        )

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
        assert table.get_row_count() == rows_before, (
            f"Expected {rows_before} rows after delete, got {table.get_row_count()}"
        )

        # Delete the last column
        table.delete_column(table.get_column_count() - 1)
        assert table.get_column_count() == cols_before, (
            f"Expected {cols_before} cols after delete, got {table.get_column_count()}"
        )

    def test_add_redlines(self, table: Table) -> None:
        """Test setting a channel on a value cell and adding redlines."""
        self.log("Testing add redlines to value cell")

        table.set_cell_channel(self.data_name, row=1, col=0)
        assert table.has_channel(self.data_name, row=1, col=0), (
            f"Cell [1,0] should have channel '{self.data_name}'"
        )

        table.set_redline(row=1, col=0, lower=10.0, upper=90.0)
        lower, upper = table.get_redline(row=1, col=0)
        assert lower == "10", f"Lower bound should be '10', got '{lower}'"
        assert upper == "90", f"Upper bound should be '90', got '{upper}'"

    def test_open_table_from_resources(self) -> None:
        """Test opening a table by double-clicking in the project resources toolbar."""
        assert self.main_table_name is not None
        self.log("Testing open table from resources toolbar")

        table = self.console.project.open_table(self.main_table_name)
        assert table.is_pane_visible, (
            f"Table '{self.main_table_name}' pane not visible after opening"
        )

        opened_link = table.copy_link()
        assert opened_link == self.main_table_link, (
            f"Link mismatch: expected {self.main_table_link}, got {opened_link}"
        )
        table.close()

    def test_drag_table_onto_mosaic(self) -> None:
        """Test dragging a table from the resources toolbar onto the mosaic."""
        assert self.main_table_name is not None
        self.log("Testing drag table onto mosaic")

        table = self.console.project.drag_table_to_mosaic(self.main_table_name)
        assert table.is_pane_visible, (
            f"Table '{self.main_table_name}' pane not visible after drag"
        )

        opened_link = table.copy_link()
        assert opened_link == self.main_table_link, (
            f"Link mismatch: expected {self.main_table_link}, got {opened_link}"
        )
        table.close()

    def test_open_table_from_search(self) -> None:
        """Test opening a table by searching its name in the command palette."""
        assert self.main_table_name is not None
        self.log("Testing open table from search palette")

        table = self.console.project.open_from_search(Table, self.main_table_name)
        assert table.is_pane_visible, (
            f"Table '{self.main_table_name}' pane not visible after search"
        )

        opened_link = table.copy_link()
        assert opened_link == self.main_table_link, (
            f"Link mismatch: expected {self.main_table_link}, got {opened_link}"
        )
        table.close()

    def test_import_table_from_file(self) -> None:
        """Test importing a table from a JSON file."""
        self.log("Testing import table from file")
        json_path = get_fixture_path("ImportSpace/Metrics Table.json")
        imported_name = self.console.project.import_page(json_path)

        assert self.console.project.page_exists(imported_name), (
            f"Imported table '{imported_name}' should appear in project"
        )

        table = Table.from_open_page(self.console.layout, self.client, imported_name)
        assert table.get_row_count() == 2, (
            f"Expected 2 rows, got {table.get_row_count()}"
        )
        assert table.get_column_count() == 2, (
            f"Expected 2 columns, got {table.get_column_count()}"
        )
        table.close()
        self.console.project.delete_page(imported_name)

    def test_ctx_copy_link(self) -> None:
        """Test copying a link to a table via context menu."""
        assert self.ctx_table_name is not None
        self.log("Testing copy link via context menu")
        link = self.console.project.copy_page_link(self.ctx_table_name)
        assert_link_format(link, "table")

    def test_ctx_export_json(self) -> None:
        """Test exporting a table as JSON via context menu."""
        assert self.ctx_table_name is not None
        self.log("Testing export table via context menu")
        exported = self.console.project.export_page(self.ctx_table_name)
        assert "key" in exported, "Exported JSON should contain 'key'"
        assert len(exported["key"]) == 36, "Table key should be a UUID"

    def test_ctx_delete(self) -> None:
        """Test deleting a table via context menu."""
        assert self.ctx_table_name is not None
        self.log("Testing delete table via context menu")
        self.console.project.delete_page(self.ctx_table_name)
        self.ctx_table_name = None

    def test_cell_text_editing(self, table: Table) -> None:
        """Test that typing in a text cell persists the new value."""
        self.log("Testing cell text editing")
        table.set_cell_text(0, 0, "alpha")
        assert table.has_text("alpha", row=0, col=0), (
            f"Cell (0,0) should read 'alpha', got '{table.get_cell_text(0, 0)}'"
        )
        table.set_cell_text(1, 1, "bravo")
        assert table.has_text("bravo", row=1, col=1), (
            f"Cell (1,1) should read 'bravo', got '{table.get_cell_text(1, 1)}'"
        )
        # First edit must remain unaffected by the second.
        assert table.has_text("alpha", row=0, col=0), (
            "Editing one cell must not change another's content"
        )

    def test_context_menu_insert(self, table: Table) -> None:
        """Test inserting rows/columns via context menu relative to a cell."""
        self.log("Testing context menu insert above/below/left/right")
        rows_before = table.get_row_count()
        cols_before = table.get_column_count()

        table.add_row_above(0, 0)
        assert table.get_row_count() == rows_before + 1, (
            f"Add row above: expected {rows_before + 1}, got {table.get_row_count()}"
        )
        table.add_row_below(table.get_row_count() - 1, 0)
        assert table.get_row_count() == rows_before + 2, (
            f"Add row below: expected {rows_before + 2}, got {table.get_row_count()}"
        )
        table.add_col_left(0, 0)
        assert table.get_column_count() == cols_before + 1, (
            f"Add col left: expected {cols_before + 1}, got {table.get_column_count()}"
        )
        table.add_col_right(table.get_column_count() - 1, 0)
        assert table.get_column_count() == cols_before + 2, (
            f"Add col right: expected {cols_before + 2}, got {table.get_column_count()}"
        )

        # Restore to the starting shape by removing what we added.
        for _ in range(2):
            table.delete_row(table.get_row_count() - 1)
            table.delete_column(table.get_column_count() - 1)
        assert table.get_row_count() == rows_before
        assert table.get_column_count() == cols_before

    def test_delete_clears_cell_variant(self, table: Table) -> None:
        """Test that Delete on a Value cell resets it to the default Text variant."""
        self.log("Testing Delete key clears cell variant")
        table.set_cell_channel(self.data_name, row=0, col=0)
        assert table.has_channel(self.data_name, row=0, col=0), (
            "Setup: cell should hold the channel after set_cell_channel"
        )
        table.select_cell(0, 0)
        table.delete_selected()
        # After Delete the cell should be a default text cell with empty content.
        assert table.has_text("", row=0, col=0), (
            "Cell should be empty text after Delete"
        )

    def test_delete_removes_full_row(self, table: Table) -> None:
        """Test that selecting an entire row via its header and pressing Delete removes the row."""
        self.log("Testing Delete key removes a fully selected row")
        # Add a buffer row so we don't run the table down to zero rows.
        table.add_row()
        rows_before = table.get_row_count()
        cols_before = table.get_column_count()
        target_row = rows_before - 1
        table.select_row_via_header(target_row)
        table.delete_selected()
        assert table.get_row_count() == rows_before - 1, (
            f"Expected {rows_before - 1} rows after row delete, got {table.get_row_count()}"
        )
        assert table.get_column_count() == cols_before, (
            "Deleting a row must not change column count"
        )

    def test_delete_removes_full_col(self, table: Table) -> None:
        """Test that selecting an entire column via its header and pressing Delete removes the column."""
        self.log("Testing Delete key removes a fully selected column")
        table.add_column()
        rows_before = table.get_row_count()
        cols_before = table.get_column_count()
        target_col = cols_before - 1
        table.select_col_via_header(target_col)
        table.delete_selected()
        assert table.get_column_count() == cols_before - 1, (
            f"Expected {cols_before - 1} cols after col delete, got {table.get_column_count()}"
        )
        assert table.get_row_count() == rows_before, (
            "Deleting a column must not change row count"
        )

    def test_copy_paste_in_bounds(self, table: Table) -> None:
        """Test that copying a 1x2 region and pasting at a new anchor overwrites cells."""
        self.log("Testing copy/paste within current table bounds")
        # Grow the table to a known 2x2 starting shape and seed source cells.
        while table.get_row_count() < 2:
            table.add_row()
        while table.get_column_count() < 2:
            table.add_column()
        table.set_cell_text(0, 0, "src-a")
        table.set_cell_text(0, 1, "src-b")
        table.set_cell_text(1, 0, "dst-a")
        table.set_cell_text(1, 1, "dst-b")

        # Copy the top row (0,0)+(0,1).
        table.select_cell(0, 0)
        table.shift_select_cell(0, 1)
        table.copy()

        # Paste anchored at the bottom-left cell.
        table.select_cell(1, 0)
        table.paste()

        assert table.has_text("src-a", row=1, col=0), (
            f"Pasted (1,0) should be 'src-a', got '{table.get_cell_text(1, 0)}'"
        )
        assert table.has_text("src-b", row=1, col=1), (
            f"Pasted (1,1) should be 'src-b', got '{table.get_cell_text(1, 1)}'"
        )
        assert table.has_text("src-a", row=0, col=0), (
            "Source row must remain untouched after paste"
        )

    def test_copy_paste_value_variant(self, table: Table) -> None:
        """Test that copy/paste carries a value cell's variant + channel onto
        a previously-text destination cell. Exercises the Aether
        register/unregister path on the destination because the Spec.Cell
        component swaps from Text to Value mid-render."""
        self.log("Testing copy/paste of a value-variant cell")
        # The grid is left in some prior shape by earlier interaction tests;
        # normalize to a clean 2x2 of text cells before seeding the source.
        while table.get_row_count() < 2:
            table.add_row()
        while table.get_column_count() < 2:
            table.add_column()
        table.set_cell_text(0, 0, "kept")
        table.set_cell_text(1, 0, "overwritten")

        table.set_cell_channel(self.data_name, row=0, col=0)
        assert table.has_channel(self.data_name, row=0, col=0), (
            "Setup: source cell should hold the channel after set_cell_channel"
        )

        table.select_cell(0, 0)
        table.copy()
        table.select_cell(1, 0)
        table.paste()

        assert table.has_channel(self.data_name, row=1, col=0), (
            f"Destination (1,0) should hold channel '{self.data_name}' after "
            f"paste, got '{table.get_cell_channel(1, 0)}'"
        )
        assert table.has_channel(self.data_name, row=0, col=0), (
            "Source (0,0) must remain a value cell after copy"
        )

        # Reset (0,0) and (1,0) back to text so the rest of the interaction
        # suite can dblclick them into edit mode. A value cell's dblclick is
        # a no-op, so any keyboard input that follows leaks to the document
        # and fires console shortcuts (notably opening nav drawers).
        for r in (0, 1):
            table.select_cell(r, 0)
            table.delete_selected()
            assert table.has_text("", row=r, col=0), (
                f"({r},0) should be cleared to empty text after Delete"
            )

    def test_paste_grows_table(self, table: Table) -> None:
        """Test that pasting beyond the current bounds grows the table."""
        self.log("Testing paste grows table on overflow")
        rows_before = table.get_row_count()
        cols_before = table.get_column_count()

        # Copy two cells from the top row, paste anchored at the bottom-right.
        table.set_cell_text(0, 0, "grow-a")
        table.set_cell_text(0, 1, "grow-b")
        table.select_cell(0, 0)
        table.shift_select_cell(0, 1)
        table.copy()
        table.select_cell(rows_before - 1, cols_before - 1)
        table.paste()

        # Pasting 1x2 with anchor at the last column adds exactly one column;
        # pasting at the last row doesn't add rows because the source is one row tall.
        assert table.get_column_count() == cols_before + 1, (
            f"Expected {cols_before + 1} cols after overflow paste, got {table.get_column_count()}"
        )
        assert table.get_row_count() == rows_before, (
            "Paste should not grow rows when the source is one row tall and anchored on the last row"
        )

    def test_undo_redo_add_row(self, table: Table) -> None:
        """Test that Cmd+Z undoes an add-row and Cmd+Shift+Z re-applies it."""
        self.log("Testing undo/redo for add row")
        # Each shortcut needs the table focused; re-click before every key combo
        # since cross-render state can drop focus on a re-rendered table.
        table.select_cell(0, 0)
        rows_before = table.get_row_count()
        table.add_row()
        assert table.get_row_count() == rows_before + 1
        table.select_cell(0, 0)
        table.undo()
        assert table.get_row_count() == rows_before, (
            f"Undo should restore row count to {rows_before}, got {table.get_row_count()}"
        )
        table.select_cell(0, 0)
        table.redo()
        assert table.get_row_count() == rows_before + 1, (
            f"Redo should re-add the row, got {table.get_row_count()}"
        )
        # Restore for the next test.
        table.select_cell(0, 0)
        table.undo()
        assert table.get_row_count() == rows_before

    def test_resize_col_via_drag(self, table: Table) -> None:
        """Test that dragging the column resize handle changes the column width."""
        self.log("Testing column resize via drag")
        initial = table.get_column_width(0)
        table.drag_col_resizer(0, 80)
        # Allow for some slack since the drag end coordinate maps through
        # the action handler's minimum-cell-dimension clamp and rounding.
        after = table.get_column_width(0)
        assert after >= initial + 40, (
            f"Column 0 should be wider after drag right: was {initial}, now {after}"
        )
        table.drag_col_resizer(0, -120)
        narrow = table.get_column_width(0)
        assert narrow < after, (
            f"Column 0 should be narrower after drag left: was {after}, now {narrow}"
        )

    def test_resize_row_via_drag(self, table: Table) -> None:
        """Test that dragging the row resize handle changes the row height."""
        self.log("Testing row resize via drag")
        initial = table.get_row_height(0)
        table.drag_row_resizer(0, 60)
        after = table.get_row_height(0)
        assert after >= initial + 30, (
            f"Row 0 should be taller after drag down: was {initial}, now {after}"
        )

    def test_edit_toggle_gates_editing(self, table: Table) -> None:
        """Test that toggling editing off hides edit affordances and re-enables them on toggle."""
        self.log("Testing edit toggle gates editing affordances")
        add_row_btn = self.page.locator(".pluto-table-frame__add-row").first
        add_col_btn = self.page.locator(".pluto-table-frame__add-col").first
        add_row_btn.wait_for(state="visible", timeout=5000)
        add_col_btn.wait_for(state="visible", timeout=5000)

        table.toggle_editing()
        add_row_btn.wait_for(state="hidden", timeout=5000)
        add_col_btn.wait_for(state="hidden", timeout=5000)

        # While editing is off, attempting the Delete shortcut on a selected
        # row must be a no-op since the gate sits inside the trigger callback.
        rows_before = table.get_row_count()
        table.select_row_via_header(0)
        table.delete_selected()
        assert table.get_row_count() == rows_before, (
            "Delete shortcut must be a no-op when editing is disabled"
        )

        table.toggle_editing()
        add_row_btn.wait_for(state="visible", timeout=5000)
        add_col_btn.wait_for(state="visible", timeout=5000)

    def test_undo_redo_text_edit(self, table: Table) -> None:
        """Test that Cmd+Z reverts a text edit and Cmd+Shift+Z re-applies it.

        Successive set_cell dispatches on the same cell coalesce into one
        undo entry (that's the "typing in a cell is one undo" feature). To
        verify a single-edit undo, an edit on a different cell sits between
        the two edits on cell (0,0) and breaks the coalesce chain.
        """
        self.log("Testing undo/redo for text edit")
        table.set_cell_text(0, 0, "before-undo")
        table.set_cell_text(0, 1, "scratch-break-coalesce")
        table.set_cell_text(0, 0, "after-edit")
        assert table.has_text("after-edit", row=0, col=0)
        table.select_cell(0, 0)
        table.undo()
        assert table.has_text("before-undo", row=0, col=0), (
            f"Undo should restore 'before-undo', got '{table.get_cell_text(0, 0)}'"
        )
        table.select_cell(0, 0)
        table.redo()
        assert table.has_text("after-edit", row=0, col=0), (
            f"Redo should restore 'after-edit', got '{table.get_cell_text(0, 0)}'"
        )

    def test_multi_cell_variant_swap(self, table: Table) -> None:
        """Test that the toolbar's Variant dropdown swaps every selected cell's
        variant when 2+ cells are selected."""
        self.log("Testing multi-cell variant swap via toolbar")
        while table.get_row_count() < 1:
            table.add_row()
        while table.get_column_count() < 2:
            table.add_column()
        table.set_cell_text(0, 0, "swap-a")
        table.set_cell_text(0, 1, "swap-b")

        table.select_cell(0, 0)
        table.shift_select_cell(0, 1)
        assert table.get_toolbar_cell_count() == 2, (
            "Toolbar breadcrumb should report 2-cell selection before variant swap"
        )

        table.set_toolbar_variant("Value")

        table.select_cell(0, 0)
        assert table.get_toolbar_variant() == "Value", (
            "Cell (0,0) variant should be 'Value' after multi-cell swap"
        )
        table.select_cell(0, 1)
        assert table.get_toolbar_variant() == "Value", (
            "Cell (0,1) variant should be 'Value' after multi-cell swap"
        )

        # Restore so the next test starts from text cells.
        for c in (0, 1):
            table.select_cell(0, c)
            table.delete_selected()

    def test_multi_cell_color_grouping(self, table: Table) -> None:
        """Test that selecting 2+ text cells with the same backgroundColor
        renders a single Color.Swatch in the toolbar's Selection colors
        group (not one swatch per cell)."""
        self.log("Testing multi-cell color grouping")
        while table.get_row_count() < 1:
            table.add_row()
        while table.get_column_count() < 2:
            table.add_column()
        # Fresh text cells inherit the same default backgroundColor, so
        # selecting two of them should collapse to one swatch.
        table.set_cell_text(0, 0, "group-a")
        table.set_cell_text(0, 1, "group-b")

        table.select_cell(0, 0)
        table.shift_select_cell(0, 1)
        assert table.get_toolbar_cell_count() == 2, (
            "Toolbar breadcrumb should report 2-cell selection before color check"
        )
        assert table.get_color_swatch_count() == 1, (
            f"Expected one color swatch for two same-colored cells, "
            f"got {table.get_color_swatch_count()}"
        )

    def test_multi_cell_level_bulk_apply(self, table: Table) -> None:
        """Test that the toolbar's Size selector bulk-applies a level to
        every selected cell when 2+ cells are selected."""
        self.log("Testing multi-cell level bulk-apply")
        while table.get_row_count() < 1:
            table.add_row()
        while table.get_column_count() < 3:
            table.add_column()
        table.set_cell_text(0, 0, "level-a")
        table.set_cell_text(0, 1, "level-b")
        table.set_cell_text(0, 2, "level-c")

        table.select_cell(0, 0)
        table.shift_select_cell(0, 2)
        assert table.get_toolbar_cell_count() == 3, (
            "Toolbar breadcrumb should report 3-cell selection before level set"
        )

        table.set_toolbar_size("L")
        assert table.get_toolbar_size() == "L", (
            "Toolbar should report common size 'L' after bulk-apply while the "
            "multi-selection is still active"
        )

        for c in (0, 1, 2):
            table.select_cell(0, c)
            assert table.get_toolbar_size() == "L", (
                f"Cell (0,{c}) size should be 'L' after bulk-apply, "
                f"got '{table.get_toolbar_size()}'"
            )

    def test_multi_cell_level_empty_when_disagreed(self, table: Table) -> None:
        """Test that the toolbar's Size selector shows no active button when
        the selected cells disagree on level.

        Regression guard: the v0 toolbar fell back to a synthetic 'p'
        default, which would have falsely highlighted the 'p' button when
        cells disagreed.
        """
        self.log("Testing multi-cell size empty when cells disagree")
        while table.get_row_count() < 1:
            table.add_row()
        while table.get_column_count() < 2:
            table.add_column()
        table.set_cell_text(0, 0, "disagree-a")
        table.set_cell_text(0, 1, "disagree-b")

        # Seed each cell to a different level via single-cell selection.
        table.select_cell(0, 0)
        table.set_toolbar_size("XL")
        table.select_cell(0, 1)
        table.set_toolbar_size("XS")

        # Sanity check the per-cell levels.
        table.select_cell(0, 0)
        assert table.get_toolbar_size() == "XL"
        table.select_cell(0, 1)
        assert table.get_toolbar_size() == "XS"

        table.select_cell(0, 0)
        table.shift_select_cell(0, 1)
        assert table.get_toolbar_cell_count() == 2
        assert table.get_toolbar_size() is None, (
            "Size selector must report no active button when selected cells "
            f"disagree on level, got '{table.get_toolbar_size()}'"
        )
