#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from framework.utils import get_fixture_path, get_random_name


class Ontology(ConsoleCase):
    """Test workspace ontology operations: groups, nesting, and drag-to-move."""

    suffix: str
    page_a: str
    page_b: str
    page_c: str
    page_d: str
    group_a: str
    group_b: str

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self.group_a = f"Group A {self.suffix}"
        self.group_b = f"Group B {self.suffix}"

    def create_test_pages(self) -> None:
        """Create a mix of page types for grouping tests."""
        self.page_a = f"Ontology Schematic {self.suffix}"
        json_path = get_fixture_path("ImportSpace/Metrics Schematic.json")
        self.console.workspace.import_page(json_path, self.page_a)
        self.console.layout.close_tab(self.page_a)

        self.page_b = f"Ontology Log {self.suffix}"
        json_path = get_fixture_path("ImportSpace/Metrics Log.json")
        self.console.workspace.import_page(json_path, self.page_b)
        self.console.layout.close_tab(self.page_b)

        self.page_c = f"Ontology Table {self.suffix}"
        table = self.console.workspace.create_table(self.page_c)
        self.page_c = table.page_name
        table.close()

        self.page_d = f"Ontology Plot {self.suffix}"
        json_path = get_fixture_path("ImportSpace/Metrics Plot.json")
        self.console.workspace.import_page(json_path, self.page_d)
        self.console.layout.close_tab(self.page_d)

    def run(self) -> None:
        """Run all ontology tests."""
        self.create_test_pages()
        self.test_create_group()
        self.test_move_page_to_group()
        self.test_rename_group()
        self.test_create_nested_group()
        self.test_delete_groups()

    def test_create_group(self) -> None:
        """Test creating a group from multiple pages via multi-select."""
        self.log("Testing create group")
        self.console.workspace.group_pages(
            names=[self.page_a, self.page_b],
            group_name=self.group_a,
        )
        assert self.console.workspace.page_exists(
            self.group_a
        ), f"Group '{self.group_a}' should exist after creation"

    def test_move_page_to_group(self) -> None:
        """Test moving a page into a group via drag-and-drop."""
        self.log("Testing move page to group")
        self.console.workspace.move_to_group(self.page_c, self.group_a)

        self.console.workspace.expand_active()
        group_item = self.console.workspace.get_page(self.group_a)
        if not self.console.workspace.tree.is_expanded(group_item):
            self.console.workspace.tree.expand(group_item)

        page_item = self.console.workspace.get_page(self.page_c)
        assert (
            page_item.is_visible()
        ), f"Page '{self.page_c}' should be visible inside '{self.group_a}'"
        self.console.layout.close_left_toolbar()

    def test_rename_group(self) -> None:
        """Test renaming a group via context menu."""
        self.log("Testing rename group")
        new_name = f"Renamed Group {self.suffix}"
        self.console.workspace.rename_group(self.group_a, new_name)
        self.group_a = new_name

        assert self.console.workspace.page_exists(
            self.group_a
        ), f"Renamed group '{self.group_a}' should exist"

    def test_create_nested_group(self) -> None:
        """Test creating a nested group inside an existing group."""
        self.log("Testing create nested group")

        # Pages are inside group_a, so expand it before grouping.
        self.console.workspace.expand_active()
        group_item = self.console.workspace.get_page(self.group_a)
        if not self.console.workspace.tree.is_expanded(group_item):
            self.console.workspace.tree.expand(group_item)

        items = [self.console.workspace.get_page(n) for n in [self.page_a, self.page_b]]
        self.console.workspace.tree.group(items, self.group_b)
        self.console.layout.close_left_toolbar()

        # group_b was created inside group_a. Expand both to verify.
        self.console.workspace.expand_active()
        group_item = self.console.workspace.get_page(self.group_a)
        if not self.console.workspace.tree.is_expanded(group_item):
            self.console.workspace.tree.expand(group_item)

        nested = self.console.workspace.get_page(self.group_b)
        assert (
            nested.is_visible()
        ), f"Nested group '{self.group_b}' should be visible inside '{self.group_a}'"
        self.console.layout.close_left_toolbar()

    def test_delete_groups(self) -> None:
        """Test deleting groups via context menu."""
        self.log("Testing delete groups")

        # Expand group_a to access nested group_b.
        self.console.workspace.expand_active()
        parent = self.console.workspace.get_page(self.group_a)
        if not self.console.workspace.tree.is_expanded(parent):
            self.console.workspace.tree.expand(parent)

        nested = self.console.workspace.tree.get_group(self.group_b)
        self.console.workspace.tree.delete_group(nested)
        self.console.layout.close_left_toolbar()

        self.console.workspace.delete_group(self.group_a)

    def teardown(self) -> None:
        self.console.workspace.expand_active()
        for name in [self.page_a, self.page_b, self.page_c, self.page_d]:
            item = self.console.workspace.get_page(name)
            if item.count() > 0 and item.is_visible():
                self.console.workspace.delete_page(name)
        super().teardown()
