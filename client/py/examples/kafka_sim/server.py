#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A mock Kafka plant: an Apache Kafka broker plus a producer that simulates sensors.

Each topic gets one record of each kind at the given rate. Records have no key unless
a topic says so.

- ``plant.sensors``: JSON ``{"temperature", "pressure", "count", "state"}``.
- ``plant.scalar``: A bare number.
- ``plant.timed``: JSON ``{"value", "timestamp"}``, where the timestamp is in
  nanoseconds since the epoch.
- ``plant.events``: Two kinds with the key ``pump``, JSON ``{"speed"}`` and
  ``{"fault"}``, and JSON ``{"speed"}`` with the key ``valve``.
- ``plant.mixed``: JSON ``{"count", "value"}``, where the value of every tenth record
  is the string ``"ERR"``.

Run it directly for the examples in this directory:

    uv run python examples/kafka_sim/server.py
"""

import argparse
import asyncio
import json
import math
import signal
import tempfile
import time
from pathlib import Path
from typing import override
from uuid import uuid4

from confluent_kafka import Producer

import synnax as sy
from examples.kafka_sim.broker import Broker, install
from examples.simulators.device_sim import DeviceSim
from synnax import kafka


async def run_server(
    host: str, port: int, controller_port: int, *, rate_hz: float = 10
) -> None:
    """Run the mock plant until it is cancelled or receives SIGTERM, then stop the
    broker."""
    main = asyncio.current_task()
    assert main is not None
    asyncio.get_running_loop().add_signal_handler(signal.SIGTERM, main.cancel)
    broker = Broker(
        install(),
        host=host,
        port=port,
        controller_port=controller_port,
        data_dir=Path(tempfile.gettempdir()) / f"synnax-kafka-sim-{port}",
    )
    try:
        await broker.start()
        producer = Producer({"bootstrap.servers": f"{host}:{port}"})

        def send(topic: str, value: object, key: str | None = None) -> None:
            producer.produce(topic, json.dumps(value), key=key)

        count = 0
        start = time.monotonic()
        while True:
            t = time.monotonic() - start
            count += 1
            send(
                "plant.sensors",
                {
                    "temperature": 23.5 + 5 * math.sin(t),
                    "pressure": 101.3 + 2 * math.cos(t / 2),
                    "count": count,
                    "state": "running" if count % 20 < 10 else "idle",
                },
            )
            send("plant.scalar", round(50 + 10 * math.sin(t), 3))
            send("plant.timed", {"value": math.sin(t), "timestamp": time.time_ns()})
            send("plant.events", {"speed": 1500 + 100 * math.cos(t)}, key="pump")
            send("plant.events", {"fault": count % 4}, key="pump")
            send("plant.events", {"speed": 20 + math.sin(t)}, key="valve")
            value = "ERR" if count % 10 == 0 else math.cos(t)
            send("plant.mixed", {"count": count, "value": value})
            producer.poll(0)
            await asyncio.sleep(1 / rate_hz)
    finally:
        await broker.stop()


class KafkaSim(DeviceSim):
    """The mock Kafka plant as a device simulator. It needs a POSIX host."""

    host = "127.0.0.1"
    # Not 9092 and 9093, so that the simulator can run next to a local broker.
    port = 19092
    controller_port = 19093
    startup_timeout = 60 * sy.TimeSpan.SECOND
    device_name = "Kafka Test Cluster"

    @override
    def start(self) -> None:
        # The first download takes longer than the startup timeout.
        install()
        super().start()

    @override
    def stop(self, timeout: sy.TimeSpan = 30 * sy.TimeSpan.SECOND) -> None:
        super().stop(timeout)

    @override
    async def _run_server(self) -> None:
        await run_server(
            self.host, self.port, self.controller_port, rate_hz=float(self.rate)
        )

    @override
    @staticmethod
    def create_device(client: sy.Synnax, rack_key: int) -> sy.Device:
        broker = f"{KafkaSim.host}:{KafkaSim.port}"
        return sy.Device(
            key=str(uuid4()),
            rack=rack_key,
            location=broker,
            make="kafka",
            model="cluster",
            name=KafkaSim.device_name,
            configured=True,
            properties=kafka.Properties(brokers=[broker]).model_dump(mode="json"),
        )


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Mock Kafka plant")
    parser.add_argument("--host", default=KafkaSim.host)
    parser.add_argument("--port", type=int, default=KafkaSim.port)
    parser.add_argument("--controller-port", type=int, default=KafkaSim.controller_port)
    parser.add_argument("--rate", type=float, default=10, help="Records per second")
    args = parser.parse_args()
    print(f"Mock Kafka plant on {args.host}:{args.port}. Press Ctrl+C to stop.")
    try:
        asyncio.run(
            run_server(args.host, args.port, args.controller_port, rate_hz=args.rate)
        )
    except KeyboardInterrupt:
        pass
