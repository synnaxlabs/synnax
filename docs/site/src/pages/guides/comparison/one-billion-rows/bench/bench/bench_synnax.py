from bench.config import (
    new_data_iterator,
    ITERATIONS,
    START_TIME,
    TOTAL_ROWS,
)
import synnax as sy
from contextlib import contextmanager
import docker
import time


@contextmanager
def start_synnax():
    client = docker.from_env()
    NAME = "synnax_bench"
    VOLUME_NAME = "synnax_data"
    try:
        volume = client.volumes.get(VOLUME_NAME)
        print(f"Docker volume '{VOLUME_NAME}' already exists.")
        volume.remove()
    except docker.errors.NotFound:
        volume = client.volumes.create(VOLUME_NAME)
        print(f"Docker volume '{VOLUME_NAME}' created.")

    # Define volume bindings
    volumes = {
        VOLUME_NAME: {'bind': '/var/lib/synnax/data', 'mode': 'rw'}
    }
    client.containers.run(
        "synnaxlabs/synnax",
        detach=True,
        name=NAME,
        ports={"9090/tcp": 9090},
        environment=[f"SYNNAX_INSECURE=true", "SYNNAX_DATA=/var/lib/synnax/data"],
        volumes=volumes,
    )
    while True:
        try:
            time.sleep(0.25)
            sy.Synnax()
            break
        except Exception:
            ...
    try:
        yield
    finally:
        c = client.containers.get(NAME)
        c.stop()
        c.remove()


def bench_synnax():
    with start_synnax():
        client = sy.Synnax()
        time_ch = client.channels.create(
            name="time",
            data_type="timestamp",
            is_index=True,
            retrieve_if_name_exists=True
        )
        client.channels.create(
            name="data",
            data_type="float32",
            index=time_ch.key,
            retrieve_if_name_exists=True
        )
        total_time = sy.TimeSpan.SECOND * 0
        with client.open_writer(
            start=START_TIME,
            channels=["time", "data"],
            enable_auto_commit=True,
            auto_index_persist_interval=sy.TimeSpan(-1),
        ) as w:
            for i, df in enumerate(new_data_iterator()):
                perf_start = sy.TimeStamp.now()
                w.write(df)
                total_time += sy.TimeSpan.since(perf_start)
                print(f"Iteration {i + 1}/{ITERATIONS} completed.")

        return total_time
