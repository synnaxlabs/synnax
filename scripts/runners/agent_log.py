#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The runner's log channel, for agent.py."""

from pathlib import Path

import synnax as sy

RUNNER_DIRS = (
    Path("C:/actions-runner"),
    Path("C:/Users/Administrator/Desktop/actions-runner"),
    Path("/home/ubuntu/actions-runner"),
)
LOG_DIRS = tuple(p / "_diag" / "pages" for p in RUNNER_DIRS)
LOG_BUDGET = 1 << 20
# Measured values cannot be taken over and spoofed.
PUBLISH_AUTHORITY = sy.Authority.ABSOLUTE


class Log:
    """Tails the actions-runner page logs and streams new lines to a virtual channel.

    Reads at most LOG_BUDGET bytes per cycle and stops each file at its last newline,
    so a line split across two reads arrives whole. Lines queued with log() go first.
    """

    def __init__(self, client: sy.Synnax, name: str) -> None:
        self.name = name
        self.queue: list[str] = []
        self.writer = client.open_writer(
            sy.TimeStamp.now(), [name], authorities=PUBLISH_AUTHORITY, name=name
        )
        self.offsets = self.sizes()

    def log(self, line: str) -> None:
        print(line, flush=True)
        self.queue.append(f"[agent] {line}")

    def sizes(self) -> dict[Path, int]:
        out: dict[Path, int] = {}
        for directory in LOG_DIRS:
            try:
                for path in directory.glob("*.log"):
                    out[path] = path.stat().st_size
            except OSError:
                continue
        return out

    def read(self) -> list[str]:
        sizes = self.sizes()
        self.offsets = {p: o for p, o in self.offsets.items() if p in sizes}
        lines: list[str] = []
        budget = LOG_BUDGET
        for path, size in sorted(sizes.items()):
            offset = self.offsets.get(path, 0)
            if size < offset:
                offset = 0
            if budget <= 0 or size <= offset:
                self.offsets[path] = offset
                continue
            try:
                with path.open("rb") as f:
                    f.seek(offset)
                    raw = f.read(min(size - offset, budget))
            except OSError:
                continue
            cut = raw.rfind(b"\n") + 1
            # A line longer than the budget goes out in pieces, or it stalls its file.
            if cut == 0 and len(raw) >= LOG_BUDGET:
                cut = len(raw)
            if cut == 0:
                continue
            self.offsets[path] = offset + cut
            budget -= cut
            lines.extend(raw[:cut].decode("utf-8", "replace").splitlines())
        return lines

    def write(self) -> None:
        if lines := self.queue + self.read():
            # CrudeFrame's annotation omits name-keyed dicts, which write accepts.
            self.writer.write({self.name: lines})  # pyright: ignore[reportArgumentType]
            self.queue.clear()

    def close(self) -> None:
        self.writer.close()
