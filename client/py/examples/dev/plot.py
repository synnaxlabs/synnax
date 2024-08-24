import synnax as sy
import numpy as np

client = sy.Synnax()

data = client.read(sy.TimeRange(1722464238568134144, 1722464246886652416), "press_pt_1")

print("Average", np.average(data))
