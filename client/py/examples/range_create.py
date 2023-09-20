import synnax as sy
import matplotlib.pyplot as plt

client = sy.Synnax()

with client.new_streamer("sy_range_set") as s:
    for r in s:
        r = client.ranges.retrieve(r.series[0][0])
        t = r.read("Time (hs)")
        d = r.read("ec.pressure[12] (hs)")
        print(t.__array__(), d.__array__())
        plt.plot(t, d, "r-")
        plt.show()
