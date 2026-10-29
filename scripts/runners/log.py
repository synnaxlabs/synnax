#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The log client every core script routes its output through: each line goes to a
virtual string channel, one sample per line, and to stdout only when echo is on."""

import synnax as sy

from common import Channels
from publisher import Publisher


class Log:
    """A failed channel write reports to stdout, since it cannot report to itself."""

    def __init__(
        self, *, client: sy.Synnax, channels: Channels, channel: str, echo: bool = False
    ) -> None:
        self.channel = channel
        self.echo = echo
        channels.virtual(channel, data_type=sy.DataType.STRING)
        self.publisher = Publisher(
            client=client, name=channel, channels=[channel], report=to_stdout
        )

    def __call__(self, line: str) -> None:
        if self.echo:
            print(line, flush=True)
        self.publisher.write({self.channel: [line]})

    def error(self, what: str, e: Exception) -> None:
        self(f"[ERROR] {what} failed: {type(e).__name__}: {e}")

    def close(self) -> None:
        self.publisher.close()


def to_stdout(what: str, e: Exception) -> None:
    print(f"[ERROR] {what} failed: {type(e).__name__}: {e}", flush=True)
