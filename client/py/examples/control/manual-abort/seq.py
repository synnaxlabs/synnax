#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax(
    host="10.0.0.210",
    port=9090,
    username="synnax",
    password="seldon"
)

ABORT_BUTTON = "test_bool"
DATA = "USB-6008_ai_0"

abort_ch = client.channels.retrieve(ABORT_BUTTON)

def print_state(auto):
    print(auto.state)
    return abort_ch.key in auto.state

with client.control.acquire(
    name="manual-abort",
    write=[ABORT_BUTTON],
    read=[ABORT_BUTTON, DATA],
) as auto:
    auto.set(ABORT_BUTTON, False)
    auto.wait_until(print_state)
