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
    tend: telem.TimeStamp = 0
    w = client.data.new_writer([ch.key])
    n_seg = 1000
    n_samples = int(1000)
    data = np.random.rand(n_samples)
    for i in range(n_seg):
        tend = t0.add(ch.rate.span(n_samples) * i)
        w.write(
            ch.key,
            data,
            tend
        )
    w.close()
    t0 = datetime.now()
    tend = tend.add(ch.rate.span(n_samples))
    tr = telem.TimeRange(t0, tend)
    res_data = client.data.read_seg(ch.key, tr)
    print(np.array_equal(data, res_data.data))
    print(datetime.now() - t0)
