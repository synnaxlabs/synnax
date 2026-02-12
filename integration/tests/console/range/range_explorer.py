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
from framework.utils import get_random_name


class RangeExplorer(ConsoleCase):
    """Test Range Explorer: context menu operations, multi-select, search, and filter."""

    suffix: str
    label_a_name: str
    label_b_name: str
    range_a_name: str
    range_b_name: str
    range_c_name: str
    range_d_name: str
    range_e_name: str
    child_range_name: str | None = None

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self.label_a_name = f"LabelA_{self.suffix}"
        self.label_b_name = f"LabelB_{self.suffix}"
        self.range_a_name = f"RangeA_{self.suffix}"
        self.range_b_name = f"RangeB_{self.suffix}"
        self.range_c_name = f"RangeC_{self.suffix}"
        self.range_d_name = f"RangeD_{self.suffix}"
        self.range_e_name = f"RangeE_{self.suffix}"

        self.console.labels.create(name=self.label_a_name)
        self.console.labels.create(name=self.label_b_name)

        now = sy.TimeStamp.now()
        tr = sy.TimeRange(now - sy.TimeSpan.HOUR, now + sy.TimeSpan.HOUR)
        for name in [
            self.range_a_name,
            self.range_b_name,
            self.range_c_name,
            self.range_d_name,
            self.range_e_name,
        ]:
            self.client.ranges.create(name=name, time_range=tr)

        self.console.ranges.open_explorer()

    def teardown(self) -> None:
        ranges_to_delete = [
            self.child_range_name,
            self.range_a_name,
            self.range_b_name,
            self.range_c_name,
        ]

        self.console.ranges.open_explorer()
        for name in ranges_to_delete:
            if name and self.console.ranges.exists_in_explorer(name):
                self.console.ranges.delete_from_explorer(name)

        if self.console.labels.exists(self.label_a_name):
            self.console.labels.delete(self.label_a_name)
        if self.console.labels.exists(self.label_b_name):
            self.console.labels.delete(self.label_b_name)

        super().teardown()

    def run(self) -> None:
        """Run all Range Explorer tests."""
        # Context Menu
        self.test_create_child_range()
        self.test_favorite_multiple_ranges()
        self.test_unfavorite_range()
        self.test_unfavorite_multiple_ranges()
        self.test_copy_link()
        self.test_delete_multiple_ranges()

    def test_create_child_range(self) -> None:
        """Test creating a child range via explorer context menu."""
        self.log("Testing: Create child range from explorer context menu")
        self.child_range_name = f"ExplorerChild_{self.suffix}"
        self.console.ranges.create_child_range_from_explorer(self.range_a_name)

        name_input = self.page.locator(
            f"input[placeholder='{self.console.ranges.NAME_INPUT_PLACEHOLDER}']"
        )
        name_input.fill(self.child_range_name)
        save_button = self.page.get_by_role("button", name="Save to Synnax")
        save_button.click(timeout=2000)
        modal = self.page.locator(self.console.ranges.CREATE_MODAL_SELECTOR)
        modal.wait_for(state="hidden", timeout=5000)

        assert self.console.ranges.exists_in_explorer(
            self.child_range_name
        ), f"Child range '{self.child_range_name}' should exist in explorer"

    def test_favorite_multiple_ranges(self) -> None:
        """Test favoriting multiple ranges via multi-select context menu."""
        self.log("Testing: Favorite multiple ranges")
        self.console.ranges.favorite_explorer_ranges(
            [self.range_b_name, self.range_c_name]
        )

        assert self.console.ranges.exists_in_toolbar(
            self.range_b_name
        ), f"'{self.range_b_name}' should appear in toolbar after favoriting"
        assert self.console.ranges.exists_in_toolbar(
            self.range_c_name
        ), f"'{self.range_c_name}' should appear in toolbar after favoriting"

    def test_unfavorite_range(self) -> None:
        """Test unfavoriting a single range via explorer context menu."""
        self.log("Testing: Unfavorite range from explorer")
        self.console.ranges.open_explorer()
        self.console.ranges.unfavorite_from_explorer(self.range_b_name)

        assert self.console.ranges.exists_in_toolbar(
            self.range_c_name
        ), f"'{self.range_c_name}' should still be in toolbar"

    def test_unfavorite_multiple_ranges(self) -> None:
        """Test unfavoriting multiple ranges via multi-select context menu."""
        self.log("Testing: Unfavorite multiple ranges")
        self.console.ranges.open_explorer()
        self.console.ranges.favorite_explorer_ranges([self.range_b_name])

        self.console.ranges.unfavorite_explorer_ranges(
            [self.range_b_name, self.range_c_name]
        )

    def test_copy_link(self) -> None:
        """Test copying link to a range via explorer context menu."""
        self.log("Testing: Copy link from explorer")
        self.console.ranges.open_explorer()
        self.console.ranges.copy_link_from_explorer(self.range_a_name)

        clipboard = self.console.layout.read_clipboard()
        assert len(clipboard) > 0, "Clipboard should not be empty after copying link"

    def test_delete_multiple_ranges(self) -> None:
        """Test deleting multiple ranges via multi-select context menu."""
        self.log("Testing: Delete multiple ranges")
        self.console.ranges.open_explorer()
        self.console.ranges.delete_explorer_ranges(
            [self.range_d_name, self.range_e_name]
        )

        for name in [self.range_d_name, self.range_e_name]:
            try:
                self.client.ranges.retrieve(name=name)
                raise AssertionError(f"Range '{name}' should be deleted from server")
            except sy.QueryError:
                pass
