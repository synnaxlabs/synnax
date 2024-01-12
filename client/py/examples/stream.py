import synnax as sy

client = sy.Synnax()

with client.new_streamer(["sy_node_1_comms"]) as s:
    for frame in s:
        print(frame.series[0])
