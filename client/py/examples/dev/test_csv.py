import synnax as sy

client = sy.Synnax()

index_channel = client.channels.create(
    name="test_csv_index",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
)

print(
    f"Created index channel {index_channel.name} with key {index_channel.key} and data type {index_channel.data_type}"
)
data_channel = client.channels.create(
    name="test_csv_data",
    data_type=sy.DataType.FLOAT32,
    index=index_channel.key,
)
print(
    f"Created data channel {data_channel.name} with key {data_channel.key} and data type {data_channel.data_type}"
)

end = sy.TimeStamp.now()
start = end - sy.TimeSpan.MINUTE
time_data: list[sy.TimeStamp] = []
data_data: list[float] = []
i = 0
time_i = start
while time_i.before_eq(end):
    time_data.append(time_i)
    data_data.append(i)
    time_i += sy.TimeSpan.SECOND
    i += 1

client.write(start, {index_channel.key: time_data, data_channel.key: data_data})
