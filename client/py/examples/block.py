import synnax as sy
import time

client = sy.Synnax(
    host="nuc",
    port=9090,
    username="synnax",
    password="seldon"
)

with client.open_streamer(["ox_pt_1", "ox_pt_2"]) as streamer:
    time.sleep(50000)