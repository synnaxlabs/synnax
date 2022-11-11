from datetime import datetime

import numpy as np

import synnax
from synnax import Synnax, TIME_STAMP_MIN, TIME_STAMP_MAX

s = Synnax(
    host="161.35.124.196",
    port=80,
    username="synnax",
    password="seldon"
)

ch = s.channel.create(
    name="gse.pressure",
    rate=25 * synnax.HZ,
    data_type=synnax.FLOAT64,
)

ch.write(datetime.now(), np.random.rand(100000))

t0 = datetime.now()
d = ch.read(TIME_STAMP_MIN, TIME_STAMP_MAX)
print(d)
print(datetime.now() - t0)

# t0 = datetime.now()
# ch = s.channel.retrieve_by_name("gse.pressure[15] (psi)")[0]
# ch2 = s.channel.retrieve_by_name("gse.timestamp (ul)")[0]
#
# d = ch.read(TIME_STAMP_MIN, TIME_STAMP_MAX)
# t = ch2.read(TIME_STAMP_MIN, TIME_STAMP_MAX)
#
# t1 = datetime.now()
# plt.plot(t, d)
# plt.show()
