import synnax as sy
import matplotlib.pyplot as plt
import numpy as np

client = sy.Synnax()

data = client.ranges.retrieve("April 9 Wetdress")

data.ec_pressure_12.set_alias("flowmeter_dp")

elapsed = sy.elapsed_seconds(data.Time)

for chan in data["ec_vlv_*"]:
    # plot dashed
    plt.plot(elapsed, chan, "--", label=chan.name)

plt.show()


# press_mask = data.flowmeter_dp > 5
# valid_times = data.Time[press_mask]
# valid_pressures = data.flowmeter_dp[press_mask]
# elapsed_times = sy.elapsed_seconds(valid_times)
#
# plt.plot(elapsed_times, valid_pressures)
# plt.xlabel("Elapsed Time (s)")
# plt.ylabel("Pressure (psi)")
# plt.show()
#
# # grab the peak pressure from valid_pressures
# # grab the average pressure from valid_pressures
# data.meta_data.set("peak_pressure", valid_pressures.max())
# print(data.meta_data.get("peak_pressure"))
# data.meta_data.set("avg_pressure", valid_pressures.mean())
# print(data.meta_data.get("avg_pressure"))
