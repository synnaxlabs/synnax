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
from common import Channels, channel_name, instance_prefix, latest
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


class Gate:
    """Fans global_gh_cmd out to every runner, once per change, and reports what is
    on each <runner>_gh_cmd, whoever wrote it, in <runner>_gh_state on change. A
    fan-out whose write fails is sent again on the next step."""

    def __init__(
        self, *, client: sy.Synnax, channels: Channels, log: Log, enabled: bool = True
    ) -> None:
        self.commander = Commander(
            client=client, command="gh_cmd", name="GH Gate", log=log
        )
        self.client = client
        self.channels = channels
        self.log = log
        self.enabled = enabled
        # What the last fan-out sent, so only a change fans out again. None until the
        # first, which applies the start state over the runners' saved ones.
        self.sent: bool | None = None
        self.known: set[str] = set()
        self.state = Publisher(
            client=client, name=GATE_STATE_INDEX, channels=[], report=log.error
        )
        self.published: dict[str, bool] = {}
        self.writes = Subscription(
            client=client, name="runner_gh_cmd", channels=[], report=log.error
        )

    def state_channel(self, runner: str) -> str:
        return channel_name("gh_state", runner=runner)

    def step(self, known: set[str]) -> None:
        self.commander.refresh_writer(known)
        self.refresh(known)
        # After the command writer opens, so commands land after its start.
        now = sy.TimeStamp.now()
        self.follow(now)
        changed = self.enabled != self.sent
        commanded = self.commander.commanded
        commands = {
            r: self.enabled
            for r in sorted(self.known)
            if (last := commanded.get(r)) is None
            or (changed and last[1] != self.enabled)
        }
        if not commands or self.commander.command(commands, now=now):
            self.sent = self.enabled
        self.publish()

    def refresh(self, known: set[str]) -> None:
        runners = {r for r in known if instance_prefix(r) is not None}
        if runners == self.known:
            return
        self.known = runners
        names = [self.state_channel(r) for r in sorted(runners)]
        index = self.channels.index(GATE_STATE_INDEX)
        self.channels.data(
            index=index, channels=dict.fromkeys(names, sy.DataType.BOOLEAN)
        )
        self.state.retarget([GATE_STATE_INDEX, *names])
        self.writes.retarget([self.commander.channel(r) for r in sorted(runners)])
        self.create_commanded(runners - set(self.commander.commanded))

    def follow(self, now: sy.TimeStamp) -> None:
        """Takes in every gh_cmd write since the last step."""
        while (frame := self.writes.read(timeout=DRAIN_TIMEOUT)) is not None:
            for r in self.known:
                series = frame.get(self.commander.channel(r))
                if series is not None and len(series):
                    self.commander.commanded[r] = (now, bool(series[-1]))

    def create_commanded(self, runners: set[str]) -> None:
        """The last gh_state of each runner, so a restart resends nothing."""
        if not runners:
            return
        now = sy.TimeStamp.now()
        found = latest(
            self.client, names=[self.state_channel(r) for r in sorted(runners)]
        )
        for r in runners:
            if (v := found.get(self.state_channel(r))) is not None:
                self.commander.commanded[r] = (now, bool(v))

    def publish(self) -> None:
        current = {
            self.state_channel(r): last[1]
            for r in sorted(self.known)
            if (last := self.commander.commanded.get(r)) is not None
        }
        # Stamped after every blocking call of the step, since the write may open the
        # writer. The Core rejects a sample before the writer's start.
        now = sy.TimeStamp.now()
        frame = on_change(
            GATE_STATE_INDEX, last=self.published, current=current, now=now
        )
        if frame is not None and self.state.write(frame):
            self.published = current

    def close(self) -> None:
        self.writes.close()
        self.state.close()
        self.commander.close()
