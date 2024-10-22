import time

from bench.config import TestConfig
from bench.dockerutil import run_container, peak_stats
import synnax as sy
from influxdb_client import InfluxDBClient
from influxdb_client.client.write_api import WriteOptions

PORT = 8086
URL = f"http://localhost:{PORT}"
TOKEN = "i0pa4Dd2IePcaxErwfsk-kjW7lOqMX6FjCJ5bKRHb8x6RMJ3d-o8Jp34eIzHBcw5kaFT2LOcjTVIWvBj8JaT5Q=="
ORG = "myorg"
BUCKET = "mybucket"


IMAGE_NAME = "influxdb"
CONTAINER_NAME = "influxdb_bench"
VOLUME_NAME = "influxdb_data"
VOLUME_DIR = "/var/lib/influxdb2"
PORTS = {"8086/tcp": 8086}
ENVIRONMENT = [
    "DOCKER_INFLUXDB_INIT_MODE=setup",
    "DOCKER_INFLUXDB_INIT_USERNAME=admin",
    "DOCKER_INFLUXDB_INIT_PASSWORD=adminpassword",
    f"DOCKER_INFLUXDB_INIT_ORG={ORG}",
    f"DOCKER_INFLUXDB_INIT_BUCKET={BUCKET}",
    f"DOCKER_INFLUXDB_INIT_ADMIN_TOKEN={TOKEN}",
]

def bench_influx(cfg: TestConfig):
    with run_container(
        image=IMAGE_NAME,
        name=CONTAINER_NAME,
        volume_name=VOLUME_NAME,
        ports=PORTS,
        environment=ENVIRONMENT,
        volume_dir=VOLUME_DIR,
        health_check=None,
    ) as get_stats:
        client = InfluxDBClient(url=URL, token=TOKEN, org=ORG)
        write_api = client.write_api(write_options=WriteOptions(batch_size=5000))
        total_time = sy.TimeSpan.SECOND * 0
        stats = list()
        for i, df in enumerate(cfg.frames(index=True)):
            perf_start = sy.TimeStamp.now()
            write_api.write(
                bucket=BUCKET,
                org=ORG,
                record=df,
                data_frame_measurement_name="data",
            )
            total_time += sy.TimeStamp.since(perf_start)
            stats.append(get_stats())
            print(f"Iteration {i + 1}/{cfg.iterations} completed.")
        write_api.close()
        client.close()
        return peak_stats(stats), total_time
