import synnax as sy

client = sy.Synnax(
    host="141.212.23.215",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False
)

tr = sy.TimeRange(1707070831214829300, 1707070848461897000)

print(tr)

data = client.read(tr, "gse_ai_time")

print(tr.span)

print(len(data), data.time_range.span)
