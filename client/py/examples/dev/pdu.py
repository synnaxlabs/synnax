#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
import functools

import synnax as sy
import time

client = sy.Synnax()


def write_pdu(output: str, value: int):
    ...


N_VALVES = 1

with client.open_streamer([f"fridge_vlv_{i}_state" for i in range(N_VALVES)]) as stream:
    data = stream.read()
    for k, s in data.items():
        write_pdu(k, s[0])
