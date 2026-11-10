#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The runner names seen so far, remembered across restarts so a removed runner keeps
reporting -1 and dispatch.py knows which runners exist."""

import json
import threading
from pathlib import Path

STATE_FILE = Path(__file__).with_name("registry.json")


class RunnerRegistry:
    """Runner names seen so far, shared between the poll threads and persisted to path
    so other scripts can read them with known_runners()."""

    def __init__(self, path: Path = STATE_FILE) -> None:
        self.path = path
        self.lock = threading.Lock()
        self.names = known_runners(path)

    def snapshot(self) -> set[str]:
        with self.lock:
            return set(self.names)

    def add(self, names: set[str]) -> None:
        with self.lock:
            self.names |= names
            # Replaced whole, so a stop mid-write leaves the previous file intact.
            tmp = self.path.with_suffix(".tmp")
            tmp.write_text(json.dumps(sorted(self.names), indent=2))
            tmp.replace(self.path)


def known_runners(path: Path = STATE_FILE) -> set[str]:
    """Returns the runner names remembered in path, empty if none yet."""
    if not path.exists():
        return set()
    return set(json.loads(path.read_text()))
