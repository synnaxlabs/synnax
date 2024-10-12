import synnax as sy

client = sy.Synnax()

ch = client.channels.create(
    name="strings",
    virtual=True,
    data_type=sy.DataType.STRING,
    retrieve_if_name_exists=True,
)

loop = sy.Loop(sy.Rate.HZ * 5)

i = 0

with client.open_writer(sy.TimeStamp.now(), [ch.key]) as w:
    while loop.wait():
        i += 1
        w.write(ch.key, [f"Sphinx {i}"])