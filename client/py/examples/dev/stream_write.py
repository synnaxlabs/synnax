import numpy as np

import synnax as sy

client = sy.Synnax()

LBS003_time = client.channels.create(
    name="LBS003_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

LBS003 = client.channels.create(
    name="LBS003",
    data_type=sy.DataType.FLOAT32,
    index=LBS003_time.key,
    retrieve_if_name_exists=True,
)

LBS002_time = client.channels.create(
    name="LBS002_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

LBS002 = client.channels.create(
    name="LBS002",
    data_type=sy.DataType.FLOAT32,
    index=LBS002_time.key,
    retrieve_if_name_exists=True,
)

LBS003_div_LBS002_time = client.channels.create(
    name="LBS003_div_LBS002_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

LBS003_div_LBS002 = client.channels.create(
    name="LBS003_div_LBS002",
    data_type=sy.DataType.FLOAT32,
    index=LBS003_div_LBS002_time.key,
    retrieve_if_name_exists=True,
)

loop = sy.Loop(sy.Rate.HZ * 10)

with client.open_writer(
    start=sy.TimeStamp.now(), channels=[LBS002, LBS003, LBS002_time, LBS003_time]
) as writer:
    while loop.wait():
        writer.write(
            {
                LBS002.key: np.random.rand(1),
                LBS003.key: np.random.rand(1),
                LBS002_time.key: sy.TimeStamp.now(),
                LBS003_time.key: sy.TimeStamp.now(),
            }
        )
