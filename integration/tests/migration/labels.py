#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test: create labels on old version, verify after migration."""

from abc import abstractmethod

from console.case import ConsoleCase

LABEL_1_NAME = "mig_label_bug"
LABEL_1_COLOR = "#E63946"

LABEL_2_NAME = "mig_label_feature"
LABEL_2_COLOR = "#457B9D"

LABEL_3_NAME = "mig_label_urgent"
LABEL_3_COLOR = "#2A9D8F"

LABELS: list[tuple[str, str]] = [
    (LABEL_1_NAME, LABEL_1_COLOR),
    (LABEL_2_NAME, LABEL_2_COLOR),
    (LABEL_3_NAME, LABEL_3_COLOR),
]

RANGE_NAME = "mig_label_range"


class LabelsMigration(ConsoleCase):
    """Base class defining the migration test contract for labels.

    Subclasses must implement each test method — setup creates the state,
    verify checks it after migration.
    """

    def run(self) -> None:
        self.test_labels()
        self.test_labels_on_range()

    @abstractmethod
    def test_labels(self) -> None: ...

    @abstractmethod
    def test_labels_on_range(self) -> None: ...


class LabelsSetup(LabelsMigration):
    """Create labels and attach them to a range."""

    def test_labels(self) -> None:
        self.log("Testing: Create labels")
        for name, color in LABELS:
            self.console.labels.create(name, color=color)
            assert self.console.labels.exists(name), f"Label '{name}' was not created"
            actual_color = self.console.labels.get_color(name)
            assert actual_color == color, (
                f"Label '{name}' color: '{actual_color}' != '{color}'"
            )

    def test_labels_on_range(self) -> None:
        self.log("Testing: Create range with labels")
        label_names = [name for name, _ in LABELS]
        self.console.ranges.create(RANGE_NAME, persisted=True, labels=label_names)
        self.console.ranges.open_explorer()
        for name, _ in LABELS:
            assert self.console.ranges.label_exists_in_toolbar(RANGE_NAME, name), (
                f"Label '{name}' not on range '{RANGE_NAME}'"
            )


class LabelsVerify(LabelsMigration):
    """Verify labels and range attachment survived migration."""

    def test_labels(self) -> None:
        self.log("Testing: Labels survived migration")
        for name, color in LABELS:
            assert self.console.labels.exists(name), (
                f"Label '{name}' not found after migration"
            )
            actual_color = self.console.labels.get_color(name)
            assert actual_color == color, (
                f"Label '{name}' color: '{actual_color}' != '{color}'"
            )

    def test_labels_on_range(self) -> None:
        self.log("Testing: Labels on range survived migration")
        self.console.ranges.open_explorer()
        self.console.ranges.favorite_from_explorer(RANGE_NAME)
        for name, _ in LABELS:
            assert self.console.ranges.label_exists_in_toolbar(RANGE_NAME, name), (
                f"Label '{name}' not on range '{RANGE_NAME}' after migration"
            )
