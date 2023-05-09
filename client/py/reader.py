import synnax as sy

client = sy.Synnax()

with client.stream(sy.TimeStamp.now(), "my_chan") as r:
    for v in r:
        print(v)
