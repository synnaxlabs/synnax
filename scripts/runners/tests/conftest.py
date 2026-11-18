#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import datetime as dt
import urllib.error
import urllib.request
from datetime import UTC, datetime
from email.message import Message
from functools import cache
from typing import Literal

import numpy as np
import pytest
import synnax as sy
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from mypy_boto3_secretsmanager.type_defs import (
    FilterTypeDef,
    SecretListEntryTypeDef,
    TagTypeDef,
)

from github import API, REPO, GitHub
from logins import Listed, Secret


class Writer:
    def __init__(self, *, client: "Client", start: sy.TimeStamp) -> None:
        self.client = client
        self.start = start

    def write(self, frame: dict) -> None:
        if self.client.failing:
            raise ConnectionError("Core unreachable")
        for value in frame.values():
            for sample in value if isinstance(value, list) else [value]:
                if isinstance(sample, sy.TimeStamp) and sample < self.start:
                    raise ValueError(f"sample {sample} is before start {self.start}")
        self.client.frames.append(frame)

    def set_authority(self, authority: int) -> None:
        pass

    def close(self) -> None:
        pass


class Streamer:
    def __init__(self, client: "Client") -> None:
        self.client = client

    def read(self, *, timeout: sy.TimeSpan) -> dict | None:
        return self.client.streamed.pop(0) if self.client.streamed else None

    def close(self) -> None:
        pass


class ChannelStore:
    """Stands in for the Core's channels: records every channel created, returns the
    stored channel for a name that exists, and refuses a create of a stored key."""

    def __init__(self) -> None:
        self.created: list[sy.Channel] = []
        self.stored: dict[str, sy.Channel] = {}

    def create(
        self, channels: list[sy.Channel], /, *, retrieve_if_name_exists: bool
    ) -> list[sy.Channel]:
        keys = {c.key for c in self.stored.values()}
        if any(c.key in keys for c in channels):
            raise sy.ValidationError("channel already exists")
        self.created += channels
        out = []
        for c in channels:
            if c.name not in self.stored or not retrieve_if_name_exists:
                c.key = len(self.stored) + 1
                self.stored[c.name] = c
            out.append(self.stored[c.name])
        return out

    def set_expression(self, key: int, expression: str) -> None:
        next(c for c in self.stored.values() if c.key == key).expression = expression


class Client:
    """Stands in for the Core: records every frame written, rejects a timestamp before
    the writer's start, fails writes while failing is set, moves the clock latency
    ahead on each streamer open, streams the frames in streamed, and reads samples
    from latest."""

    def __init__(self) -> None:
        self.frames: list[dict] = []
        self.failing = False
        self.latency = sy.TimeSpan(0)
        self.elapsed = sy.TimeSpan(0)
        self.streamed: list[dict] = []
        self.latest: dict[str, np.ndarray] = {}
        self.channels = ChannelStore()

    def open_writer(
        self, start: sy.TimeStamp, *args: object, **kwargs: object
    ) -> Writer:
        return Writer(client=self, start=start)

    def open_streamer(self, *args: object, **kwargs: object) -> Streamer:
        self.elapsed += self.latency
        return Streamer(self)

    def read_latest(self, names: list[str]) -> dict:
        return {n: self.latest[n] for n in names if n in self.latest}

    def written(self, channel: str) -> list[object]:
        return [f[channel] for f in self.frames if channel in f]


@pytest.fixture
def client(monkeypatch: pytest.MonkeyPatch) -> Client:
    c = Client()
    now = sy.TimeStamp.now
    monkeypatch.setattr(
        sy.TimeStamp, "now", classmethod(lambda _: sy.TimeStamp(now() + c.elapsed))
    )
    return c


APP_ID = "5687191"
INSTALLATION = 180309468
NOW = datetime(2026, 11, 10, 12, tzinfo=UTC).timestamp()
HOUR = 60 * 60


@cache
def key() -> rsa.RSAPrivateKey:
    return rsa.generate_private_key(public_exponent=65537, key_size=2048)


def pem() -> str:
    return (
        key()
        .private_bytes(
            serialization.Encoding.PEM,
            serialization.PrivateFormat.PKCS8,
            serialization.NoEncryption(),
        )
        .decode()
    )


class Clock:
    def __init__(self) -> None:
        self.now = NOW

    def __call__(self) -> float:
        return self.now


class Server:
    """Answers GitHub requests from memory, minting token-1, token-2, and so on, each
    valid for an hour from the clock. Fails the next request to a URL in failures with
    its status code. Lists the runners in registered as online, looks them up by name,
    deletes them, and refuses with a 422 to delete one in busy."""

    def __init__(self, clock: Clock) -> None:
        self.clock = clock
        self.requests: list[urllib.request.Request] = []
        self.minted = 0
        self.installation = INSTALLATION
        self.failures: dict[str, int] = {}
        self.registered: dict[str, int] = {}
        self.busy: set[str] = set()

    def __call__(self, req: urllib.request.Request) -> dict:
        self.requests.append(req)
        url = req.full_url
        if code := self.failures.pop(url, None):
            raise urllib.error.HTTPError(url, code, "failed", Message(), None)
        if url == f"{API}/repos/{REPO}/installation":
            return {"id": self.installation}
        if url == f"{API}/app/installations/{self.installation}/access_tokens":
            assert req.get_method() == "POST"
            self.minted += 1
            expires = datetime.fromtimestamp(self.clock.now + HOUR, tz=UTC)
            return {
                "token": f"token-{self.minted}",
                "expires_at": expires.strftime("%Y-%m-%dT%H:%M:%SZ"),
            }
        runners = f"{API}/repos/{REPO}/actions/runners"
        if url.startswith(f"{runners}?per_page="):
            return {
                "runners": [
                    {
                        "id": i,
                        "name": n,
                        "status": "online",
                        "busy": False,
                        "labels": [],
                    }
                    for n, i in self.registered.items()
                ]
            }
        if url.startswith(f"{runners}?name="):
            name = url.removeprefix(f"{runners}?name=")
            if name not in self.registered:
                return {"runners": []}
            return {"runners": [{"id": self.registered[name], "name": name}]}
        if req.get_method() == "DELETE":
            ids = {v: k for k, v in self.registered.items()}
            name = ids[int(url.removeprefix(f"{runners}/"))]
            if name in self.busy:
                raise urllib.error.HTTPError(url, 422, "busy", Message(), None)
            del self.registered[name]
            return {}
        return {"url": url}

    def auth(self, i: int) -> str:
        return str(self.requests[i].get_header("Authorization"))

    def deletes(self) -> list[str]:
        return [r.full_url for r in self.requests if r.get_method() == "DELETE"]


def github(server: Server) -> GitHub:
    return GitHub(APP_ID, pem(), send=server, clock=server.clock)


class SecretsExceptions:
    class ResourceNotFoundException(Exception):
        pass


class Deleted(Secret):
    DeletedDate: dt.datetime


class Secrets:
    """Stands in for Secrets Manager: the tags of each secret by name. Writes fail
    while failing is set. A secret in deleted is scheduled for deletion, and one in
    gone is listed but no longer exists."""

    exceptions = SecretsExceptions()

    def __init__(self) -> None:
        self.tags: dict[str, dict[str, str]] = {}
        self.deleted: set[str] = set()
        self.gone: set[str] = set()
        self.failing = False

    def get_paginator(self, operation: Literal["list_secrets"]) -> "Secrets":
        return self

    def paginate(self, *, Filters: list[FilterTypeDef]) -> list[Listed]:
        prefix = Filters[0]["Values"][0]
        # The list carries no tags, so a read that trusts it sees none.
        listed: list[SecretListEntryTypeDef] = [
            {"Name": name, "ARN": f"arn:{name}", "Tags": []}
            for name in self.tags
            if name.startswith(prefix)
        ]
        return [{"SecretList": listed}]

    def describe_secret(self, *, SecretId: str) -> Secret:
        name = SecretId.removeprefix("arn:")
        if name in self.gone:
            raise self.exceptions.ResourceNotFoundException(name)
        tags: list[TagTypeDef] = [
            {"Key": k, "Value": v} for k, v in self.tags[name].items()
        ]
        if name in self.deleted:
            return Deleted(
                Name=name,
                ARN=SecretId,
                Tags=tags,
                DeletedDate=dt.datetime(2026, 11, 15, tzinfo=dt.UTC),
            )
        return {"Name": name, "ARN": SecretId, "Tags": tags}

    def tag_resource(self, *, SecretId: str, Tags: list[TagTypeDef]) -> None:
        if self.failing:
            raise PermissionError("not authorized")
        self.tags[SecretId.removeprefix("arn:")] |= {t["Key"]: t["Value"] for t in Tags}

    def untag_resource(self, *, SecretId: str, TagKeys: list[str]) -> None:
        if self.failing:
            raise PermissionError("not authorized")
        for key in TagKeys:
            self.tags[SecretId.removeprefix("arn:")].pop(key, None)
