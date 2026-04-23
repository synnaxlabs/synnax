#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration setup: ranges with children, channels, aliases, and sample data."""

from collections.abc import Callable

import numpy as np

import synnax as sy

SETUP_VERSION = "0.49"

EPOCH = sy.TimeStamp(1_000_000_000 * sy.TimeSpan.SECOND)

PARENT_NAME = "mig_range_parent"
PARENT_COLOR = "#E63946"
PARENT_TR = sy.TimeRange(EPOCH, EPOCH + 100 * sy.TimeSpan.SECOND)

CHILDREN = [
    (
        "mig_range_child_1",
        "#457B9D",
        sy.TimeRange(EPOCH, EPOCH + 40 * sy.TimeSpan.SECOND),
    ),
    (
        "mig_range_child_2",
        "#2A9D8F",
        sy.TimeRange(EPOCH + 50 * sy.TimeSpan.SECOND, EPOCH + 90 * sy.TimeSpan.SECOND),
    ),
]

DATA_NAME = "mig_range_data"
ALIAS_NAME = "mig_range_sensor"

DATA_VALUES = np.array(
    [1.1, 2.2, 3.3, 4.4, 5.5, 6.6, 7.7, 8.8, 9.9, 10.10],
    dtype=np.float64,
)


def setup(client: sy.Synnax, log: Callable[[str], None]) -> None:
    log("  [ranges] Creating parent range...")

    parent = client.ranges.create(
        name=PARENT_NAME,
        time_range=PARENT_TR,
        color=PARENT_COLOR,
    )

    log("  [ranges] Creating child ranges...")
    for name, color, tr in CHILDREN:
        parent.create_child_range(name=name, time_range=tr, color=color)

    log("  [ranges] Creating channels and writing data...")
    idx = client.channels.create(
        name="mig_range_idx",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )
    data_ch = client.channels.create(
        name=DATA_NAME,
        data_type=sy.DataType.FLOAT64,
        index=idx.key,
        retrieve_if_name_exists=True,
    )

    sample_count = len(DATA_VALUES)
    timestamps = np.array(
        [EPOCH + i * sy.TimeSpan.SECOND for i in range(sample_count)],
        dtype=np.int64,
    )
    with client.open_writer(
        start=PARENT_TR.start,
        channels=[idx.key, data_ch.key],
        name="mig_ranges_writer",
        enable_auto_commit=True,
    ) as writer:
        writer.write({idx.key: timestamps, data_ch.key: DATA_VALUES})

    log("  [ranges] Setting alias...")
    parent.set_alias(DATA_NAME, ALIAS_NAME)


if __name__ == "__main__":
    from setup import run

    run(setup)
