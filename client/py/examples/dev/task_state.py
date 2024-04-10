import synnax as sy

client = sy.Synnax()


with client.open_streamer(["sy_task_state"]) as s:
    for frame in s:
        print(frame["sy_task_state"][0])
