#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import cast

import numpy as np
import pytest
import synnax as sy


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
