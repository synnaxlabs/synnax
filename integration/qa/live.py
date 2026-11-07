#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes an hour of history and then live data to `qa_*` channels until killed.

Usage: uv run python qa/live.py PORT
"""

import sys

import numpy as np

import synnax as sy

RATE = 50
HISTORY = sy.TimeSpan.HOUR


def frame(stamps: np.ndarray, t: np.ndarray) -> dict[str, np.ndarray]:
    rng = np.random.default_rng()
    return {
        "qa_time": stamps,
        "qa_sine": np.sin(t / 10) * 50 + 50,
        "qa_noise": rng.normal(20, 2, t.size),
        "qa_square": (np.sin(t / 30) > 0).astype(np.uint8),
    }


def main(port: int) -> None:
    client = sy.Synnax(
        host="localhost", port=port, username="synnax", password="seldon"
    )
    index = client.channels.create(
        name="qa_time",
        is_index=True,
        data_type=sy.DataType.TIMESTAMP,
        retrieve_if_name_exists=True,
    )
    first = frame(np.zeros(1, dtype=np.int64), np.zeros(1))
    for name, data in list(first.items())[1:]:
        client.channels.create(
            name=name,
            index=index.key,
            data_type=data.dtype,
            retrieve_if_name_exists=True,
        )
    end = sy.TimeStamp.now()
    start = end - HISTORY
    count = int(HISTORY.seconds * RATE)
    stamps = np.linspace(int(start), int(end), count, endpoint=False, dtype=np.int64)
    t = np.arange(count) / RATE
    client.write(start, frame(stamps, t))
    client.ranges.create(name="QA history", time_range=sy.TimeRange(start, end))
    loop = sy.Loop(sy.Rate.HZ * RATE)
    with client.open_writer(end, list(first)) as writer:
        i = count
        while loop.wait():
            now = np.array([int(sy.TimeStamp.now())], dtype=np.int64)
            writer.write(frame(now, np.array([i / RATE])))
            i += 1


if __name__ == "__main__":
    main(int(sys.argv[1]))
