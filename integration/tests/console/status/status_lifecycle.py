#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from framework.utils import get_random_name


class StatusLifecycle(ConsoleCase):
    """Test Status Explorer: create, favorite, unfavorite, delete, and filter."""

    suffix: str
    label_a_name: str
    label_b_name: str
    status_a_name: str
    status_b_name: str
    status_c_name: str
    status_d_name: str
    status_d_renamed: str | None = None

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self.label_a_name = f"StatusLabelA_{self.suffix}"
        self.label_b_name = f"StatusLabelB_{self.suffix}"
        self.status_a_name = f"StatusA_{self.suffix}"
        self.status_b_name = f"StatusB_{self.suffix}"
        self.status_c_name = f"StatusC_{self.suffix}"
        self.status_d_name = f"StatusD_{self.suffix}"

        self.console.labels.create(name=self.label_a_name)
        self.console.labels.create(name=self.label_b_name)

        self.console.statuses.create(
            self.status_a_name, labels=[self.label_a_name, self.label_b_name]
        )
        self.console.statuses.create(self.status_b_name, labels=[self.label_b_name])
        self.console.statuses.create(self.status_c_name)
        self.console.statuses.create(self.status_d_name)

        self.console.statuses.open_explorer()

    def teardown(self) -> None:
        all_names = [
            self.status_a_name,
            self.status_b_name,
            self.status_c_name,
            self.status_d_name,
            self.status_d_renamed,
            f"NotifyStatus_{self.suffix}",
        ]
        for name in filter(None, all_names):
            statuses = self.client.statuses.retrieve(search_term=name)
            if len(statuses) > 0:
                self.client.statuses.delete([s.key for s in statuses])
        self.console.labels.delete(self.label_a_name)
        self.console.labels.delete(self.label_b_name)
        super().teardown()

    def run(self) -> None:
        """Run all Status Explorer tests."""
        # Notifications
        self.test_status_notification()

        # Explorer
        self.test_status_exists_in_explorer()
        self.test_status_has_labels()
        self.test_filter_by_labels()
        self.test_favorite_unfavorite_from_explorer()
        self.test_delete_single_status()
        self.test_delete_multiple_statuses()

        # Toolbar
        self.test_rename_from_toolbar()
        self.test_unfavorite_from_toolbar()
        self.test_delete_from_toolbar()

    def test_status_notification(self) -> None:
        """Test that creating a status shows a notification in the bottom right."""
        self.log("Testing: Status notification appears on create")
        notification_status_name = f"NotifyStatus_{self.suffix}"
        self.console.statuses.notifications.close_all()
        self.console.statuses.create(notification_status_name)
        assert self.console.statuses.notifications.wait_for(
            notification_status_name
        ), f"Notification for '{notification_status_name}' should appear"

    def test_status_exists_in_explorer(self) -> None:
        """Test that created statuses appear in the explorer."""
        self.log("Testing: Statuses exist in explorer")
        assert self.console.statuses.exists_in_explorer(
            self.status_a_name
        ), f"Status '{self.status_a_name}' should exist in explorer"
        assert self.console.statuses.exists_in_explorer(
            self.status_b_name
        ), f"Status '{self.status_b_name}' should exist in explorer"

    def test_status_has_labels(self) -> None:
        """Test that status_a displays both labels in the explorer."""
        self.log("Testing: Status has labels in explorer")
        labels = self.console.statuses.get_labels_in_explorer(self.status_a_name)
        assert self.label_a_name in labels, (
            f"Label '{self.label_a_name}' should be displayed on status "
            f"'{self.status_a_name}'. Found labels: {labels}"
        )
        assert self.label_b_name in labels, (
            f"Label '{self.label_b_name}' should be displayed on status "
            f"'{self.status_a_name}'. Found labels: {labels}"
        )

    def test_filter_by_labels(self) -> None:
        """Test filtering statuses by label in the explorer."""
        self.log("Testing: Filter statuses by label")
        self.console.statuses.open_explorer()
        self.console.statuses.select_explorer_label_filter(self.label_a_name)

        assert self.console.statuses.exists_in_explorer(
            self.status_a_name
        ), f"'{self.status_a_name}' should be visible when filtering by its label"

        self.console.statuses.wait_for_removed_from_explorer(self.status_c_name)

        self.console.statuses.clear_explorer_label_filter(self.label_a_name)

    def test_favorite_unfavorite_from_explorer(self) -> None:
        """Test favoriting then unfavoriting a status via the explorer context menu."""
        self.log("Testing: Favorite and unfavorite status from explorer")
        self.console.statuses.favorite_from_explorer(self.status_a_name)

        assert self.console.statuses.exists_in_toolbar(
            self.status_a_name
        ), f"'{self.status_a_name}' should appear in toolbar after favoriting"

        self.console.statuses.unfavorite_from_explorer(self.status_a_name)

        self.console.statuses.wait_for_removed_from_toolbar(self.status_a_name)

    def test_delete_single_status(self) -> None:
        """Test deleting a single status via the explorer context menu."""
        self.log("Testing: Delete single status from explorer")
        self.console.statuses.delete_from_explorer(self.status_c_name)

    def test_delete_multiple_statuses(self) -> None:
        """Test deleting multiple statuses via multi-select context menu."""
        self.log("Testing: Delete multiple statuses from explorer")
        self.console.statuses.delete_explorer_statuses(
            [self.status_a_name, self.status_b_name]
        )

    # ── Toolbar ────────────────────────────────────────────────────────────

    def test_rename_from_toolbar(self) -> None:
        """Test renaming a status from the toolbar and verifying sync in explorer."""
        self.log("Testing: Rename status from toolbar")
        self.console.statuses.open_explorer()
        self.console.statuses.favorite_from_explorer(self.status_d_name)
        self.status_d_renamed = f"StatusD_Renamed_{self.suffix}"
        self.console.statuses.rename_from_toolbar(
            self.status_d_name, self.status_d_renamed
        )

        self.console.statuses.open_explorer()
        assert self.console.statuses.exists_in_explorer(
            self.status_d_renamed
        ), f"'{self.status_d_renamed}' should appear in explorer after rename"

    def test_unfavorite_from_toolbar(self) -> None:
        """Test unfavoriting a status from the toolbar."""
        self.log("Testing: Unfavorite status from toolbar")
        assert self.status_d_renamed is not None
        self.console.statuses.unfavorite_from_toolbar(self.status_d_renamed)
        self.console.statuses.wait_for_removed_from_toolbar(self.status_d_renamed)

    def test_delete_from_toolbar(self) -> None:
        """Test deleting a status from the toolbar and verifying sync in explorer."""
        self.log("Testing: Delete status from toolbar")
        assert self.status_d_renamed is not None
        self.console.statuses.open_explorer()
        self.console.statuses.favorite_from_explorer(self.status_d_renamed)
        self.console.statuses.delete_from_toolbar(self.status_d_renamed)

        self.console.statuses.open_explorer()
        self.console.statuses.wait_for_removed_from_explorer(self.status_d_renamed)
