import synnax as sy 
import time

client = sy.Synnax()

idx = client.channels.create(
    name="index",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True
)

NUM_CHANNELS = int(1e4)

channels = client.channels.create([
    sy.Channel(
        name=f"channel_{i}",
        is_index=False,
        index=idx.key,
        data_type=sy.DataType.FLOAT32
    ) for i in range(NUM_CHANNELS)
],
retrieve_if_name_exists=True)

with client.open_writer(sy.TimeStamp.now(),[idx, *channels]) as writer:
    while True:
        time.sleep(0.1)
        writer.write(
            {
                "index": sy.TimeStamp.now(),
                **{f"channel_{j}": 12 for j in range(NUM_CHANNELS)}
            }
        )