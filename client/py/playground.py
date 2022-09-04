import time
from datetime import datetime

import numpy

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
    tend: telem.TimeStamp = 0
    w = client.data.new_writer([ch.key])
    n_seg = 100
    n_samples = int(30000)
    for i in range(n_seg):
        tend = t0.add(ch.rate.span(n_samples) * i)
        w.write(
            ch.key,
            np.random.rand(n_samples),
            tend
        )
    w.close()
    tend = tend.add(ch.rate.span(n_samples))
    tr = telem.TimeRange(t0, tend)
    _iter = client.data.new_iterator([ch.key], tr)
    averages = []
    t0 = datetime.now()
    c = 0
    _iter.first()
    # averages.append(np.average(_iter.value.get(ch.key).data))
    while _iter.next():
        c += 1
        # averages.append(np.average(_iter.value.get(ch.key).data))
    print(datetime.now() - t0)
    _iter.close()
    print(c)
