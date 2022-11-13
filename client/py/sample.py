from datetime import datetime

import numpy as np

import synnax
from synnax import Synnax, TIME_STAMP_MIN, TIME_STAMP_MAX

s = Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon"
)

ch = s.channel.create(
    name="gse.pressure",
    rate=25 * synnax.HZ,
    data_type=synnax.FLOAT64,
)

ch.write(datetime.now(), np.random.rand(1000000))

t0 = datetime.now()
d = ch.read(TIME_STAMP_MIN, TIME_STAMP_MAX)
