import time
from contextlib import contextmanager
import docker
from typing import Callable
from dataclasses import dataclass
import json
import threading

from synnax import TimeStamp, TimeSpan


class Stats:
    def __init__(
        self,
        memory: int,
        volume: int,
        cpu: int,
    ):
        self.memory = memory
        self.volume = volume
        self.cpu = cpu



def peak_stats(stats: list[Stats]) -> Stats:
    # Get the peak memory usage
    peak_memory = max([stat.memory for stat in stats])
    peak_volume = max([stat.volume for stat in stats])
    peak_cpu = max([stat.cpu for stat in stats])
    return Stats(
        memory=peak_memory,
        volume=peak_volume,
        cpu=peak_cpu,
    )


def get_docker_volume_size(
    client: docker.DockerClient,
    volume_name: str,
) -> int:
    try:
        # Fetch the volume details
        volume = client.volumes.get(volume_name)

        # Get the volume's usage data
        usage = client.df().get('Volumes', [])

        for vol in usage:
            if vol['Name'] == volume_name:
                # 'UsageData' contains size information, e.g. 'Size', 'RefCount'
                return vol['UsageData']['Size']

        return None
    except docker.errors.NotFound:
        print(f"Volume {volume_name} not found.")
        return None

def get_stats_factory(
    client: docker.DockerClient,
    container_name: str,
) -> tuple[Callable[[], None], Callable[[], Stats]]:
    c = client.containers.get(container_name)
    stats_stream = c.stats(stream=True)

    stats_bytes = next(stats_stream)
    stats = json.loads(stats_bytes)

    def get_curr_stats():
        while True:
            try:
                nonlocal stats
                stats = json.loads(next(stats_stream))
            except StopIteration:
                return

    t = threading.Thread(target=get_curr_stats)
    t.start()

    def get_stats():
        # Memory usage
        memory_usage = stats.get('memory_stats', {}).get('usage', 0)
        memory_limit = stats.get('memory_stats', {}).get('limit', 0)

        # CPU usage
        cpu_stats = stats.get('cpu_stats', {})
        precpu_stats = stats.get('precpu_stats', {})

        # Handle potential missing keys
        cpu_delta = cpu_stats.get('cpu_usage', {}).get('total_usage', 0) - precpu_stats.get(
            'cpu_usage', {}).get('total_usage', 0)
        system_delta = cpu_stats.get('system_cpu_usage', 0) - precpu_stats.get(
            'system_cpu_usage', 0)
        online_cpus = cpu_stats.get('online_cpus',
                                    1)  # Default to 1 if not available to avoid division by zero

        if system_delta > 0 and online_cpus > 0:
            cpu_usage = (cpu_delta / system_delta) * online_cpus * 100.0
        else:
            cpu_usage = 0.0
        # volume = get_docker_volume_size(client, container_name)
        return Stats(
            memory=memory_usage,
            volume=0,
            cpu=cpu_usage,
        )

    return get_stats

@contextmanager
def run_container(
    image: str,
    name: str,
    volume_name: str,
    ports: dict[str, int],
    environment: list[str],
    volume_dir: str,
    health_check: Callable[[], bool],
) -> Callable[[], Stats]:
    print(f"Starting container '{name}'")
    client = docker.from_env()
    try:
        volume = client.volumes.get(volume_name)
        print(f"Docker volume '{volume_name}' already exists. Removing")
        volume.remove()
        print(f"Docker volume '{volume_name}' removed.")
    except docker.errors.NotFound:
        client.volumes.create(volume_name)
        print(f"Docker volume '{volume_name}' created.")
    volumes = {
        volume_name: {"bind": volume_dir, "mode": "rw"}
    }
    client.containers.run(
        image=image,
        detach=True,
        name=name,
        ports=ports,
        environment=environment,
        volumes=volumes,
    )
    if health_check is not None:
        while True:
            try:
                if health_check():
                    break
            except Exception:
                ...
    try:
        gs = get_stats_factory(client, name)
        yield gs
    finally:
        c = client.containers.get(name)
        c.stop()
        c.remove()

