#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import freighter
import synnax as sy
from x.testutil import assert_eventually


def assert_eventually_channels_are_found(client: sy.Synnax, keys: list[int]) -> None:
    assert_eventually(lambda: client.channels.retrieve(keys))


class RetrieveCounter:
    """A middleware that counts the channel retrieve requests a client sends."""

    count: int

    def __init__(self) -> None:
        self.count = 0

    def __call__(
        self, ctx: freighter.Context, next: freighter.Next
    ) -> freighter.Context:
        # The HTTP transport puts the URL in protocol, so check both fields.
        if "/channel/retrieve" in ctx.target + ctx.protocol:
            self.count += 1
        return next(ctx)
