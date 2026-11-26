#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A single-node Apache Kafka broker in KRaft mode, for examples and tests.

The first use downloads the pinned Apache Kafka release, checks its SHA-512, and caches
it in ``~/.cache/synnax-sim/kafka``. The broker needs a POSIX shell, and Java 17 or
later on the PATH or in JAVA_HOME.
"""

import asyncio
import base64
import hashlib
import os
import tarfile
import tempfile
import urllib.request
from pathlib import Path

VERSION = "4.4.0"
RELEASE = f"kafka_2.13-{VERSION}"
URL = f"https://archive.apache.org/dist/kafka/{VERSION}/{RELEASE}.tgz"
SHA512 = (
    "f5ceba8a265ab6bb2d68362b1f6859a95e15d60ef92ba83b43fdf065b557dc18"
    "e98d6383c15b9977320e25b3a2c8b6f11decb17b0fa6c0f163423793a0eb95cb"
)
# Not in ~/.cache/synnax, which the Core owns on Linux.
CACHE = Path.home() / ".cache" / "synnax-sim" / "kafka"
# A restart must format with the ID already on disk.
CLUSTER_ID = base64.urlsafe_b64encode(b"synnax-kafka-sim").rstrip(b"=").decode()
STOP_TIMEOUT = 20


def install(cache: Path = CACHE) -> Path:
    """Return the home folder of the pinned release in cache. The first call downloads
    the release.

    :raises RuntimeError: If the download does not match the pinned SHA-512.
    """
    home = cache / RELEASE
    if home.is_dir():
        return home
    cache.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=cache) as tmp:
        archive = Path(tmp) / f"{RELEASE}.tgz"
        digest = hashlib.sha512()
        with urllib.request.urlopen(URL, timeout=60) as res, archive.open("wb") as f:
            while chunk := res.read(1 << 20):
                digest.update(chunk)
                f.write(chunk)
        if digest.hexdigest() != SHA512:
            raise RuntimeError(f"{URL} does not match its pinned SHA-512")
        with tarfile.open(archive) as tar:
            tar.extractall(tmp, filter="data")
        (Path(tmp) / RELEASE).rename(home)
    return home


class Broker:
    """A broker that listens for clients on host:port and for its controller on
    controller_port. It keeps its topics and offsets in data_dir, so they survive a
    restart. Its logs go to the ``logs`` folder of data_dir.

    :param home: The home folder of a Kafka release, from install.
    """

    def __init__(
        self,
        home: Path,
        *,
        host: str,
        port: int,
        controller_port: int,
        data_dir: Path,
    ) -> None:
        self._home = home
        self._host = host
        self._port = port
        self._controller_port = controller_port
        self._data_dir = data_dir
        self._process: asyncio.subprocess.Process | None = None

    async def start(self) -> None:
        """Format data_dir if it is new, then start the broker. The broker accepts
        clients a few seconds after this returns.

        :raises RuntimeError: If the format fails.
        """
        logs = self._data_dir / "logs"
        logs.mkdir(parents=True, exist_ok=True)
        config = self._data_dir / "server.properties"
        config.write_text(self._properties())
        env = {**os.environ, "LOG_DIR": str(logs), "KAFKA_HEAP_OPTS": "-Xmx512m"}
        fmt = await asyncio.create_subprocess_exec(
            str(self._home / "bin" / "kafka-storage.sh"),
            "format",
            "--standalone",
            "--ignore-formatted",
            "--cluster-id",
            CLUSTER_ID,
            "--config",
            str(config),
            env=env,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.STDOUT,
        )
        out, _ = await fmt.communicate()
        if fmt.returncode != 0:
            raise RuntimeError(f"Kafka storage format failed: {out.decode()}")
        with (logs / "console.log").open("ab") as console:
            self._process = await asyncio.create_subprocess_exec(
                str(self._home / "bin" / "kafka-server-start.sh"),
                str(config),
                env=env,
                stdout=console,
                stderr=console,
            )

    async def stop(self) -> None:
        """Stop the broker. Kill it if it does not stop within 20 seconds."""
        if self._process is None or self._process.returncode is not None:
            return
        self._process.terminate()
        try:
            await asyncio.wait_for(self._process.wait(), STOP_TIMEOUT)
        except TimeoutError:
            self._process.kill()
            await self._process.wait()

    def _properties(self) -> str:
        client = f"{self._host}:{self._port}"
        controller = f"{self._host}:{self._controller_port}"
        lines = [
            "process.roles=broker,controller",
            "node.id=1",
            f"controller.quorum.bootstrap.servers={controller}",
            f"listeners=PLAINTEXT://{client},CONTROLLER://{controller}",
            f"advertised.listeners=PLAINTEXT://{client},CONTROLLER://{controller}",
            "controller.listener.names=CONTROLLER",
            "inter.broker.listener.name=PLAINTEXT",
            "listener.security.protocol.map=PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT",
            f"log.dirs={self._data_dir / 'kraft'}",
            "num.partitions=1",
            "offsets.topic.replication.factor=1",
            "share.coordinator.state.topic.replication.factor=1",
            "share.coordinator.state.topic.min.isr=1",
            "transaction.state.log.replication.factor=1",
            "transaction.state.log.min.isr=1",
            "group.initial.rebalance.delay.ms=0",
        ]
        return "\n".join(lines) + "\n"
