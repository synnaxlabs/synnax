#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax()

ch = client.channels.create(
    name="strings",
    virtual=True,
    data_type=sy.DataType.STRING,
    retrieve_if_name_exists=True,
)

loop = sy.Loop(sy.Rate.HZ * 20)

i = 0

with client.open_writer(sy.TimeStamp.now(), [ch.key]) as w:
    while loop.wait():
        i += 1
        w.write(ch.key, [f"Sphinx {i}"])
