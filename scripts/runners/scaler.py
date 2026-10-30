#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Pure-demand pool scaling, driven by controller.py's poll cycle. Queued self-hosted
jobs beyond a pool's idle runners start its offline instances; once a pool's queue is
empty, runners idle past the cooldown are stopped. Commands write <runner>_ec2_cmd
for dispatch.py to act on."""

import synnax as sy

from commander import Commander
from common import IDLE, OFFLINE, instance_for
from log import Log

# Writing false to this virtual boolean channel disables scaling, true enables it.
SCALER_CMD = "runner_scaler_cmd"
# Beats the scaler's current state every heartbeat interval;
SCALER_ENABLED = "runner_scaler_enabled"
# Every line the scaler logs, one sample per line.
SCALER_LOG = "runner_scaler_log"

# A pool is the label a job asks for and its runners carry.
Pool = str

POOLS: dict[Pool, str] = {
    "windows-build-bot": "Windows Build",
    "ubuntu-build-bot": "Ubuntu Build",
    "windows-test-bot": "Windows Test",
    "ubuntu-test-bot": "Ubuntu Test",
}
WINDOWS_POOLS = ("windows-build-bot", "windows-test-bot")
# These name a self-hosted pool outright. The rest are shared with GitHub-hosted
# runners, so a job only counts as ours when it also carries "self-hosted".
DEDICATED_LABELS = {
    "windows-build-bot",
    "ubuntu-build-bot",
    "windows-test-bot",
    "ubuntu-test-bot",
}


def pool_of(labels: set[str]) -> Pool | None:
    """The pool a runner belongs to, from the labels it registered with."""
    return next((label for label in sorted(labels) if label in POOLS), None)


def job_pool(labels: set[str]) -> Pool | None:
    """The pool a queued job needs, None when it is not ours to serve."""
    if "self-hosted" not in labels and not labels & DEDICATED_LABELS:
        return None
    return pool_of(labels)


class Scaler:
    """Busy runners, runners with a job in progress, and instances up less than
    boot_grace whose runner is offline are never touched. Instances still booting,
    and runners just commanded to start, count toward a pool's demand. While
    disabled, the writer is closed and nothing is written, but idle time keeps
    accruing and each decision is logged once, marked [DISABLED], so enabling clears
    whatever backlog built up."""

    def __init__(
        self,
        *,
        client: sy.Synnax,
        cooldown: sy.TimeSpan,
        boot_grace: sy.TimeSpan,
        log: Log,
        enabled: bool = True,
        pool_cooldowns: dict[Pool, sy.TimeSpan] | None = None,
    ) -> None:
        self.commander = Commander(
            client=client,
            command="ec2_cmd",
            name="EC2 Scaler",
            log=log,
            describe=self.describe,
        )
        self.log = log
        self.cooldown = cooldown
        self.pool_cooldowns = pool_cooldowns or {}
        self.boot_grace = boot_grace
        self.enabled = enabled
        self.idle_since: dict[str, sy.TimeStamp] = {}
        self.logged: dict[str, bool] = {}
        self.announced: dict[Pool, str] = {}

    def refresh_writer(self, known: set[str]) -> None:
        if self.enabled:
            self.commander.refresh_writer(known)
        else:
            self.commander.commands.close()

    def step(
        self,
        *,
        known: set[str],
        status: dict[str, int],
        labels: dict[str, set[str]],
        queued: dict[Pool, int],
        found: dict[str, dict],
        running: set[str],
        stops: bool = True,
    ) -> None:
        """Starts instances for queued jobs and, when stops is set, stops idle
        runners past their cooldown."""
        self.refresh_writer(known)
        # After the command writer opens, so commands land after its start.
        now = sy.TimeStamp.now()

        def idle_on(r: str) -> bool:
            iid = instance_for(r, found=found)
            # A job in progress outranks a runner status that says offline.
            if iid is None or not found[iid]["running"] or r in running:
                return False
            if status.get(r) == IDLE:
                return True
            return (
                status.get(r) == OFFLINE
                and now.span(found[iid]["launched"]) >= self.boot_grace
            )

        for r in known:
            if idle_on(r):
                self.idle_since.setdefault(r, now)
            else:
                self.idle_since.pop(r, None)
        idle: dict[Pool, list[str]] = {}
        offline: dict[Pool, list[str]] = {}
        # Instances on their way up, which take a queued job once their runner
        # connects.
        booting: dict[Pool, int] = {}
        # Offline-past-grace runners are stoppable but cannot take a job.
        stoppable: dict[Pool, list[str]] = {}
        # Recently commanded runners still count, they just cannot be commanded again.
        ready = self.commander.ready(known, now=now)
        for r in sorted(known):
            pool = pool_of(labels.get(r, set()))
            iid = instance_for(r, found=found)
            if pool is None or iid is None:
                continue
            state = found[iid]["state"]
            last = self.commander.commanded.get(r)
            if state == "stopped" and (r in ready or last is None or not last[1]):
                offline.setdefault(pool, []).append(r)
            elif r in self.idle_since:
                stoppable.setdefault(pool, []).append(r)
                if status.get(r) == IDLE:
                    idle.setdefault(pool, []).append(r)
            elif state in ("stopped", "pending") or (
                state == "running" and status.get(r) == OFFLINE and r not in running
            ):
                booting[pool] = booting.get(pool, 0) + 1
        commands: dict[str, bool] = {}
        for pool, n in queued.items():
            spare = offline.get(pool, [])
            startable = [r for r in spare if r in ready]
            covered = len(idle.get(pool, [])) + booting.get(pool, 0)
            starting = startable[: max(n - covered, 0)]
            # Once a pool has been acted on, stay quiet until something changes for
            # it; only a pool with nothing left to start keeps reporting.
            if starting or not spare:
                self.announce(pool, queued=n, spare=len(spare), starting=len(starting))
            commands.update(dict.fromkeys(starting, True))
        for pool in [p for p in self.announced if p not in queued]:
            del self.announced[pool]
        for pool, runners in stoppable.items():
            if not stops or queued.get(pool, 0):
                continue
            cooldown = self.pool_cooldowns.get(pool, self.cooldown)
            for r in runners:
                if r in ready and now.span(self.idle_since[r]) >= cooldown:
                    commands[r] = False
        if commands:
            self.command(commands, now=now)

    def announce(self, pool: Pool, *, queued: int, spare: int, starting: int) -> None:
        """Logs a pool's demand once per distinct message, so a queue that sits for
        several polls does not repeat itself."""
        action = f"starting {starting}" if starting else "none to start"
        line = (
            f"{POOLS[pool]}: {queued} jobs queued, {spare} instances offline, {action}"
        )
        if self.announced.get(pool) != line:
            self.announced[pool] = line
            self.log(line)

    def describe(self, runner: str, *, value: bool, now: sy.TimeStamp) -> str:
        name = self.commander.channel(runner)
        if value:
            return f"{name}=True"
        since = self.idle_since.get(runner)
        if since is None:
            return f"{name}=False (idle)"
        return f"{name}=False (idle {now.span(since).trunc(sy.TimeSpan.SECOND)})"

    def command(self, commands: dict[str, bool], *, now: sy.TimeStamp) -> None:
        if not self.enabled:
            # Log each decision once, and leave commanded alone so enabling acts
            # on the backlog without waiting out RECOMMAND_DELAY.
            fresh = {r: v for r, v in commands.items() if self.logged.get(r) != v}
            if fresh:
                self.logged.update(fresh)
                self.log(f"[DISABLED] {self.commander.lines(fresh, now=now)}")
            return
        self.logged.clear()
        self.commander.command(commands, now=now)

    def close(self) -> None:
        self.commander.close()
