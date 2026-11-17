#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The runners seen so far and their last labels, remembered across restarts so a
removed runner keeps reporting, the scaler can start a runner whose registration it
deleted, and dispatch.py knows which runners exist."""

import json
import threading
from pathlib import Path

from common import instance_prefix

STATE_FILE = Path(__file__).with_name("registry.json")


class RunnerRegistry:
    """Runners seen so far and their last labels, shared between the poll threads and
    persisted to path so other scripts can read the names with known_runners()."""

    def __init__(self, path: Path = STATE_FILE) -> None:
        self.path = path
        self.lock = threading.Lock()
        self.entries = read(path)

    def snapshot(self) -> set[str]:
        with self.lock:
            return set(self.entries)

    def labels(self) -> dict[str, set[str]]:
        """The labels each runner last registered with."""
        with self.lock:
            return {r: set(labels) for r, labels in self.entries.items()}

    def add(self, labels: dict[str, set[str]]) -> None:
        """Adds the runners GitHub lists and replaces their labels. Drops the other
        names of each listed runner's instance, so a renamed instance has one entry."""
        with self.lock:
            listed = {p for r in labels if (p := instance_prefix(r)) is not None}
            self.entries = {
                r: v
                for r, v in self.entries.items()
                if r in labels or instance_prefix(r) not in listed
            } | {r: set(v) for r, v in labels.items()}
            # Replaced whole, so a stop mid-write leaves the previous file intact.
            tmp = self.path.with_suffix(".tmp")
            entries = {r: sorted(v) for r, v in sorted(self.entries.items())}
            tmp.write_text(json.dumps(entries, indent=2))
            tmp.replace(self.path)


def read(path: Path) -> dict[str, set[str]]:
    if not path.exists():
        return {}
    stored = json.loads(path.read_text())
    # A file written before labels were kept is a list of names.
    if isinstance(stored, list):
        return {r: set() for r in stored}
    return {r: set(labels) for r, labels in stored.items()}


def known_runners(path: Path = STATE_FILE) -> set[str]:
    """Returns the runner names remembered in path, empty if none yet."""
    return set(read(path))
