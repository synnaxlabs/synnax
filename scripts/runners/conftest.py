#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import cast

import pytest
import synnax as sy


class Writer:
    def __init__(self, client: "Client") -> None:
        self.client = client

    def write(self, frame: dict) -> None:
        if self.client.failing:
            raise ConnectionError("Core unreachable")
        self.client.frames.append(frame)

    def set_authority(self, authority: int) -> None:
        pass

    def close(self) -> None:
        pass


class Streamer:
    def read(self, timeout: sy.TimeSpan) -> None:
        return None

    def close(self) -> None:
        pass


class Client:
    """Stands in for the Core: records every frame written, and fails writes while
    failing is set."""

    def __init__(self) -> None:
        self.frames: list[dict] = []
        self.failing = False

    def open_writer(self, *args: object, **kwargs: object) -> Writer:
        return Writer(self)

    def open_streamer(self, *args: object, **kwargs: object) -> Streamer:
        return Streamer()

    def read_latest(self, names: list[str]) -> dict:
        return {}

    def written(self, channel: str) -> list[object]:
        return [f[channel] for f in self.frames if channel in f]


@pytest.fixture
def client() -> Client:
    return Client()


def core(client: Client) -> sy.Synnax:
    return cast(sy.Synnax, client)
