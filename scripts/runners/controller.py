#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# /// script
# requires-python = ">=3.12,<4"
# dependencies = ["boto3", "certifi>=2026.10.19", "synnax>=0.59,<0.60"]
# ///

"""Polls GitHub and EC2 every --interval and writes the fleet's state to Synnax.
Also runs the scaler, the gate, and the cost fetch, and creates every channel below
plus the agents' channels, for each new runner too, unless --no-create-channels.

Per-runner channels, written on change:

    <runner>_gh            int8 on gh_time:
                               -1  channel exists but GitHub no longer lists the runner
                                0  offline
                                1  online and idle
                                2  online and busy
    <runner>_ec2           bool on ec2_time: instance running
    <runner>_state_uptime  int32 on state_uptime_time: seconds idle or busy, else
                           since the instance launched
    <runner>_events        string on its own index, written only on job and step
                           transitions
    <runner>_ec2_cmd       bool on its own index, for dispatch.py
    <runner>_gh_cmd        same
    <runner>_gh_state      bool on gh_state_time: the last gh_cmd sent

Cost channels (cost.py), float32 on the cost_query_time index. The totals come
from a fetch every 6 h; every channel is written each minute, with live runner rates:

    cost_<window>                     the account, <window> one of 1d, 7d, mtd, 30d,
                                      90d, lm (last month), ytd, 365d
    cost_<region>_<window>            per region: use1, use2, global
    cost_<region>_<service>_<window>  per service: ec2, ec2_other, s3, ssm,
                                      cloudwatch, other
    cost_<type>_<window>              per instance type seen in the account
    <any of the above>_net            the same with credits applied
    hours_<region>_<window>           EC2 instance hours
    cost_month_remaining_forecast     AWS's forecast, today to month end
    month_hours_remaining             hours from the write to month end
    cost_fetched_at                   timestamp of the last fetch
    <runner>_cost_per_hour            list price plus volumes, 0 while stopped, live
    <runner>_ebs_cost_per_hour        the volume share
    <runner>_cost_1d_est              price times hours ran yesterday
    <runner>_cost_mtd_est             the same for the month through yesterday

Sums, errors, and month projections are calculated channels, see cost.calculated().

Job starts and finishes arrive as workflow_job webhooks on --webhook-port (9091),
checked against GH_WEBHOOK_SECRET; steps are polled from five running runs per
cycle, and the queue is rebuilt from the API every 5 minutes. Runner names persist
in registry.json so a removed runner keeps reporting -1.

Scaler (scaler.py): starts stopped runners when jobs queue in their pool, stops
runners idle past --idle-cooldown (1 min), or --windows-idle-cooldown (15 min) for the
Windows pools, whose cold start costs each merge group minutes. An offline runner on
an instance up longer than --boot-grace (2 min) counts as idle. --scale sets the
start state (enabled); runner_scaler_cmd switches it at runtime,
runner_scaler_enabled beats the state, runner_scaler_log carries its log.

Gate (gate.py): fans global_gh_cmd out to every <runner>_gh_cmd, disconnecting the
fleet from GitHub without stopping instances. --gh sets the start state (enabled);
runner_gh_global_set and runner_gh_log as above.

Both write commands at authority 254 for the instant of a write, so the Console
(255) always wins.

Own channels: runner_controller_heartbeat (true every 5 s, false on clean exit),
runner_controller_log (every line logged here).

Reads the GH_TOKEN, GH_WEBHOOK_SECRET, and SYNNAX_PASSWORD secrets (see common.secret).
Needs AWS credentials from the boto3 chain that may DescribeInstances, DescribeVolumes,
cloudwatch:GetMetricData, ce:GetCostAndUsage, ce:GetCostForecast, pricing:GetProducts,
and ssm:GetParameter on /runners/*.
"""

import argparse
import os
from concurrent.futures import Future, ThreadPoolExecutor
from concurrent.futures import wait as futures_wait

import synnax as sy

from aws import AWS
from common import (
    BUSY,
    IDLE,
    MISSING,
    OFFLINE,
    ChannelCreateError,
    Channels,
    add_connection_args,
    channel_name,
    command_channels,
    connect,
    ec2_channels,
    gh_channels,
    instance_status,
    parameter_store,
    patch_client,
    seconds,
    secret,
    shield,
)
from cost import QUERY_TIME, Cost
from events import Events
from gate import (
    DRAIN_TIMEOUT,
    GATE_CMD,
    GATE_ENABLED,
    GATE_ENABLED_INDEX,
    GATE_LOG,
    Gate,
)
from github import GitHub
from heartbeat import Heartbeat
from jobs import Jobs
from log import Log
from publisher import Publisher, on_change
from registry import RunnerRegistry
from scaler import SCALER_CMD, SCALER_ENABLED, SCALER_LOG, WINDOWS_POOLS, Scaler
from subscription import Subscription
from uptime import Uptime
from webhook import Webhook

GH_INDEX = "gh_time"
EC2_INDEX = "ec2_time"
HEARTBEAT = "runner_controller_heartbeat"
LOG = "runner_controller_log"
WEBHOOK_PORT = 9091

# A runtime switch: what a command channel enables, its heartbeat, its log name.
Toggle = tuple[Scaler | Gate, Heartbeat, str]


def resolve(futures: list[Future]) -> list:
    """Waits for the futures in short slices so Ctrl+C can land on Windows, where an
    unbounded wait defers the interrupt until the work finishes."""
    while not all(f.done() for f in futures):
        futures_wait(futures, timeout=0.25)
    return [f.result() for f in futures]


class Controller:
    """One poll cycle: fetch GitHub and EC2 concurrently, refresh channels when the
    runner set changes, publish status and events, and let the gate and the scaler
    act. Owns the status writer, the events writer, and the switch subscription;
    after reset() the next cycle reopens them."""

    def __init__(
        self,
        *,
        client: sy.Synnax,
        channels: Channels,
        github: GitHub,
        aws: AWS,
        runners: RunnerRegistry,
        jobs: Jobs,
        cost: Cost,
        scaler: Scaler,
        gate: Gate,
        log: Log,
        webhook: Webhook,
    ) -> None:
        self.channels = channels
        self.github = github
        self.aws = aws
        self.runners = runners
        self.jobs = jobs
        self.cost = cost
        self.scaler = scaler
        self.gate = gate
        self.log = log
        self.webhook = webhook
        self.pool = ThreadPoolExecutor(8)
        self.gh_index = channels.index(GH_INDEX)
        self.ec2_index = channels.index(EC2_INDEX)
        self.known: set[str] = set()
        self.cost_index = channels.index(QUERY_TIME)
        # What the writer carries, so a fetch that adds a type retargets first.
        self.cost_channels: set[str] = set()
        self.uptime = Uptime(client=client, channels=channels, report=log.error)
        self.status = Publisher(
            client=client, name="Controller", channels=[], report=log.error
        )
        self.last_gh: dict[str, int] = {}
        self.last_ec2: dict[str, bool] = {}
        self.cost_pending: dict[str, float] = {}
        self.events = Events(
            client=client, channels=channels, runners=set(), report=log.error
        )
        self.commands = Subscription(
            client=client,
            name="switches",
            channels=[SCALER_CMD, GATE_CMD],
            report=log.error,
        )

    def cycle(self, timer: sy.Timer) -> None:
        (status, labels), found = self.fetch()
        self.runners.add(set(status))
        cost = self.cost.step(
            sy.TimeStamp.now(), known=self.runners.snapshot(), found=found
        )
        if (
            self.runners.snapshot() != self.known
            or not cost.keys() <= self.cost_channels
        ):
            self.refresh_channels(found)
        events = self.jobs.apply(self.webhook.drain())
        self.jobs.resync(sy.TimeStamp.now())
        for runner, lines in self.jobs.poll().items():
            events.setdefault(runner, []).extend(lines)
        events = {r: lines for r, lines in events.items() if r in self.known}
        queued = self.jobs.counts()
        # Stamped once every call that can block is done, so the samples cannot land
        # before the writer this frame opens.
        now = sy.TimeStamp.now()
        states = {r: status.get(r, MISSING) for r in self.known}
        values = {channel_name("gh", runner=r): v for r, v in states.items()}
        counts = {v: list(states.values()).count(v) for v in states.values()}
        ec2_values = {
            channel_name("ec2", runner=r): v
            for r, v in instance_status(self.known, found=found).items()
        }
        self.uptime.update(states, found=found, now=now)
        frame = on_change(GH_INDEX, last=self.last_gh, current=values, now=now) or {}
        frame |= (
            on_change(EC2_INDEX, last=self.last_ec2, current=ec2_values, now=now) or {}
        )
        if cost:
            self.cost_pending = cost
        if self.cost_pending:
            frame |= {QUERY_TIME: now, **self.cost_pending}
        if frame and self.status.write(frame):
            self.last_gh, self.last_ec2, self.cost_pending = values, ec2_values, {}
        self.events.write(events)
        self.gate.step(self.known)
        # A closed gate would disconnect whatever the scaler started.
        self.scaler.step(
            known=self.known,
            status=status,
            labels=labels,
            queued=queued if self.gate.enabled else {},
            found=found,
            running=set(self.jobs.active),
            # An unread queue looks empty, which would stop runners jobs wait for.
            stops=self.jobs.ready,
        )
        self.log(
            f"busy={counts.get(BUSY, 0)} "
            f"idle={counts.get(IDLE, 0)} offline={counts.get(OFFLINE, 0)} "
            f"missing={counts.get(MISSING, 0)} "
            f"ec2_running={sum(ec2_values.values())} "
            f"queued={sum(queued.values())} "
            f"events={sum(len(v) for v in events.values())} "
            f"hooks={self.webhook.received} "
            f"poll={timer.elapsed().seconds:.2f}s"
        )

    def fetch(self) -> list:
        """GitHub runner status and EC2 instances, queried concurrently."""
        return resolve(
            [
                self.pool.submit(self.github.runner_status),
                self.pool.submit(self.aws.instances),
            ]
        )

    def refresh_channels(self, found: dict[str, dict]) -> None:
        """Creates the per-runner channels and moves the writers onto the current
        runner set."""
        self.known = self.runners.snapshot()
        cost = self.cost.channels()
        self.cost_channels = set(cost)
        self.channels.data(index=self.gh_index, channels=gh_channels(self.known))
        self.channels.data(index=self.cost_index, channels=cost)
        for name, expression in self.cost.calculated(found).items():
            self.channels.calculated(name, expression=expression)
        self.uptime.retarget(self.known)
        self.channels.data(index=self.ec2_index, channels=ec2_channels(self.known))
        for name in command_channels(self.known):
            index = self.channels.index(f"{name}_time")
            self.channels.data(index=index, channels={name: sy.DataType.BOOLEAN})
        self.channels.agent(self.known)
        self.status.retarget(
            [
                GH_INDEX,
                *gh_channels(self.known),
                EC2_INDEX,
                *ec2_channels(self.known),
                QUERY_TIME,
                *cost,
            ]
        )
        self.events.retarget(self.known)

    def reset(self) -> None:
        """Closes the cycle's resources after a failure; the next cycle reopens
        them."""
        for r in (self.status, self.events, self.commands):
            r.close()

    def close(self) -> None:
        self.pool.shutdown(wait=False)
        self.cost.close()
        self.uptime.stop()
        self.scaler.close()
        self.gate.close()
        self.webhook.close()
        self.reset()


def wait_for_cmds(
    commands: Subscription, *, remaining: sy.TimeSpan, toggles: dict[str, Toggle]
) -> None:
    """Sleeps out the rest of the poll interval reading the switch subscription,
    applying toggle writes as they arrive. Drains pending writes even when no time
    remains, so slow polls cannot lock out the switches. Short reads keep Ctrl+C
    responsive on Windows."""
    timer = sy.Timer()
    while True:
        left = remaining - timer.elapsed()
        frame = commands.read(timeout=max(min(left, sy.TimeSpan.SECOND), DRAIN_TIMEOUT))
        if frame is None:
            if left <= DRAIN_TIMEOUT:
                return
            continue
        for channel, (target, enabled, name) in toggles.items():
            series = frame.get(channel)
            if series is None or len(series) == 0:
                continue
            state = bool(series[-1])
            if state != target.enabled:
                target.enabled = state
                target.log(f"{name} enabled" if state else f"{name} disabled")
                enabled.beat()


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    add_connection_args(p)
    p.add_argument(
        "--interval",
        type=seconds,
        default=5 * sy.TimeSpan.SECOND,
        help="seconds between polls",
    )
    p.add_argument(
        "--scale",
        choices=("disabled", "enabled"),
        default="enabled",
        help="scaler start state; while disabled, decisions print without writing",
    )
    p.add_argument(
        "--idle-cooldown",
        type=seconds,
        default=1 * sy.TimeSpan.MINUTE,
        help="idle seconds before the scaler stops a runner",
    )
    p.add_argument(
        "--windows-idle-cooldown",
        type=seconds,
        default=15 * sy.TimeSpan.MINUTE,
        help="idle seconds before the scaler stops a Windows build or test runner",
    )
    p.add_argument(
        "--boot-grace",
        type=seconds,
        default=2 * sy.TimeSpan.MINUTE,
        help="seconds after launch before an offline runner counts as idle",
    )
    p.add_argument(
        "--gh",
        choices=("disabled", "enabled"),
        default="enabled",
        help="gate start state; while disabled, runners are disconnected from GitHub",
    )
    p.add_argument(
        "--webhook-port",
        type=int,
        default=WEBHOOK_PORT,
        help="port the workflow_job listener serves on",
    )
    return p.parse_args()


def main() -> None:
    args = parse_args()
    store = parameter_store()
    github = GitHub(secret("GH_TOKEN", store=store))
    patch_client()
    client = connect(args, store=store)
    channels = Channels(client=client, enabled=args.create_channels)
    log = Log(client=client, channels=channels, channel=LOG)
    alive = Heartbeat(client=client, channels=channels, name=HEARTBEAT)
    channels.virtual(SCALER_CMD, data_type=sy.DataType.BOOLEAN)
    channels.data(
        index=channels.index(f"{GATE_CMD}_time"),
        channels={GATE_CMD: sy.DataType.BOOLEAN},
    )
    scaler = Scaler(
        client=client,
        cooldown=args.idle_cooldown,
        boot_grace=args.boot_grace,
        log=Log(client=client, channels=channels, channel=SCALER_LOG),
        enabled=args.scale == "enabled",
        pool_cooldowns=dict.fromkeys(WINDOWS_POOLS, args.windows_idle_cooldown),
    )
    gate = Gate(
        client=client,
        channels=channels,
        log=Log(client=client, channels=channels, channel=GATE_LOG),
        enabled=args.gh == "enabled",
    )
    toggles: dict[str, Toggle] = {
        SCALER_CMD: (
            scaler,
            Heartbeat(
                client=client,
                channels=channels,
                name=SCALER_ENABLED,
                value=lambda: scaler.enabled,
            ),
            "scaler",
        ),
        GATE_CMD: (
            gate,
            Heartbeat(
                client=client,
                channels=channels,
                name=GATE_ENABLED,
                value=lambda: gate.enabled,
                index=GATE_ENABLED_INDEX,
            ),
            "gate",
        ),
    }
    # Required, so a misconfigured deploy fails at startup instead of serving a
    # listener that accepts unsigned posts.
    webhook = Webhook(
        secret=secret("GH_WEBHOOK_SECRET", store=store),
        port=args.webhook_port,
        report=log.error,
    )
    aws = AWS()
    runners = RunnerRegistry()
    runners.add(set(github.runner_status()[0]))
    cost = Cost(log=log, aws=aws)
    cost.create_state(client, known=runners.snapshot())
    resync_pool = ThreadPoolExecutor(1)
    controller = Controller(
        client=client,
        channels=channels,
        github=github,
        aws=aws,
        runners=runners,
        jobs=Jobs(github=github, log=log, pool=resync_pool),
        cost=cost,
        scaler=scaler,
        gate=gate,
        log=log,
        webhook=webhook,
    )
    try:
        while True:
            timer = sy.Timer()
            try:
                controller.cycle(timer)
            except ChannelCreateError:
                raise
            except Exception as e:
                log.error("poll", e)
                controller.reset()
            wait_for_cmds(
                controller.commands,
                remaining=args.interval - timer.elapsed(),
                toggles=toggles,
            )
    finally:
        shield()
        resync_pool.shutdown(wait=False)
        controller.close()
        for _, enabled, _ in toggles.values():
            enabled.stop()
        alive.stop()
        log.close()
        client.close()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        # Skips joining pool threads stuck in network waits; the heartbeat 0 and the
        # writer closes already ran in main's finally.
        os._exit(130)
