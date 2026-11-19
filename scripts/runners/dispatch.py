#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Dispatches each write to a runner's command channel to the matching script.

    <runner>_ec2_cmd    true  ec2_on.py       false  ec2_off.py

Every REFRESH it rereads the runners in registry.json and moves its streamer onto
any command channel created since. Beats runner_commands_heartbeat and logs to
runner_commands_log.

    uv run dispatch.py
"""

import argparse
import os
from collections.abc import Callable, Iterable
from functools import partial
from pathlib import Path
from typing import Protocol

import synnax as sy

import ec2_off
import ec2_on
from aws import AWS
from common import (
    Channels,
    add_connection_args,
    close_quietly,
    command_channels,
    connect,
    parameter_store,
    secret,
    shield,
)
from github import GitHub
from heartbeat import Heartbeat
from log import Log
from registry import STATE_FILE, known_runners

HEARTBEAT = "runner_commands_heartbeat"
LOG = "runner_commands_log"
RETRY_DELAY = 5 * sy.TimeSpan.SECOND
# Short reads keep Ctrl+C responsive: on Windows a blocking receive cannot be
# interrupted, so the signal only lands once the read returns.
READ_TIMEOUT = sy.TimeSpan.SECOND
# How often to look for command channels the controller created since the last look.
REFRESH = 30 * sy.TimeSpan.SECOND

Handler = Callable[[str], str | None]
Handlers = dict[str, dict[bool, Handler]]


class Frame(Protocol):
    def items(self) -> Iterable[tuple[int | str, sy.Series]]: ...


class Streamer(Protocol):
    def read(self, *, timeout: sy.TimeSpan) -> Frame | None: ...

    def close(self) -> object: ...


class Retriever(Protocol):
    def retrieve(self, name: str, /) -> sy.Channel: ...


class Client(Protocol):
    """The part of the Core that dispatch reads."""

    @property
    def channels(self) -> Retriever: ...

    def open_streamer(self, channels: list[str], /) -> Streamer: ...


def handlers(*, aws: AWS, github: GitHub) -> Handlers:
    return {
        "ec2_cmd": {
            True: partial(ec2_on.run, aws=aws),
            False: partial(ec2_off.run, aws=aws, github=github),
        },
    }


def existing_channels(
    client: Client, *, channels: dict[str, tuple[str, str]]
) -> tuple[list[str], dict[int | str, tuple[str, str]]]:
    """Returns the names of the command channels that exist so far, plus a lookup of
    runner and command by channel name and key, since a streamer may label frames by
    either."""
    names: list[str] = []
    lookup: dict[int | str, tuple[str, str]] = {}
    for name, target in channels.items():
        try:
            key = client.channels.retrieve(name).key
        except sy.NotFoundError:
            continue
        names.append(name)
        lookup[name] = target
        lookup[key] = target
    return names, lookup


def dispatch(
    frame: Frame,
    *,
    lookup: dict[int | str, tuple[str, str]],
    commands: Handlers,
    log: Log,
) -> None:
    for channel, series in frame.items():
        if channel not in lookup or len(series) == 0:
            continue
        runner, command = lookup[channel]
        value = bool(series[-1])
        handler = commands[command].get(value)
        if handler is None:
            log(f"{runner}: ignoring {command}={value}")
            continue
        log(f"{runner}: {command}={value}")
        try:
            if (skipped := handler(runner)) is not None:
                log(skipped)
        except Exception as e:
            log.error(f"{runner} {command}={value}", e)


class Listener:
    """Streams the command channels of the runners in registry and dispatches each
    write. Every REFRESH it looks for channels created since and reopens onto them."""

    def __init__(
        self,
        *,
        client: Client,
        commands: Handlers,
        log: Log,
        registry: Path = STATE_FILE,
    ) -> None:
        self.client = client
        self.commands = commands
        self.log = log
        self.registry = registry
        self.names: list[str] = []
        self.lookup: dict[int | str, tuple[str, str]] = {}
        self.refreshed: sy.TimeStamp | None = None
        self.streamer: Streamer | None = None

    def refresh(self, now: sy.TimeStamp) -> None:
        if self.refreshed is not None and now - self.refreshed < REFRESH:
            return
        channels = command_channels(
            known_runners(self.registry), commands=tuple(self.commands)
        )
        names, self.lookup = existing_channels(self.client, channels=channels)
        self.refreshed = now
        if names != self.names:
            self.log(f"listening on {len(names)}/{len(channels)} channels")
            self.names = names
            self.close()

    def step(self, now: sy.TimeStamp) -> None:
        """Reads and dispatches one frame, waiting up to READ_TIMEOUT for it."""
        self.refresh(now)
        if not self.names:
            sy.sleep(READ_TIMEOUT)
            return
        if self.streamer is None:
            self.streamer = self.client.open_streamer(self.names)
        frame = self.streamer.read(timeout=READ_TIMEOUT)
        if frame is not None:
            dispatch(frame, lookup=self.lookup, commands=self.commands, log=self.log)

    def close(self) -> None:
        close_quietly(self.streamer)
        self.streamer = None


def main() -> None:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    add_connection_args(p)
    args = p.parse_args()
    store = parameter_store()
    client = connect(args, store=store)
    creator = Channels(client=client, enabled=args.create_channels)
    log = Log(client=client, channels=creator, channel=LOG)
    github = GitHub(secret("GH_APP_ID", store=store), secret("GH_APP_KEY", store=store))
    commands = handlers(aws=AWS(), github=github)
    alive = Heartbeat(client=client, channels=creator, name=HEARTBEAT)
    listener = Listener(client=client, commands=commands, log=log)
    try:
        while True:
            try:
                listener.step(sy.TimeStamp.now())
            except Exception as e:
                log.error("listener", e)
                listener.close()
                sy.sleep(RETRY_DELAY)
    finally:
        shield()
        listener.close()
        alive.stop()
        log.close()
        client.close()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        os._exit(130)
