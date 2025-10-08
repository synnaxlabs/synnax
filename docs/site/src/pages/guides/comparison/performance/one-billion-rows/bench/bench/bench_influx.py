#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from bench.config import (
    ITERATIONS,
    new_data_iterator,
)
import synnax as sy
from contextlib import contextmanager
from influxdb_client import InfluxDBClient
from influxdb_client.client.write_api import WriteOptions
import docker
import time

PORT = 8086
URL = f"http://localhost:{PORT}"
TOKEN = "i0pa4Dd2IePcaxErwfsk-kjW7lOqMX6FjCJ5bKRHb8x6RMJ3d-o8Jp34eIzHBcw5kaFT2LOcjTVIWvBj8JaT5Q=="
ORG = "myorg"
BUCKET = "mybucket"


@contextmanager
def start_influxdb():
    client = docker.from_env()
    NAME = "influxdb_bench"
    VOLUME_NAME = "influxdb_data"
    try:
        volume = client.volumes.get(VOLUME_NAME)
        print(f"Docker volume '{VOLUME_NAME}' already exists.")
        volume.remove()
    except docker.errors.NotFound:
        volume = client.volumes.create(VOLUME_NAME)
        print(f"Docker volume '{VOLUME_NAME}' created.")
    volumes = {
        VOLUME_NAME: {"bind": "/var/lib/influxdb2", "mode": "rw"}
    }
    client.containers.run(
        "influxdb",
        detach=True,
        name=NAME,
        ports={"8086/tcp": 8086},
        environment=[
            "DOCKER_INFLUXDB_INIT_MODE=setup",
            "DOCKER_INFLUXDB_INIT_USERNAME=admin",
            "DOCKER_INFLUXDB_INIT_PASSWORD=adminpassword",
            f"DOCKER_INFLUXDB_INIT_ORG={ORG}",
            f"DOCKER_INFLUXDB_INIT_BUCKET={BUCKET}",
            f"DOCKER_INFLUXDB_INIT_ADMIN_TOKEN={TOKEN}",
        ],
        volumes=volumes,
    )
    time.sleep(1)
    try:
        yield
    finally:
        c = client.containers.get(NAME)
        c.stop()
        c.remove()

def bench_influx():
    with start_influxdb():
        client = InfluxDBClient(url=URL, token=TOKEN, org=ORG)
        write_api = client.write_api(write_options=WriteOptions(batch_size=5000))
        total_time = sy.TimeSpan.SECOND * 0
        for i, df in enumerate(new_data_iterator()):
            perf_start = sy.TimeStamp.now()
            df.set_index("time", inplace=True)
            print("iter start")
            write_api.write(
                bucket=BUCKET,
                org=ORG,
                record=df,
                data_frame_measurement_name="data",
            )
            total_time += sy.TimeStamp.since(perf_start)
            print(f"Iteration {i + 1}/{ITERATIONS} completed.")
        write_api.close()
        client.close()
        return total_time
