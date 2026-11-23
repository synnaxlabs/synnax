#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes an hour of history and then live data to `qa_*` channels until killed.

Usage: uv run python qa/live.py PORT [--no-history]
"""

import argparse

import numpy as np

import synnax as sy

RATE = 50
HISTORY = sy.TimeSpan.HOUR
STEP = int(sy.TimeSpan.SECOND) // RATE


def stamps(start: sy.TimeStamp, count: int) -> np.ndarray:
    """Returns count exact stamps from start, RATE per second."""
    return int(start) + np.arange(count, dtype=np.int64) * STEP


def frame(stamps: np.ndarray) -> dict[str, np.ndarray]:
    # The phase comes from the stamp, so the signals stay smooth across a restart.
    t = stamps / int(sy.TimeSpan.SECOND)
    rng = np.random.default_rng()
    return {
        "qa_time": stamps,
        "qa_sine": np.sin(t / 10) * 50 + 50,
        "qa_noise": rng.normal(20, 2, t.size),
        "qa_square": (np.sin(t / 30) > 0).astype(np.uint8),
    }


def main(port: int, history: bool) -> None:
    client = sy.Synnax(
        host="localhost", port=port, username="synnax", password="seldon"
    )
    index = client.channels.create(
        name="qa_time",
        is_index=True,
        data_type=sy.DataType.TIMESTAMP,
        retrieve_if_name_exists=True,
    )
    first = frame(np.zeros(1, dtype=np.int64))
    for name, data in list(first.items())[1:]:
        client.channels.create(
            name=name,
            index=index.key,
            data_type=data.dtype,
            retrieve_if_name_exists=True,
        )
    end = sy.TimeStamp.now()
    if history:
        start = end - HISTORY
        count = int(HISTORY.seconds * RATE)
        client.write(start, frame(stamps(start, count)))
        client.ranges.create(name="QA history", time_range=sy.TimeRange(start, end))
    loop = sy.Loop(sy.Rate.HZ * RATE)
    with client.open_writer(end, list(first)) as writer:
        while loop.wait():
            writer.write(frame(np.array([int(sy.TimeStamp.now())], dtype=np.int64)))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("port", type=int)
    parser.add_argument(
        "--no-history",
        action="store_true",
        help="Write only live data, as after a Core restart.",
    )
    args = parser.parse_args()
    main(args.port, not args.no_history)
