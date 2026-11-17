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
empty, runners idle past the cooldown are drained, unregistered from GitHub, then
stopped. A drain writes false to <runner>_gh_cmd for agent.py to act on, and a start or
a stop writes <runner>_ec2_cmd for dispatch.py to act on."""

import synnax as sy

from commander import Commander
from common import IDLE, OFFLINE, Writes, instance_for
from gate import Gate
from github import GitHub
from log import Log
from pools import POOLS, Pool, pool_of

# Writing false to this virtual boolean channel disables scaling, true enables it.
SCALER_CMD = "runner_scaler_cmd"
# Beats the scaler's current state every heartbeat interval;
SCALER_ENABLED = "runner_scaler_enabled"
# Every line the scaler logs, one sample per line.
SCALER_LOG = "runner_scaler_log"


class Scaler:
    """Busy runners, runners with a job in progress, and instances up less than
    boot_grace whose runner is offline and not drained are never touched. Instances
    still booting, and runners just commanded to start, count toward a pool's demand. An
    instance stops only after its runner is drained and offline: the agent holds a drain
    while a job runs, so a runner that takes a job after the decision keeps its
    instance. The runner's GitHub registration is deleted before the stop, so GitHub
    cannot assign a job to a runner that is going down; a runner GitHub still has a job
    for keeps its instance and is enabled again to take the job. The caller reports a
    runner GitHub does not list as offline, and its start registers it again. A drain is
    sent again each RECOMMAND_DELAY while the runner stays online. Once the instance is
    stopped, the runner is enabled again for its next start, unless the gate is closed.
    While disabled, the writer is closed and no drain, start, stop, or delete is sent,
    but idle time keeps accruing and each decision is logged once, marked [DISABLED], so
    enabling clears whatever backlog built up. A runner this scaler drained is enabled
    again while disabled, so no drain outlives the switch.

    :param gate: Sends the runner commands, and holds the last one of each runner.
    :param github: Deletes runner registrations.
    """

    def __init__(
        self,
        *,
        client: Writes,
        cooldown: sy.TimeSpan,
        boot_grace: sy.TimeSpan,
        log: Log,
        gate: Gate,
        github: GitHub,
        enabled: bool = True,
    ) -> None:
        self.commander = Commander(
            client=client,
            command="ec2_cmd",
            name="EC2 Scaler",
            log=log,
            describe=self.describe,
        )
        self.log = log
        self.gate = gate
        self.github = github
        self.cooldown = cooldown
        self.boot_grace = boot_grace
        self.enabled = enabled
        self.idle_since: dict[str, sy.TimeStamp] = {}
        # The runners this scaler drained that are still drained.
        self.drains: set[str] = set()
        # The decisions logged while disabled, by command channel.
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
    ) -> set[str]:
        """Starts instances for queued jobs and, when stops is set, drains idle
        runners past their cooldown and stops the instances of the drained ones.
        Returns the runners on their way up."""
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
            # A drained runner does not come online, so it gets no time to boot.
            return status.get(r) == OFFLINE and (
                self.drained(r) or now.span(found[iid]["launched"]) >= self.boot_grace
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
        booting: dict[Pool, list[str]] = {}
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
                # A drained runner is on its way down, so it covers no queued job.
                if status.get(r) == IDLE and not self.drained(r):
                    idle.setdefault(pool, []).append(r)
            elif state in ("stopped", "pending") or (
                state == "running" and status.get(r) == OFFLINE and r not in running
            ):
                booting.setdefault(pool, []).append(r)
        commands: dict[str, bool] = {}
        for pool, n in queued.items():
            if not POOLS[pool].scaled:
                continue
            spare = offline.get(pool, [])
            startable = [r for r in spare if r in ready]
            covered = len(idle.get(pool, [])) + len(booting.get(pool, []))
            starting = startable[: max(n - covered, 0)]
            # Once a pool has been acted on, stay quiet until something changes for
            # it; only a pool with nothing left to start keeps reporting.
            if starting or not spare:
                self.announce(pool, queued=n, spare=len(spare), starting=len(starting))
            commands.update(dict.fromkeys(starting, True))
        for pool in [p for p in self.announced if p not in queued]:
            del self.announced[pool]
        runners_ready = self.gate.commander.ready(known, now=now)
        drains: dict[str, bool] = {}
        for pool, runners in stoppable.items():
            if not POOLS[pool].scaled or not stops:
                continue
            for r in runners:
                if now.span(self.idle_since[r]) < self.cooldown:
                    continue
                down = status.get(r) == OFFLINE
                if down and self.drained(r):
                    if r in ready:
                        commands[r] = False
                # An online runner stays up to take a job of its pool's queue.
                elif (down or not queued.get(pool, 0)) and (
                    not self.drained(r) or r in runners_ready
                ):
                    drains[r] = False
        # Runners GitHub refused to delete because it bound a job to them.
        bound: set[str] = set()
        if self.enabled:
            for r in [r for r, start in commands.items() if not start]:
                if not self.github.delete(r):
                    del commands[r]
                    bound.add(r)
                    # A fresh cooldown gives the runner time to take the job.
                    self.idle_since.pop(r, None)
                    self.log(f"{r} has a job, starting its runner to take it")
        self.drains = {r for r in self.drains if self.drained(r)}
        enables: dict[str, bool] = {}
        for r in known if self.gate.enabled else ():
            iid = instance_for(r, found=found)
            if iid is None or not self.drained(r):
                continue
            state = found[iid]["state"]
            if self.enabled:
                enabling = state == "stopped"
            else:
                # A running instance whose stop was just commanded can still go down,
                # and a runner enabled on it could take a job.
                enabling = r in self.drains and (
                    state == "stopped" or (state == "running" and r in ready)
                )
            if enabling or r in bound:
                enables[r] = True
        if enables:
            self.gate.commander.command(enables, now=now)
        if drains:
            self.command(drains, commander=self.gate.commander, now=now)
            self.drains |= {r for r in drains if self.drained(r)}
        if commands:
            self.command(commands, commander=self.commander, now=now)
        return {r for runners in booting.values() for r in runners}

    def drained(self, runner: str) -> bool:
        last = self.gate.commander.commanded.get(runner)
        return last is not None and not last[1]

    def announce(self, pool: Pool, *, queued: int, spare: int, starting: int) -> None:
        """Logs a pool's demand once per distinct message, so a queue that sits for
        several polls does not repeat itself."""
        action = f"starting {starting}" if starting else "none to start"
        title = POOLS[pool].title
        line = f"{title}: {queued} jobs queued, {spare} instances offline, {action}"
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

    def command(
        self, commands: dict[str, bool], *, commander: Commander, now: sy.TimeStamp
    ) -> None:
        if not self.enabled:
            # Log each decision once, and leave commanded alone so enabling acts
            # on the backlog without waiting out RECOMMAND_DELAY.
            fresh = {
                r: v
                for r, v in commands.items()
                if self.logged.get(commander.channel(r)) != v
            }
            if fresh:
                self.logged.update({commander.channel(r): v for r, v in fresh.items()})
                self.log(f"[DISABLED] {commander.lines(fresh, now=now)}")
            return
        self.logged.clear()
        commander.command(commands, now=now)

    def close(self) -> None:
        self.commander.close()
