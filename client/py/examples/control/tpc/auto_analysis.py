import time

import synnax as sy
import matplotlib.pyplot as plt
import asyncio

client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
)


@client.ranges.on_create
def process(rng: sy.Range) -> None:
    time.sleep(2)  # Temp fix to wait for data to commit
    print(f"New range created: {rng.name}")
    plt.plot(sy.elapsed_seconds(rng["daq_time"]), rng["fuel_tank_pt"], label=rng.name)
    plt.title(f"{rng.name} - OX Pressure")
    plt.xlabel("Time (s)")
    plt.ylabel("Pressure (PSI)")
    plt.legend()
    plt.savefig(f"{rng.name}.png")


if __name__ == "__main__":
    asyncio.run(client.signals.process())
