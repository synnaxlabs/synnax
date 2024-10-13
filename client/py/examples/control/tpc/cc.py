import synnax as sy
from uuid import uuid4

client = sy.Synnax()

ox_pt_1 = client.channels.retrieve("ox_pt_1")

name = str(uuid4())

doubled = client.channels.create(
    name=name,
    data_type=sy.DataType.FLOAT64,
    virtual=True,
    requires=[ox_pt_1.key],
    expression="result = np.where(ox_pt_1 > 0, 1, 0)",
    retrieve_if_name_exists=True,
)

print(doubled.expression)

with client.open_streamer(["ox_pt_1", name]) as s:
    for data in s:
        print(list(data[name]))