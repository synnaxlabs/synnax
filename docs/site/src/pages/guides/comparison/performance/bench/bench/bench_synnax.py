from bench.config import TestConfig
from bench.dockerutil import run_container, peak_stats
import synnax as sy
from contextlib import contextmanager
import docker
import time


@contextmanager
def start_synnax():
    NAME = "synnax_bench"
    VOLUME_NAME = "synnax_data"

    client = docker.from_env()

    try:
        volume = client.volumes.get(VOLUME_NAME)
        print(f"Docker volume '{VOLUME_NAME}' already exists.")
        volume.remove()
        print(f"Docker volume '{VOLUME_NAME}' removed.")
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
        environment=[f"SYNNAX_INSECURE=true", "SYNNAX_DATA=/var/lib/synnax/data",
                     "SYNNAX_LICENSE_KEY=885508-64317283-9499549876"],
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

        # get the memory usage of the container
        stats = c.stats(stream=False)
        print(stats)

        c.stop()
        c.remove()


IMAGE_NAME = "synnaxlabs/synnax"
CONTAINER_NAME = "synnax_bench"
VOLUME_NAME = "synnax_data"
VOLUME_PATH = "/var/lib/synnax/data"
PORTS = {"9090/tcp": 9090}
ENVIRONMENT = [
    f"SYNNAX_INSECURE=true",
    f"SYNNAX_DATA={VOLUME_PATH}",
    f"SYNNAX_LICENSE_KEY=885508-64317283-9499549876",
]


def health_check():
    try:
        sy.Synnax()
        return True
    except Exception:
        return False


def bench_synnax(
    cfg: TestConfig,
):
    with run_container(
        image=IMAGE_NAME,
        name=CONTAINER_NAME,
        volume_name=VOLUME_NAME,
        ports=PORTS,
        environment=ENVIRONMENT,
        volume_dir=VOLUME_PATH,
        health_check=health_check,
    ) as get_stats:
        client = sy.Synnax(read_timeout=30*sy.TimeSpan.SECOND)
        time_ch = client.channels.create(cfg.channels[0])
        cfg.channels[0] = time_ch
        for ch in cfg.channels[1:]:
            ch.index = time_ch.key
        oc = client.channels.create(cfg.channels[1:])
        cfg.channels[1:] = oc
        total_time = sy.TimeSpan.SECOND * 0
        stats = list()
        with client.open_writer(
            start=cfg._start_time,
            channels=[c.name for c in cfg.channels],
            enable_auto_commit=True,
            auto_index_persist_interval=sy.TimeSpan(-1),
        ) as w:
            for i, df in enumerate(cfg.frames(index=False)):
                perf_start = sy.TimeStamp.now()
                w.write(df)
                total_time += sy.TimeSpan.since(perf_start)
                stats.append(get_stats())
                print(f"Iteration {i + 1}/{cfg.iterations} completed.")
        return peak_stats(stats), total_time
