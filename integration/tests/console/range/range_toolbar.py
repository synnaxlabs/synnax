#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable

import synnax as sy
from console.case import ConsoleCase
from x import random_name


class RangeToolbar(ConsoleCase):
    """Test the live range controls of the ranges toolbar: Start, Start child range,
    Next, and Stop. Each check reads the result from the Core."""

    suffix: str
    run_name: str
    phase_1_name: str
    phase_2_name: str

    def setup(self) -> None:
        super().setup()
        self.suffix = random_name()
        self.run_name = f"ToolbarRun_{self.suffix}"
        self.phase_1_name = f"Phase_{self.suffix}_1"
        self.phase_2_name = f"Phase_{self.suffix}_2"
        self.console.ranges.toolbar.show()

    def teardown(self) -> None:
        if self.console.layout.is_modal_open():
            self.page.keyboard.press("Escape")
        with self._try_to("delete ranges"):
            names = [self.run_name, self.phase_1_name, self.phase_2_name]
            keys = [r.key for r in self.client.ranges.retrieve(names=names)]
            if keys:
                self.client.ranges.delete(keys)
        super().teardown()

    def run(self) -> None:
        """Run all ranges toolbar tests."""
        self.test_start_range()
        self.test_start_child_range()
        self.test_next()
        self.test_stop()

    def wait_for_range(
        self, name: str, *, done: Callable[[sy.Range], bool]
    ) -> sy.Range:
        """Poll until the Core holds the range and it satisfies ``done``. The toolbar
        writes to the Core after Enter, so an immediate read races it."""

        def retrieve() -> sy.Range | None:
            try:
                rng = self.client.ranges.retrieve(name=name)
            except sy.NotFoundError:
                return None
            return rng if done(rng) else None

        rng = sy.poll(retrieve, timeout=5, interval=0.5)
        if rng is None:
            last = self.client.ranges.retrieve(name=name)
            raise AssertionError(
                f"'{name}' did not reach the expected state: parent {last.parent}, "
                f"time range {int(last.time_range.start)} to "
                f"{int(last.time_range.end)}"
            )
        return rng

    def test_start_range(self) -> None:
        """Test that Start creates a running top-level range."""
        self.log("Testing: Start range")
        self.console.ranges.toolbar.start(self.run_name)
        rng = self.wait_for_range(
            self.run_name, done=lambda r: r.time_range.end == sy.TimeStamp.MAX
        )
        assert rng.parent is None, f"'{self.run_name}' should have no parent"
        assert self.console.ranges.toolbar.exists(self.run_name, group="Running"), (
            f"'{self.run_name}' should show in Running"
        )

    def test_start_child_range(self) -> None:
        """Test that Start child range creates a running child of the range."""
        self.log("Testing: Start child range")
        self.console.ranges.toolbar.start_child(self.run_name, self.phase_1_name)
        run = self.client.ranges.retrieve(name=self.run_name)
        self.wait_for_range(
            self.phase_1_name,
            done=lambda r: r.parent == run.key and r.time_range.end == sy.TimeStamp.MAX,
        )

    def test_next(self) -> None:
        """Test that Next suggests the next number, stops the child, and starts the
        next child at the same time."""
        self.log("Testing: Next")
        suggested = self.console.ranges.toolbar.next(
            self.phase_1_name, self.phase_2_name
        )
        assert suggested == self.phase_2_name, (
            f"Next should suggest '{self.phase_2_name}', got '{suggested}'"
        )
        run = self.client.ranges.retrieve(name=self.run_name)
        phase_2 = self.wait_for_range(
            self.phase_2_name,
            done=lambda r: r.parent == run.key and r.time_range.end == sy.TimeStamp.MAX,
        )
        self.wait_for_range(
            self.phase_1_name,
            done=lambda r: r.time_range.end == phase_2.time_range.start,
        )

    def test_stop(self) -> None:
        """Test that Stop ends a running range now."""
        self.log("Testing: Stop")
        self.console.ranges.toolbar.stop(self.phase_2_name)
        self.wait_for_range(
            self.phase_2_name,
            done=lambda r: r.time_range.end < sy.TimeStamp.now(),
        )
