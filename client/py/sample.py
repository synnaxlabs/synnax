from datetime import datetime

import numpy as np

from synnax import Synnax

s = Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon"
)

t0 = datetime.now()
ch = s.channel.retrieve_by_name("gse.pressure[15] (psi)")[0]

d = ch.read(0,100000000000000)

t1 = datetime.now()
print(len(d))
print(t1-t0)
