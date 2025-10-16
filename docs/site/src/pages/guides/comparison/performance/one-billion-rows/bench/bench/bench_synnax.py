#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from bench.config import TestConfig
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


def bench_synnax(cfg: TestConfig):
    with start_synnax():
        client = sy.Synnax()
        time_ch = client.channels.create(cfg.channels[0])
        cfg.channels[0] = time_ch
        for ch in cfg.channels[1:]:
            ch.index = time_ch.key
        oc = client.channels.create(cfg.channels[1:])
        cfg.channels[1:] = oc
        print(cfg.channels)
        total_time = sy.TimeSpan.SECOND * 0
        print(cfg.channels)
        with client.open_writer(
            start=cfg._start_time,
            channels=[c.key for c in cfg.channels],
            enable_auto_commit=True,
            auto_index_persist_interval=sy.TimeSpan(-1),
        ) as w:
            for i, df in enumerate(cfg.frames(index=False)):
                perf_start = sy.TimeStamp.now()
                w.write(df)
                total_time += sy.TimeSpan.since(perf_start)
                print(f"Iteration {i + 1}/{cfg.iterations} completed.")
        return total_time
