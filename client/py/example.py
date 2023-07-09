import synnax as sy
import matplotlib.pyplot as plt

client = sy.Synnax()

pressures = client.channels.retrieve("ec.pressure[14] (hs)")

tr = sy.TimeRange(1681142943939013600, 1681142945321432000)

pressure_data, _ = pressures.read(tr.start, tr.end)

time = client.channels.retrieve("Time (hs)")

time_data, _ = time.read(tr.start, tr.end)

pressures_2 = pressure_data * 2
plt.plot(time_data, pressures_2)
plt.show()
