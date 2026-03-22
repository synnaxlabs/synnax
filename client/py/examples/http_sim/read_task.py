#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to read data from a mock HTTP server using
the HTTP driver.

Before running this example:
1. Start the mock HTTP server:
   uv run python examples/http_sim/server.py

2. Connect the HTTP device in Synnax:
   uv run python examples/http_sim/connect_server.py

3. Run this script:
   uv run python examples/http_sim/read_task.py

The mock server exposes GET /api/v1/data returning:
    {"temperature": 23.5, "pressure": 101.3, "humidity": 45.0}
"""

import synnax as sy
from synnax import http

client = sy.Synnax()

# Retrieve the HTTP device
dev = client.devices.retrieve(name="HTTP Server")

# Create an index channel for timestamps
http_time = client.channels.create(
    name="http_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create data channels
temperature = client.channels.create(
    name="http_temperature",
    index=http_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)

pressure = client.channels.create(
    name="http_pressure",
    index=http_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)

humidity = client.channels.create(
    name="http_humidity",
    index=http_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)

# Create the HTTP Read Task
tsk = http.ReadTask(
    name="HTTP Py - Read Task",
    device=dev.key,
    rate=1,  # Poll at 1 Hz
    data_saving=True,
    endpoints=[
        http.ReadEndpoint(
            method="GET",
            path="/api/v1/data",
            fields=[
                http.ReadField(
                    pointer="/temperature",
                    channel=temperature.key,
                    data_type="float64",
                    name="Temperature",
                ),
                http.ReadField(
                    pointer="/pressure",
                    channel=pressure.key,
                    data_type="float64",
                    name="Pressure",
                ),
                http.ReadField(
                    pointer="/humidity",
                    channel=humidity.key,
                    data_type="float64",
                    name="Humidity",
                ),
            ],
        ),
    ],
)

# Configure the task with Synnax
try:
    client.tasks.configure(tsk)
    print("✓ Task configured successfully")
except Exception as e:
    print(f"✗ Task configuration failed: {e}")
    exit(1)

print("=" * 70)
print("Starting HTTP Read Task")
print("=" * 70)
print("Polling GET /api/v1/data at 1 Hz...")
print("Running continuously - Press Ctrl+C to stop\n")

print(f"{'Sample':<10} {'Temp (°C)':>12} {'Pressure (kPa)':>16} {'Humidity (%)':>14}")
print("-" * 70)

try:
    print("\033[?25l", end="", flush=True)

    with tsk.run():
        with client.open_streamer(
            ["http_temperature", "http_pressure", "http_humidity"]
        ) as streamer:
            sample_count = 0
            while True:
                frame = streamer.read()
                if frame and "http_temperature" in frame:
                    temp = frame["http_temperature"][-1]
                    pres = frame["http_pressure"][-1]
                    hum = frame["http_humidity"][-1]
                    sample_count += 1
                    print(
                        f"{sample_count:<10} {temp:>12.2f} {pres:>16.2f} {hum:>14.2f}",
                        end="\r",
                        flush=True,
                    )

except KeyboardInterrupt:
    print("\n" + "-" * 70)
    print("✓ Read task stopped by user")
    print(f"\nCollected {sample_count} samples")
    print("=" * 70)
finally:
    print("\033[?25h", end="", flush=True)
