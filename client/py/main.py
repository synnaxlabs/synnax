import synnax
import matplotlib.pyplot as plt


client = synnax.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon"
)

ch = client.channel.retrieve(name="gse.pressure[7] (psi)")
tCH = client.channel.retrieve(name="Time")

data = ch.read(0, synnax.TIME_STAMP_MAX)
t_data = tCH.read(0, synnax.TIME_STAMP_MAX)

plt.plot(t_data, data)
plt.show()



