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

"""Dispatches each write to a runner's command channel to the matching script.

    <runner>_ec2_cmd    true  ec2_on.py       false  ec2_off.py

Runners come from registry.json, read once at startup, so restart after a new
runner appears. Beats runner_commands_heartbeat and logs to runner_commands_log.

    uv run dispatch.py
"""

import argparse
import os
from collections.abc import Callable
from functools import partial

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
    patch_client,
    secret,
    shield,
)
from github import GitHub
from heartbeat import Heartbeat
from log import Log
from registry import known_runners

HEARTBEAT = "runner_commands_heartbeat"
LOG = "runner_commands_log"
RETRY_DELAY = 5 * sy.TimeSpan.SECOND
# Short reads keep Ctrl+C responsive: on Windows a blocking receive cannot be
# interrupted, so the signal only lands once the read returns.
READ_TIMEOUT = sy.TimeSpan.SECOND

Handler = Callable[[str], str]
Handlers = dict[str, dict[bool, Handler]]


def handlers(*, aws: AWS, github: GitHub) -> Handlers:
    return {
        "ec2_cmd": {
            True: partial(ec2_on.run, aws=aws),
            False: partial(ec2_off.run, aws=aws, github=github),
        },
    }


def existing_channels(
    client: sy.Synnax, *, channels: dict[str, tuple[str, str]]
) -> tuple[list[str], dict[int | str, tuple[str, str]]]:
    """Returns the names of the command channels that exist so far, plus a lookup of
    runner and command by channel name and key, since a streamer may label frames by
    either. Channels the poller has not created yet need a restart to be picked up."""
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
    frame: sy.Frame,
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
            log(handler(runner))
        except Exception as e:
            log.error(f"{runner} {command}={value}", e)


def main() -> None:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    add_connection_args(p)
    args = p.parse_args()
    patch_client()
    store = parameter_store()
    client = connect(args, store=store)
    creator = Channels(client=client, enabled=args.create_channels)
    log = Log(client=client, channels=creator, channel=LOG)
    commands = handlers(aws=AWS(), github=GitHub(secret("GH_TOKEN", store=store)))
    alive = Heartbeat(client=client, channels=creator, name=HEARTBEAT)
    channels = command_channels(known_runners(), commands=tuple(commands))
    names, lookup = existing_channels(client, channels=channels)
    if not names:
        raise SystemExit("no command channels exist yet; run controller.py first")
    log(f"listening on {len(names)}/{len(channels)} channels")
    streamer = None
    try:
        while True:
            try:
                if streamer is None:
                    streamer = client.open_streamer(names)
                frame = streamer.read(timeout=READ_TIMEOUT)
                if frame is not None:
                    dispatch(frame, lookup=lookup, commands=commands, log=log)
            except Exception as e:
                log.error("listener", e)
                close_quietly(streamer)
                streamer = None
                sy.sleep(RETRY_DELAY)
    finally:
        shield()
        close_quietly(streamer)
        alive.stop()
        log.close()
        client.close()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        os._exit(130)
