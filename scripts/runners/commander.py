#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""What the scaler and the gate share: a writer over one command channel per runner,
and a memory of what was sent to whom."""

from typing import Protocol

import synnax as sy

from common import (
    COMMAND_AUTHORITY,
    RECOMMAND_DELAY,
    RELEASED_AUTHORITY,
    channel_name,
    command_channels,
)
from log import Log
from publisher import Publisher


class Describe(Protocol):
    def __call__(self, runner: str, /, *, value: bool, now: sy.TimeStamp) -> str: ...


class Commander:
    """Writes <runner>_<command> at COMMAND_AUTHORITY for just the instant of a write,
    so the Console (absolute) always wins, and logs each send; the same value twice
    to one runner is marked [WARN]. describe formats one command for the log."""

    def __init__(
        self,
        *,
        client: sy.Synnax,
        command: str,
        name: str,
        log: Log,
        describe: Describe | None = None,
    ) -> None:
        self.command_name = command
        self.log = log
        self.describe = describe or self._describe
        # Time and value of the last command that landed on each runner.
        self.commanded: dict[str, tuple[sy.TimeStamp, bool]] = {}
        self.commands = Publisher(
            client=client,
            name=name,
            channels=[],
            authority=RELEASED_AUTHORITY,
            report=log.error,
        )

    def _describe(self, runner: str, /, *, value: bool, now: sy.TimeStamp) -> str:
        return f"{self.channel(runner)}={value}"

    def channel(self, runner: str) -> str:
        return channel_name(self.command_name, runner=runner)

    def refresh_writer(self, known: set[str]) -> None:
        """Holds the writer open over the current runner set."""
        self.commands.retarget(
            [
                c
                for name in command_channels(known, commands=(self.command_name,))
                for c in (f"{name}_time", name)
            ]
        )
        self.commands.open()

    def close(self) -> None:
        self.commands.close()
        self.log.close()

    def ready(self, runners: set[str], *, now: sy.TimeStamp) -> set[str]:
        """The runners not commanded within RECOMMAND_DELAY."""
        return {
            r
            for r in runners
            if (last := self.commanded.get(r)) is None
            or now.span(last[0]) >= RECOMMAND_DELAY
        }

    def command(self, commands: dict[str, bool], *, now: sy.TimeStamp) -> bool:
        """Writes commands, returning whether they landed. A failed write records
        nothing, so the caller sends the same commands again."""
        frame: dict[str, object] = {}
        for r, v in commands.items():
            frame[self.channel(r)] = v
            frame[f"{self.channel(r)}_time"] = now
        if not self.commands.write(frame, authority=COMMAND_AUTHORITY):
            return False
        again = {
            r: v
            for r, v in commands.items()
            if (last := self.commanded.get(r)) is not None and last[1] == v
        }
        first = {r: v for r, v in commands.items() if r not in again}
        for r, v in commands.items():
            self.commanded[r] = (now, v)
        if first:
            self.log(self.lines(first, now=now))
        if again:
            self.log(f"[WARN] resending {self.lines(again, now=now)}")
        return True

    def lines(self, commands: dict[str, bool], *, now: sy.TimeStamp) -> str:
        return ", ".join(
            self.describe(r, value=v, now=now) for r, v in sorted(commands.items())
        )
