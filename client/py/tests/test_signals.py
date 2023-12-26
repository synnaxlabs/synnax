from synnax import Synnax

client = Synnax()


@client.signals.on(["gse_ai_20"], lambda c: c > 20)
def on_pressure(c):
    print("Pressure is")


@client.ranges.on_create(lambda r: r.labels.contains("test"))
def on_range_create(c):
    print("Range created")


