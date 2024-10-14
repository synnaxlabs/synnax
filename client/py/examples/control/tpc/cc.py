import synnax as sy
from uuid import uuid4

client = sy.Synnax()

ox_pt_1 = client.channels.retrieve("ox_pt_1")
ox_pt_2 = client.channels.retrieve("ox_pt_2")


doubled = client.channels.create(
    name="squared 55",
    data_type=sy.DataType.FLOAT64,
    virtual=True,
    requires=[ox_pt_1.key, ox_pt_2.key],
    expression="result = ox_pt_1 - ox_pt_1",
    retrieve_if_name_exists=True,
)

with client.open_streamer([doubled.key]) as s:
    for data in s:
        print(data[doubled.key].series[0])
