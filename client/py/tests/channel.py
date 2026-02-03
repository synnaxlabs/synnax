#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from tests.eventually import assert_eventually


def assert_eventually_channels_are_found(client: sy.Synnax, keys: list[int]):
    assert_eventually(lambda: client.channels.retrieve(keys))
