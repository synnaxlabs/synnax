import synnax as sy

client = sy.Synnax()

with client.open_streamer("sy_device_set") as s:
    for frame in s:
        print(frame)
