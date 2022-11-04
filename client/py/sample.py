import numpy as np

from synnax import Synnax

s = Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon"
)

ch = s.channel.retrieve_by_name("ec.vlv2.i (Amps)")[0]

ch.read(0,100000000000000)
