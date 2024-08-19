#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
import numpy as np
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
    print(f"New range created: {rng.name}")
    time = np.array(sy.elapsed_seconds(rng["daq_time"]))
    pressure = np.array(rng["press_pt_1"])
    plt.plot(time, pressure, label=rng.name)
    min_time = np.min(time)
    max_time = np.max(time)
    min_pressure = np.min(pressure)
    max_pressure = np.max(pressure)
    plt.plot(min_time, min_pressure, "ro")
    plt.plot(max_time, max_pressure, "ro")
    plt.title(f"{rng.name} - Fuel Pressure")
    plt.xlabel("Time (s)")
    plt.ylabel("Pressure (PSI)")
    plt.legend()
    plt.savefig(f"{rng.name}.png")


if __name__ == "__main__":
    asyncio.run(client.signals.process())
