import time
from datetime import datetime

import numpy

import arya
import numpy as np

from arya import telem

if __name__ == "__main__":
    client = arya.Client(host="localhost", port=3457)
    ch = client.channel.create(
        name="test",
        data_type=np.float64,
        rate=25,
        node_id=1,
    )
    time.sleep(2)
    t0 = telem.now()
    w = client.data.write(ch.key, np.random.rand(300), telem.now())
    tend = t0.add(ch.rate.span(300))
    data = client.data.read(ch.key, telem.TIME_RANGE_MAX)
    print(data)
    print(len(data))
