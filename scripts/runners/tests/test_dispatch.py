#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

import synnax as sy

from common import Channels, channel_name
from conftest import ChannelStore, Client, Streamer
from dispatch import REFRESH, Handlers, Listener
from log import Log
from registry import RunnerRegistry

FIRST = "ubuntu-build-bot-i-0f44"
SECOND = "ubuntu-build-bot-i-0a21"


class Stored(ChannelStore):
    def retrieve(self, name: str, /) -> sy.Channel:
        if name not in self.stored:
            raise sy.NotFoundError(name)
        return self.stored[name]


class Core(Client):
    """Retrieves channels by name and records the channels of each streamer open."""

    def __init__(self) -> None:
        super().__init__()
        self.channels: Stored = Stored()
        self.opened: list[list[str]] = []

    def open_streamer(self, *args: object, **kwargs: object) -> Streamer:
        channels = args[0]
        assert isinstance(channels, list)
        self.opened.append(channels)
        return super().open_streamer()


def create_command_channel(client: Core, runner: str) -> str:
    name = channel_name("ec2_cmd", runner=runner)
    client.channels.create(
        [sy.Channel(name=name, data_type=sy.DataType.BOOLEAN, virtual=True)],
        retrieve_if_name_exists=True,
    )
    return name


def listener(client: Core, *, registry: Path, commands: Handlers) -> Listener:
    return Listener(
        client=client,
        commands=commands,
        log=Log(
            client=client,
            channels=Channels(client=client, enabled=False),
            channel="log",
        ),
        registry=registry,
    )


class TestListener:
    """Tests for streaming the runners' command channels and dispatching each write."""

    def test_should_handle_a_command_on_a_channel_created_after_start(
        self, tmp_path: Path
    ) -> None:
        client = Core()
        registry = RunnerRegistry(tmp_path / "registry.json")
        registry.add({FIRST: set()})
        first = create_command_channel(client, FIRST)
        stopped: list[str] = []
        listening = listener(
            client,
            registry=registry.path,
            commands={"ec2_cmd": {False: stopped.append}},
        )
        listening.step(0.0)
        assert client.opened == [[first]]
        registry.add({SECOND: set()})
        second = create_command_channel(client, SECOND)
        client.streamed.append({second: [False]})
        listening.step(REFRESH)
        assert client.opened == [[first], [second, first]]
        assert stopped == [SECOND]

    def test_should_log_the_first_refresh_with_no_channels(
        self, tmp_path: Path
    ) -> None:
        client = Core()
        registry = RunnerRegistry(tmp_path / "registry.json")
        registry.add({FIRST: set()})
        listening = listener(client, registry=registry.path, commands={"ec2_cmd": {}})
        listening.step(0.0)
        assert client.written("log") == [["listening on 0/1 channels"]]
        assert client.opened == []
