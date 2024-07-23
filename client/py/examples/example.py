import synnax as sy
import numpy as np

client = sy.Synnax()

data = client.read(sy.TimeRange(1721697090932228864, 1721697096529795072), "press_pt_1")

print(np.average(data))