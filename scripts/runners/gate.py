#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Disconnects the fleet from GitHub without stopping instances, by writing
<runner>_gh_cmd for agent.py to act on. Stepped by controller.py each poll."""

import synnax as sy

from commander import Commander
from common import (
    channel_name,
    create_channels,
    create_index,
    instance_prefix,
    latest,
)
from log import Log
from publisher import Publisher, on_change
from subscription import Subscription

# False closes the gate, true opens it. Stored on its own <name>_time index.
GATE_CMD = "global_gh_cmd"
GATE_ENABLED = "runner_gh_global_set"
GATE_ENABLED_INDEX = "runner_gh_global_set_index"
GATE_LOG = "runner_gh_log"
GATE_STATE_INDEX = "gh_state_time"
DRAIN_TIMEOUT = sy.TimeSpan.MILLISECOND * 10


class Gate(Commander):
    """Fans global_gh_cmd out to every runner, once per change, and reports what is
    on each <runner>_gh_cmd, whoever wrote it, in <runner>_gh_state on change."""

    def __init__(self, client: sy.Synnax, log: Log, enabled: bool = True) -> None:
        super().__init__(client, "gh_cmd", "GH Gate", log)
        self.enabled = enabled
        # What the last fan-out sent, so only a change fans out again.
        self.sent = enabled
        self.known: set[str] = set()
        self.state = Publisher(client, GATE_STATE_INDEX, [], report=log.error)
        self.published: dict[str, bool] = {}
        self.writes = Subscription(client, "runner_gh_cmd", [], report=log.error)

    def state_channel(self, runner: str) -> str:
        return channel_name("gh_state", runner)

    def step(self, known: set[str]) -> None:
        now = sy.TimeStamp.now()
        self.refresh_writer(known)
        self.refresh(known)
        self.follow(now)
        changed = self.enabled != self.sent
        self.sent = self.enabled
        commands = {
            r: self.enabled
            for r in sorted(self.known)
            if (last := self.commanded.get(r)) is None
            or (changed and last[1] != self.enabled)
        }
        if commands:
            self.command(commands, now)
        self.publish(now)

    def refresh(self, known: set[str]) -> None:
        runners = {r for r in known if instance_prefix(r) is not None}
        if runners == self.known:
            return
        self.known = runners
        channels = [self.state_channel(r) for r in sorted(runners)]
        index = create_index(self.client, GATE_STATE_INDEX)
        create_channels(
            self.client, index, dict.fromkeys(channels, sy.DataType.BOOLEAN)
        )
        self.state.retarget([GATE_STATE_INDEX, *channels])
        self.writes.retarget([self.channel(r) for r in sorted(runners)])
        self.seed(runners - set(self.commanded))

    def follow(self, now: sy.TimeStamp) -> None:
        """Takes in every gh_cmd write since the last step."""
        while (frame := self.writes.read(DRAIN_TIMEOUT)) is not None:
            for r in self.known:
                series = frame.get(self.channel(r))
                if series is not None and len(series):
                    self.commanded[r] = (now, bool(series[-1]))

    def seed(self, runners: set[str]) -> None:
        """The last gh_state of each runner, so a restart resends nothing."""
        if not runners:
            return
        now = sy.TimeStamp.now()
        found = latest(self.client, [self.state_channel(r) for r in sorted(runners)])
        for r in runners:
            if (v := found.get(self.state_channel(r))) is not None:
                self.commanded[r] = (now, bool(v))

    def publish(self, now: sy.TimeStamp) -> None:
        current = {
            self.state_channel(r): last[1]
            for r in sorted(self.known)
            if (last := self.commanded.get(r)) is not None
        }
        frame = on_change(GATE_STATE_INDEX, self.published, current, now)
        if frame is not None and self.state.write(frame):
            self.published = current

    def close(self) -> None:
        self.writes.close()
        self.state.close()
        super().close()
