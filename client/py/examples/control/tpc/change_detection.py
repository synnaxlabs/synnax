import synnax as sy

client = sy.Synnax()

with client.open_streamer(["fuel_pt_1", "daq_time"]) as s:
    above_threshold = None
    for value in s:
        if value["fuel_pt_1"] > 20 and not above_threshold:
            above_threshold = sy.TimeStamp(value["daq_time"][-1])
        elif value["fuel_pt_1"] < 20 and above_threshold:
            client.ranges.create(
                name=f"Fuel Above Threshold - " + str(above_threshold)[11:19],
                time_range=sy.TimeRange(
                    start=above_threshold, end=value["daq_time"][-1]
                ),
                color="#BADA55",
            )
            above_threshold = None
