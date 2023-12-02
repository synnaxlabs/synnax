import uuid

import synnax as sy
from typing import Callable
import json

client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


def test_complete():
    def inner(f: Callable[[sy.Range], None]):
        with client.new_streamer(["sy_range_set"]) as s:
            for r in s:
                rng = json.loads(r["sy_range_set"].data)
                f(client.ranges.retrieve(uuid.UUID(rng["key"])))

    return inner
