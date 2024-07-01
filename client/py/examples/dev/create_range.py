import synnax as sy

client = sy.Synnax()

client.ranges.create(
    name="Test Range" + str(sy.TimeStamp.now()),
    time_range=sy.TimeRange(
        sy.TimeStamp.now(), sy.TimeStamp.now() + 10 * sy.TimeSpan.SECOND
    ),
    color="#bada55",
)
