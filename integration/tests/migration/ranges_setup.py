#!/usr/bin/env python3

#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: ranges with children, channels, aliases, and sample data.

Standalone script — uses only synnax + stdlib.
Once committed, this file is never modified.
"""

import numpy as np
from setup import S, log, run

import synnax as sy

SETUP_VERSION = "0.49"


def setup(client: sy.Synnax) -> None:
    log("  [ranges] Creating parent range...")

    EPOCH = sy.TimeStamp(1_000_000_000 * S)
    PARENT_TR = sy.TimeRange(EPOCH, EPOCH + 100 * S)

    parent = client.ranges.create(
        name="mig_range_parent",
        time_range=PARENT_TR,
        color="#E63946",
    )

    log("  [ranges] Creating child ranges...")
    children = [
        ("mig_range_child_1", "#457B9D", sy.TimeRange(EPOCH, EPOCH + 40 * S)),
        (
            "mig_range_child_2",
            "#2A9D8F",
            sy.TimeRange(EPOCH + 50 * S, EPOCH + 90 * S),
        ),
    ]
    for name, color, tr in children:
        parent.create_child_range(name=name, time_range=tr, color=color)

    log("  [ranges] Creating channels and writing data...")
    idx = client.channels.create(
        name="mig_range_idx",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
    )
    data_ch = client.channels.create(
        name="mig_range_data",
        data_type=sy.DataType.FLOAT64,
        index=idx.key,
    )

    sample_count = 10
    timestamps = np.array(
        [EPOCH + i * S for i in range(sample_count)],
        dtype=np.int64,
    )
    data_values = np.array(
        [1.1, 2.2, 3.3, 4.4, 5.5, 6.6, 7.7, 8.8, 9.9, 10.10],
        dtype=np.float64,
    )
    with client.open_writer(
        start=PARENT_TR.start,
        channels=[idx.key, data_ch.key],
        name="mig_ranges_writer",
        enable_auto_commit=True,
    ) as writer:
        writer.write({idx.key: timestamps, data_ch.key: data_values})

    log("  [ranges] Setting alias...")
    parent.set_alias("mig_range_data", "mig_range_sensor")


run(setup)
