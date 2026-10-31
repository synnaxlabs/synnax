#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from common import Channels, channel_name
from conftest import Client, core
from events import MAX_PENDING, Events

RUNNER = "ubuntu-build-bot-i-0f44"
CHANNEL = channel_name("events", runner=RUNNER)


def events(client: Client) -> Events:
    return Events(
        client=core(client),
        channels=Channels(client=core(client), enabled=False),
        runners={RUNNER},
    )


class TestEvents:
    """Tests for writing job and step lines to each runner's events channel."""

    def test_should_write_lines_that_failed_before(self, client: Client) -> None:
        e = events(client)
        client.failing = True
        e.write({RUNNER: ["started"]})
        client.failing = False
        e.write({RUNNER: ["done"]})
        assert client.written(CHANNEL) == [["started", "done"]]

    def test_should_write_each_line_once(self, client: Client) -> None:
        e = events(client)
        e.write({RUNNER: ["started"]})
        e.write({RUNNER: ["done"]})
        assert client.written(CHANNEL) == [["started"], ["done"]]

    def test_should_keep_only_the_newest_pending_lines(self, client: Client) -> None:
        e = events(client)
        client.failing = True
        e.write({RUNNER: [str(i) for i in range(MAX_PENDING + 1)]})
        assert e.pending[RUNNER][0] == "1"

    def test_should_drop_pending_lines_of_removed_runners(self, client: Client) -> None:
        e = events(client)
        client.failing = True
        e.write({RUNNER: ["started"]})
        e.retarget(set())
        assert e.pending == {}
