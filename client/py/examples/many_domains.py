import synnax as sy
import numpy as np

client = sy.Synnax()

time_ch = client.channels.create(
    name="time",
    data_type="timestamp",
    is_index=True,
    retrieve_if_name_exists=True,
)

data_ch = client.channels.create(
    name="data",
    data_type="float32",
    index=time_ch.key,
    retrieve_if_name_exists=True,
)

start = sy.TimeStamp.now()
count = int(1e5)

for i in range(7):
    print(start)
    stamps = np.linspace(start, start + 30 * sy.TimeSpan.MINUTE, count, dtype=np.int64)
    data = np.sin(np.linspace(0, 2 * np.pi, count))
    time_ch.write(
        start=start,
        data=stamps,
    )
    data_ch.write(
        start=start,
        data=data,
    )
    start = start + sy.TimeSpan.HOUR * 10

tr = sy.TimeRange.MAX

with client.open_iterator(tr, [data_ch.key]) as it:
    i = 0
    for frame in it:
        print(len(frame.get(data_ch.key)))
        i+=1


