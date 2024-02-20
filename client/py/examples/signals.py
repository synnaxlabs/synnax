import synnax as sy
import asyncio
import matplotlib.pyplot as plt

client = sy.Synnax()


@client.ranges.on_create
def handle(rng: sy.Range):
    plt.plot(rng['Time'], rng['Data'])
    plt.savefig("range.png")


if __name__ == "__main__":
    asyncio.run(client.signals.process())
