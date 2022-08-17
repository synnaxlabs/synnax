import asyncio
import datetime
import time

from delta import telem
from delta.segment import Segment
from delta.segment.writer import Writer
import delta
import numpy as np


async def write(i):
    client = delta.Client("localhost:3456")
    n = 1
    # if i > 50:
    #     i-=50
    #     n = 2

    key = f"{n}-{i}"
    writer = await client.write([key])
    data = np.random.rand(500).tobytes()

    t0 = telem.now()
    try:
        for j in range(0, 50):
            start = (t0 + telem.TimeStamp(telem.TimeSpan(1 * telem.SECOND) * j)).value
            err = await writer.write([
                Segment(
                    channel_key=key,
                    start=start,
                    data=data,
                ),
            ])
    except Exception as e:
        print(e)
    finally:
        await writer.close()


class TestWriter:
    async def test_write(self):
        dt0 = datetime.datetime.now()
        await asyncio.gather(
            *[write(i) for i in range(1, 101)]
        )
        print((datetime.datetime.now() - dt0).total_seconds())
