#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import datetime as dt
from typing import Literal, cast

import numpy as np
import pytest
import synnax as sy
from mypy_boto3_secretsmanager.type_defs import (
    FilterTypeDef,
    SecretListEntryTypeDef,
    TagTypeDef,
)

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


def core(client: Client) -> sy.Synnax:
    return cast(sy.Synnax, client)


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
