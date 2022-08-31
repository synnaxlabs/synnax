import time
from datetime import datetime

import arya
import numpy as np

from arya import telem

if __name__ == "__main__":
    client = arya.Client(host="localhost", port=3456)
    ch = client.channel.create(
        name="test",
        data_type=np.float64,
        rate=25,
        node_id=1,
    )
    t0 = telem.now()
    w = client.data.new_writer([ch.key])
    for i in range(100):
        w.write(
            ch.key,
            np.random.rand(100000),
            t0.add(ch.rate.span(100000) * i)
        )
    w.close()
    rng = t0.span_range(ch.rate.span(100 * 100))
    i = client.data.new_iterator([ch.key], rng)
    i.first()
    while i.next():
        pass
    print(i.close())





