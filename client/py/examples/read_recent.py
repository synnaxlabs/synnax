import synnax as sy

client = sy.Synnax()

index_ch = client.channels.create(
    name="index",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

data_ch = client.channels.create(
    name="data",
    index=index_ch.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)

# n_samples = 100
# start = sy.TimeStamp.now() - 100 * sy.TimeSpan.SECOND
# time_data = [start + sy.TimeSpan.SECOND * i for i in range(n_samples)]
# data_data = [i for i in range(n_samples)]

# client.write(start, index_ch.key, time_data)
# client.write(start, data_ch.key, data_data)

iterator = client.open_iterator(
    tr=sy.TimeRange.MAX,
    channels=[index_ch.key, data_ch.key],
)
print(iterator.seek_last())
print(iterator.prev(sy.AUTO_SPAN))
print(iterator.value)
