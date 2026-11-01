#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The bottom layer every other core module builds on: constants, channel naming and
creation, quiet teardown, and connection arguments."""

import argparse
import os
import re
import signal
from collections.abc import Mapping
from typing import Any, Protocol

import boto3
import numpy as np
import synnax as sy
from pydantic import BaseModel
from synnax import auth
from x.codec import JSONCodec
from x.exceptions.exceptions import REGISTRY, ExceptionPayload, _ExceptionProvider

HEARTBEAT_INTERVAL = 5 * sy.TimeSpan.SECOND
# How long stop() waits for the final false to land.
STOP_TIMEOUT = 2 * sy.TimeSpan.SECOND
ABBREVIATIONS = {"windows": "win", "build_bot": "build", "tester": "test"}

# Command channels that controller.py creates for every runner named after an
# instance, each a boolean on its own <channel>_time index, for dispatch.py to
# act on.
COMMAND_CHANNELS = ("ec2_cmd", "gh_cmd")

# What agent.py samples on each runner: four float32 channels on the runner's own
# index, plus a virtual string channel for the step logs. Kept here because
# controller.py creates them; keep the names in sync with agent.py.
AGENT_INDEX = "sys_time"
AGENT_LOG = "log"
AGENT_CHANNELS = {
    "cpu": sy.DataType.FLOAT32,
    "mem": sy.DataType.FLOAT32,
    "disk_free": sy.DataType.FLOAT32,
    "disk_percent": sy.DataType.FLOAT32,
}

# Values of the <runner>_gh status channels.
MISSING, OFFLINE, IDLE, BUSY = -1, 0, 1, 2

# Measured values cannot be taken over and spoofed.
PUBLISH_AUTHORITY = sy.Authority.ABSOLUTE
# Command writes sit below the Console's absolute 255; writers hold 0 in between.
COMMAND_AUTHORITY = 254
RELEASED_AUTHORITY = 0
# After commanding a runner, wait this long before commanding it again.
RECOMMAND_DELAY = 1 * sy.TimeSpan.MINUTE


def _encode(self: JSONCodec, payload: BaseModel) -> bytes:
    return payload.model_dump_json(by_alias=True, exclude_none=True).encode()


def _decode_token(p: ExceptionPayload) -> Exception | None:
    for cls in (sy.ExpiredToken, sy.InvalidToken):
        if p.type is not None and p.type.startswith(cls.TYPE):
            return cls(p.data)
    return None


def _authenticate(self: auth.Client) -> None:
    # Marked first, so the middleware does not log in again from inside the login.
    self.token = getattr(self, "token", "")
    self.authenticated = True
    try:
        res = auth.send_required(
            self.client,
            "/auth/login",
            auth.Credentials(username=self.username, password=self.password),
            auth.TokenResponse,
        )
    except Exception:
        self.authenticated = False
        raise
    self.token = res.token
    self.user = res.user


def patch_client() -> None:
    """Adapts the Synnax client to the Core the fleet runs against. Cores since
    2026-09-23 reject the null fields the client sends, and the client decodes token
    errors as plain AuthError, so it never logs in again. Call once, before connect."""
    JSONCodec.encode = _encode  # type: ignore[method-assign]
    REGISTRY.providers.insert(0, _ExceptionProvider(lambda e: None, _decode_token))
    auth.Client.authenticate = _authenticate


class Closeable(Protocol):
    def close(self) -> object: ...


def close_quietly(resource: Closeable | None) -> None:
    if resource is None:
        return
    try:
        resource.close()
    except Exception:
        pass


def shield() -> None:
    """Ignores further interrupts, so the shutdown that one Ctrl+C started runs to the
    end. The kill script signals the whole console once per process it finds, so more
    than one lands."""
    signal.signal(signal.SIGINT, signal.SIG_IGN)


def warn(what: str, e: Exception) -> None:
    print(f"[ERROR] {what} failed: {type(e).__name__}: {e}", flush=True)


def channel_name(source: str, *, runner: str) -> str:
    name = re.sub(r"[^A-Za-z0-9]+", "_", runner)
    for long, short in ABBREVIATIONS.items():
        name = name.replace(long, short)
    return f"{name}_{source}"


def instance_prefix(runner: str) -> str | None:
    m = re.search(r"-(i-[0-9a-f]+)$", runner)
    return m.group(1) if m else None


def instance_for(runner: str, *, found: dict[str, dict]) -> str | None:
    prefix = instance_prefix(runner)
    if prefix is None:
        return None
    matches = [iid for iid in found if iid.startswith(prefix)]
    if len(matches) > 1:
        raise LookupError(f"multiple EC2 instances match {runner}")
    return matches[0] if matches else None


def instance_status(runners: set[str], *, found: dict[str, dict]) -> dict[str, bool]:
    status: dict[str, bool] = {}
    for r in runners:
        if instance_prefix(r) is None:
            continue
        iid = instance_for(r, found=found)
        status[r] = iid is not None and found[iid]["running"]
    return status


def command_channels(
    runners: set[str], *, commands: tuple[str, ...] = COMMAND_CHANNELS
) -> dict[str, tuple[str, str]]:
    """Maps each command channel name to its runner and command."""
    return {
        channel_name(command, runner=r): (r, command)
        for r in sorted(runners)
        if instance_prefix(r) is not None
        for command in commands
    }


def gh_channels(runners: set[str]) -> dict[str, sy.DataType]:
    return {channel_name("gh", runner=r): sy.DataType.INT8 for r in sorted(runners)}


def uptime_channels(runners: set[str]) -> dict[str, sy.DataType]:
    return {
        channel_name("state_uptime", runner=r): sy.DataType.INT32
        for r in sorted(runners)
    }


def ec2_channels(runners: set[str]) -> dict[str, sy.DataType]:
    return {
        channel_name("ec2", runner=r): sy.DataType.BOOLEAN
        for r in sorted(runners)
        if instance_prefix(r) is not None
    }


# A request past a few dozen channels outlives the client's timeout.
BATCH = 40


class ChannelCreateError(Exception):
    """A script stops on this rather than retrying every cycle; its launcher spaces
    out the restarts."""


class Channels:
    """Creates channels on the Core. While disabled every method does nothing, for an
    account that cannot create channels."""

    def __init__(self, *, client: sy.Synnax, enabled: bool) -> None:
        self.client = client
        self.enabled = enabled

    def create(
        self, what: str, *channels: sy.Channel, update: bool = False
    ) -> list[sy.Channel]:
        """Creates, or with update rewrites channels that carry their key."""
        try:
            return self.client.channels.create(
                list(channels), retrieve_if_name_exists=not update
            )
        except Exception as e:
            raise ChannelCreateError(f"{what}: {type(e).__name__}: {e}") from e

    def data(
        self, *, index: sy.Channel | None, channels: dict[str, sy.DataType]
    ) -> None:
        if not self.enabled or index is None:
            return
        wanted = [
            sy.Channel(name=name, data_type=data_type, index=index.key)
            for name, data_type in channels.items()
        ]
        for i in range(0, len(wanted), BATCH):
            self.create(f"channels on {index.name}", *wanted[i : i + BATCH])

    def calculated(self, name: str, *, expression: str) -> None:
        """Creates, or when the expression changed rewrites, a calculated channel."""
        if not self.enabled:
            return
        channel = sy.Channel(
            name=name, data_type=sy.DataType.FLOAT32, expression=expression
        )
        found = self.create(f"calculated {name}", channel)[0]
        if found.expression != expression:
            found.expression = expression
            self.create(f"calculated {name}", found, update=True)

    def index(self, name: str) -> sy.Channel | None:
        if not self.enabled:
            return None
        index = sy.Channel(name=name, data_type=sy.DataType.TIMESTAMP, is_index=True)
        return self.create(f"index {name}", index)[0]

    def virtual(self, name: str, *, data_type: sy.DataType) -> None:
        if not self.enabled:
            return
        self.create(
            f"virtual {name}", sy.Channel(name=name, data_type=data_type, virtual=True)
        )

    def agent(self, runners: set[str]) -> None:
        """Creates what agent.py writes, one index per runner. The agents run on
        machines that only get retrieve on channels, so they resolve these rather
        than create them."""
        for r in sorted(runners):
            if instance_prefix(r) is None:
                continue
            index = self.index(channel_name(AGENT_INDEX, runner=r))
            self.data(
                index=index,
                channels={
                    channel_name(s, runner=r): t for s, t in AGENT_CHANNELS.items()
                },
            )
            self.virtual(
                channel_name(AGENT_LOG, runner=r), data_type=sy.DataType.STRING
            )


def latest(client: sy.Synnax, *, names: list[str]) -> dict[str, np.generic]:
    """The last sample of each numeric channel that has one. Misses samples still
    under an open writer."""
    read = client.read_latest(names)
    # A series for one name, a frame for more.
    by_name = {names[0]: read} if isinstance(read, sy.MultiSeries) else read
    found: dict[str, np.generic] = {}
    for n in names:
        series = by_name.get(n)
        if series is None or len(series) == 0:
            continue
        if isinstance(v := series[-1], np.generic):
            found[n] = v
    return found


def seconds(raw: str) -> sy.TimeSpan:
    return sy.TimeSpan.from_seconds(float(raw))


class ParameterStore(Protocol):
    def get_parameter(
        self, *, Name: str, WithDecryption: bool = ...
    ) -> Mapping[str, Any]: ...


def parameter_store() -> ParameterStore:
    """The us-east-1 Parameter Store, from the boto3 credential chain."""
    return boto3.client("ssm", region_name="us-east-1")


def secret(name: str, *, store: ParameterStore) -> str:
    """The environment variable name, else the SecureString parameter /runners/<name>
    in store, lowercased with dashes (GH_TOKEN reads /runners/gh-token). Raises when
    the parameter does not exist."""
    if value := os.environ.get(name):
        return value
    path = "/runners/" + name.lower().replace("_", "-")
    return str(
        store.get_parameter(Name=path, WithDecryption=True)["Parameter"]["Value"]
    )


def add_connection_args(p: argparse.ArgumentParser) -> None:
    p.add_argument("--host", default="localhost")
    p.add_argument("--port", type=int, default=9090)
    p.add_argument("--username", default="synnax")
    p.add_argument("--password", help="defaults to the SYNNAX_PASSWORD secret")
    p.add_argument("--secure", action="store_true", help="connect over TLS")
    p.add_argument(
        "--create-channels",
        action=argparse.BooleanOptionalAction,
        default=True,
        help="create missing channels, such as a new runner's; needs a role that "
        "may create them",
    )


def connect(args: argparse.Namespace, *, store: ParameterStore) -> sy.Synnax:
    return sy.Synnax(
        host=args.host,
        port=args.port,
        username=args.username,
        password=args.password or secret("SYNNAX_PASSWORD", store=store),
        secure=args.secure,
    )
