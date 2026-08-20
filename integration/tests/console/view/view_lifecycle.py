#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable
from dataclasses import dataclass

import synnax as sy
from console.case import ConsoleCase
from console.views import ViewsClient
from x import random_name


@dataclass(frozen=True)
class _Explorer:
    """One explorer under test, with the item operations the suite needs."""

    name: str
    plural: str
    views: ViewsClient
    open: Callable[[], None]
    exists: Callable[[str], bool]
    wait_for_removed: Callable[[str], None]
    delete_item: Callable[[str], None]
    relabel: Callable[[str, str], None] | None
    labeled_a: str
    labeled_b: str
    unlabeled: str


class ViewLifecycle(ConsoleCase):
    """Test views in the Range and Status Explorers. Two views with disjoint label
    filters prove each holds its own query; the same suite runs once per explorer."""

    suffix: str
    label_a_name: str
    label_b_name: str
    range_a_name: str
    range_b_name: str
    range_n_name: str
    status_a_name: str
    status_b_name: str
    status_n_name: str
    view_names: list[str]

    def setup(self) -> None:
        self.suffix = random_name()
        self.label_a_name = f"ViewLabelA_{self.suffix}"
        self.label_b_name = f"ViewLabelB_{self.suffix}"
        self.range_a_name = f"ViewRangeA_{self.suffix}"
        self.range_b_name = f"ViewRangeB_{self.suffix}"
        self.range_n_name = f"ViewRangeN_{self.suffix}"
        self.status_a_name = f"ViewStatusA_{self.suffix}"
        self.status_b_name = f"ViewStatusB_{self.suffix}"
        self.status_n_name = f"ViewStatusN_{self.suffix}"
        self.view_names = []
        super().setup()

        # The Python client has no label client, so the labels and the labeled
        # resources go through the Console. Unlabeled ones take the fast path.
        now = sy.TimeStamp.now()
        tr = sy.TimeRange(now - sy.TimeSpan.HOUR, now + sy.TimeSpan.HOUR)
        self.client.ranges.create(name=self.range_n_name, time_range=tr)
        self.client.statuses.set(
            sy.Status(
                variant=sy.status.VARIANT_INFO,
                message="Unlabeled status for view test",
                name=self.status_n_name,
            )
        )
        self.console.labels.create(name=self.label_a_name)
        self.console.labels.create(name=self.label_b_name)
        self.console.ranges.create(
            self.range_a_name, persisted=True, labels=[self.label_a_name]
        )
        self.console.ranges.create(
            self.range_b_name, persisted=True, labels=[self.label_b_name]
        )
        self.console.statuses.create(self.status_a_name, labels=[self.label_a_name])
        self.console.statuses.create(self.status_b_name, labels=[self.label_b_name])

    def teardown(self) -> None:
        for name in self.view_names:
            views = self.client.views.retrieve(search_term=name)
            keys = [v.key for v in views if v.name == name]
            if len(keys) > 0:
                self.client.views.delete(keys)
        for name in [self.range_a_name, self.range_b_name, self.range_n_name]:
            self._delete_range(name)
        for name in [self.status_a_name, self.status_b_name, self.status_n_name]:
            self._delete_status(name)
        self.console.labels.delete(self.label_a_name)
        self.console.labels.delete(self.label_b_name)
        super().teardown()

    def _delete_range(self, name: str) -> None:
        keys = [r.key for r in self.client.ranges.search(name) if r.name == name]
        if len(keys) > 0:
            self.client.ranges.delete(keys)

    def _delete_status(self, name: str) -> None:
        statuses = self.client.statuses.retrieve(search_term=name)
        keys = [s.key for s in statuses if s.name == name]
        if len(keys) > 0:
            self.client.statuses.delete(keys)

    def _range_removed(self, name: str) -> None:
        """Wait for a range row to leave the list without the explorer's scroll
        sweep. Filtered views are small, fully mounted lists."""
        ranges = self.console.ranges.explorer
        self.console.layout.wait_for_hidden(
            self.console.layout.get_list_item(ranges.ITEM_SELECTOR, name)
        )

    def _relabel_range(self, name: str, label_name: str) -> None:
        """Add a label to a range via its overview, then return to the explorer."""
        self.console.ranges.overview.open(name)
        self.console.ranges.overview.add_label(label_name)
        self.console.layout.close_tab(name)
        self.console.ranges.explorer.open()

    def run(self) -> None:
        """Run the view suite against each explorer."""
        ranges = self.console.ranges.explorer
        self._run_suite(
            _Explorer(
                name="Range",
                plural="ranges",
                views=ranges.views,
                open=ranges.open,
                exists=ranges.exists,
                wait_for_removed=self._range_removed,
                delete_item=self._delete_range,
                relabel=self._relabel_range,
                labeled_a=self.range_a_name,
                labeled_b=self.range_b_name,
                unlabeled=self.range_n_name,
            )
        )
        self.console.layout.close_tab("Range explorer")

        # Statuses have no label-edit path: the Console has no UI for it and the
        # Python client has no label client. Relabel is skipped for them.
        statuses = self.console.statuses
        self._run_suite(
            _Explorer(
                name="Status",
                plural="statuses",
                views=statuses.views,
                open=statuses.open_explorer,
                exists=statuses.exists_in_explorer,
                wait_for_removed=statuses.wait_for_removed_from_explorer,
                delete_item=self._delete_status,
                relabel=None,
                labeled_a=self.status_a_name,
                labeled_b=self.status_b_name,
                unlabeled=self.status_n_name,
            )
        )

    def _run_suite(self, explorer: _Explorer) -> None:
        view_a = f"{explorer.name}ViewA_{self.suffix}"
        view_b = f"{explorer.name}ViewB_{self.suffix}"
        renamed = f"{explorer.name}ViewARenamed_{self.suffix}"
        self.view_names.extend([view_a, view_b, renamed])

        explorer.open()
        self.test_create_views(explorer, view_a, view_b)
        self.test_switch_between_views(explorer, view_a, view_b)
        self.test_relabel_moves_item(explorer, view_a, view_b)
        self.test_delete_item_leaves_view(explorer, view_a, view_b)
        self.test_rename_view(explorer, view_a, renamed)
        self.test_delete_view(explorer, renamed, view_b)

    def _assert_only(
        self, explorer: _Explorer, shown: list[str], hidden: list[str]
    ) -> None:
        for name in shown:
            assert explorer.exists(name), (
                f"'{name}' should be visible in the {explorer.name} Explorer"
            )
        for name in hidden:
            explorer.wait_for_removed(name)

    def test_create_views(self, explorer: _Explorer, view_a: str, view_b: str) -> None:
        """Test creating two views, each filtered to a different label."""
        self.log(f"Testing: Create views in {explorer.name} Explorer")
        views = explorer.views
        views.create(view_a)
        assert views.is_selected(view_a), f"'{view_a}' should be selected on create"
        views.select_filter("Select labels", self.label_a_name)
        self._assert_only(
            explorer,
            shown=[explorer.labeled_a],
            hidden=[explorer.labeled_b, explorer.unlabeled],
        )

        views.create(view_b)
        assert views.is_selected(view_b), f"'{view_b}' should be selected on create"
        assert explorer.exists(explorer.labeled_b), (
            f"New view '{view_b}' should start unfiltered"
        )
        views.select_filter("Select labels", self.label_b_name)
        self._assert_only(
            explorer,
            shown=[explorer.labeled_b],
            hidden=[explorer.labeled_a, explorer.unlabeled],
        )

        saved = [
            v
            for v in self.client.views.retrieve(search_term=view_b)
            if v.name == view_b
        ]
        assert len(saved) == 1 and len(saved[0].query) > 0, (
            f"'{view_b}' should be saved to the server with its label filter"
        )

    def test_switch_between_views(
        self, explorer: _Explorer, view_a: str, view_b: str
    ) -> None:
        """Test each view keeps its own filter when switching between them."""
        self.log(f"Testing: Switch between views in {explorer.name} Explorer")
        views = explorer.views
        views.select(view_a)
        self._assert_only(
            explorer,
            shown=[explorer.labeled_a],
            hidden=[explorer.labeled_b, explorer.unlabeled],
        )
        views.select(view_b)
        self._assert_only(
            explorer, shown=[explorer.labeled_b], hidden=[explorer.labeled_a]
        )
        views.select(views.static_view_name)
        self._assert_only(
            explorer,
            shown=[explorer.labeled_a, explorer.labeled_b, explorer.unlabeled],
            hidden=[],
        )

    def test_relabel_moves_item(
        self, explorer: _Explorer, view_a: str, view_b: str
    ) -> None:
        """Test adding a label to an item makes it appear in that label's view."""
        if explorer.relabel is None:
            self.log(f"Skipping: Relabel in {explorer.name} Explorer (no label edit)")
            return
        self.log(f"Testing: Relabel moves item in {explorer.name} Explorer")
        views = explorer.views
        views.select(views.static_view_name)
        explorer.relabel(explorer.unlabeled, self.label_a_name)
        views.select(view_a)
        self._assert_only(
            explorer,
            shown=[explorer.labeled_a, explorer.unlabeled],
            hidden=[explorer.labeled_b],
        )
        views.select(view_b)
        self._assert_only(
            explorer, shown=[explorer.labeled_b], hidden=[explorer.unlabeled]
        )

    def test_delete_item_leaves_view(
        self, explorer: _Explorer, view_a: str, view_b: str
    ) -> None:
        """Test deleting an item removes it from its view, leaving other views alone."""
        self.log(f"Testing: Delete item leaves view in {explorer.name} Explorer")
        views = explorer.views
        views.select(view_b)
        explorer.delete_item(explorer.labeled_b)
        explorer.wait_for_removed(explorer.labeled_b)
        empty = self.page.get_by_text(f"No {explorer.plural} found")
        assert empty.is_visible(), f"'{view_b}' should show the empty state"
        views.select(view_a)
        assert explorer.exists(explorer.labeled_a), (
            f"'{explorer.labeled_a}' should still be visible in '{view_a}'"
        )

    def test_rename_view(
        self, explorer: _Explorer, view_name: str, renamed: str
    ) -> None:
        """Test renaming a view in place via its context menu."""
        self.log(f"Testing: Rename view in {explorer.name} Explorer")
        explorer.views.rename(view_name, renamed)
        assert explorer.views.exists(renamed), (
            f"View '{renamed}' should appear after rename"
        )
        explorer.views.wait_for_removed(view_name)

    def test_delete_view(self, explorer: _Explorer, view_name: str, other: str) -> None:
        """Test deleting the selected view snaps back to the static view and leaves
        other views intact."""
        self.log(f"Testing: Delete view in {explorer.name} Explorer")
        views = explorer.views
        views.select(view_name)
        views.delete(view_name)
        static = views.static_view_name
        assert views.is_selected(static), (
            f"'{static}' should be selected after deleting '{view_name}'"
        )
        assert views.exists(other), f"'{other}' should survive deleting '{view_name}'"
        remaining = [
            v
            for v in self.client.views.retrieve(search_term=view_name)
            if v.name == view_name
        ]
        assert len(remaining) == 0, f"View '{view_name}' should be deleted from server"
