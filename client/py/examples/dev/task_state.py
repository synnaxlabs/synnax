import synnax as sy

client = sy.Synnax()


with client.open_streamer(["sy_rack16_meminfo"]) as s:
    for frame in s:
        print(sy.TimeStamp.now())
        print(sy.Size.BYTE * frame["sy_rack16_meminfo"][0])
