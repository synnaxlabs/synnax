#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A mock MQTT plant: a broker plus a publisher that simulates sensors.

Topics:

- ``plant/sensors``: JSON ``{"temperature", "pressure", "count", "state"}`` at the
  given rate.
- ``plant/timed``: JSON ``{"value", "timestamp"}``, where the timestamp is in
  nanoseconds since the epoch.
- ``plant/scalar``: A bare number.
- ``plant/info``: A retained JSON message, published once.
- ``plant/<name>/set``: A command topic. The plant publishes each payload it receives
  again, retained, on ``plant/<name>/state``.

Sparkplug B: the edge node ``Plant/Line1`` has the tags ``temperature``, ``count``, and
``running``. Its device ``Pump1`` has the tags ``speed`` and ``setpoint``. A command
for a tag sets its value. The plant also mirrors every edge node on the broker: each
tag appears, retained, on ``plant/sparkplug/<group>/<edge node>/<tag>`` as JSON
``{"value": ...}``, and a value published on that topic plus ``/set`` becomes a command
for the tag.

Run it directly for the examples in this directory:

    uv run python examples/mqtt_sim/server.py
"""

import argparse
import asyncio
import json
import math
import time

from examples.mqtt_sim import sparkplug
from examples.mqtt_sim.broker import Broker
from examples.simulators.device_sim import DeviceSim
from synnax import mqtt

COMMAND_PREFIX = "plant/"
COMMAND_SUFFIX = "/set"
SPARKPLUG_GROUP = "Plant"
SPARKPLUG_EDGE_NODE = "Line1"
SPARKPLUG_DEVICE = "Pump1"


async def run_server(host: str, port: int, rate_hz: float = 10) -> None:
    """Runs the mock plant until it is cancelled."""
    broker = Broker(host, port)

    def echo(topic: str, payload: bytes) -> None:
        if topic.startswith(COMMAND_PREFIX) and topic.endswith(COMMAND_SUFFIX):
            name = topic[len(COMMAND_PREFIX) : -len(COMMAND_SUFFIX)]
            broker.publish(f"plant/{name}/state", payload, retain=True)

    broker.on_message(echo)
    temperature = sparkplug.Tag("temperature", sparkplug.DOUBLE, 0.0)
    counter = sparkplug.Tag("count", sparkplug.INT64, 0)
    running = sparkplug.Tag("running", sparkplug.BOOLEAN, False)
    speed = sparkplug.Tag("speed", sparkplug.FLOAT, 0.0)
    setpoint = sparkplug.Tag("setpoint", sparkplug.DOUBLE, 0.0)
    edge_node = sparkplug.EdgeNode(
        broker,
        SPARKPLUG_GROUP,
        SPARKPLUG_EDGE_NODE,
        [temperature, counter, running],
        {SPARKPLUG_DEVICE: [speed, setpoint]},
    )
    sparkplug.HostMirror(broker)
    await broker.start()
    edge_node.birth()
    broker.publish(
        "plant/info", json.dumps({"name": "Mock plant", "version": 1}), retain=True
    )
    count = 0
    start = time.monotonic()
    try:
        while True:
            t = time.monotonic() - start
            count += 1
            broker.publish(
                "plant/sensors",
                json.dumps(
                    {
                        "temperature": 23.5 + 5 * math.sin(t),
                        "pressure": 101.3 + 2 * math.cos(t / 2),
                        "count": count,
                        "state": "running" if count % 20 < 10 else "idle",
                    }
                ),
            )
            broker.publish(
                "plant/timed",
                json.dumps({"value": math.sin(t), "timestamp": time.time_ns()}),
            )
            broker.publish("plant/scalar", repr(round(50 + 10 * math.sin(t), 3)))
            temperature.value = 23.5 + 5 * math.sin(t)
            counter.value = count
            running.value = count % 20 < 10
            speed.value = 1500 + 100 * math.cos(t)
            edge_node.data()
            edge_node.data(SPARKPLUG_DEVICE, {speed.name})
            await asyncio.sleep(1 / rate_hz)
    finally:
        edge_node.death()
        await broker.stop()


class MQTTSim(DeviceSim):
    """The mock MQTT plant as a device simulator."""

    host = "127.0.0.1"
    # Not 1883, so that the simulator can run next to a local broker.
    port = 1884
    device_name = "MQTT Test Broker"

    async def _run_server(self) -> None:
        await run_server(self.host, self.port, float(self.rate))

    @staticmethod
    def create_device(rack_key: int) -> mqtt.Device:
        return mqtt.Device(
            host=MQTTSim.host,
            port=MQTTSim.port,
            name=MQTTSim.device_name,
            rack=rack_key,
        )


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Mock MQTT plant")
    parser.add_argument("--host", default=MQTTSim.host)
    parser.add_argument("--port", type=int, default=MQTTSim.port)
    parser.add_argument("--rate", type=float, default=10, help="Messages per second")
    args = parser.parse_args()
    print(f"Mock MQTT plant on {args.host}:{args.port}. Press Ctrl+C to stop.")
    try:
        asyncio.run(run_server(args.host, args.port, args.rate))
    except KeyboardInterrupt:
        pass
