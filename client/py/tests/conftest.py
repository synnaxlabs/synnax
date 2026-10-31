#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
from collections.abc import Iterator

import pytest

import synnax as sy
from tests.channel import RetrieveCounter
from tests.proxy import Proxy
from x.strings import random_name

HOST = os.environ.get("SYNNAX_HOST", "localhost")
PORT = int(os.environ.get("SYNNAX_PORT", 9090))
USERNAME = "synnax"
PASSWORD = "seldon"


@pytest.fixture(scope="session")
def login_info() -> tuple[str, int, str, str]:
    return HOST, PORT, USERNAME, PASSWORD


@pytest.fixture(scope="session")
def client() -> sy.Synnax:
    return sy.Synnax(
        host=HOST,
        port=PORT,
        username=USERNAME,
        password=PASSWORD,
        secure=False,
    )


@pytest.fixture
def counted_client() -> tuple[sy.Synnax, RetrieveCounter]:
    """A client that counts the channel retrieve requests it sends."""
    c = sy.Synnax(
        host=HOST,
        port=PORT,
        username=USERNAME,
        password=PASSWORD,
        secure=False,
    )
    counter = RetrieveCounter()
    c._transport.use(counter)
    return c, counter


def _connect(proxy: Proxy) -> sy.Synnax:
    return sy.Synnax(
        host="localhost",
        port=proxy.port,
        username=USERNAME,
        password=PASSWORD,
        secure=False,
    )


@pytest.fixture
def proxy() -> Iterator[Proxy]:
    p = Proxy(HOST, PORT)
    yield p
    p.close()


@pytest.fixture
def proxied_client(proxy: Proxy) -> sy.Synnax:
    return _connect(proxy)


@pytest.fixture
def early_proxy() -> Iterator[Proxy]:
    """A proxy that sends a keep-alive ahead of each streamer's open ack."""
    p = Proxy(HOST, PORT, inject_keep_alive=True)
    yield p
    p.close()


@pytest.fixture
def early_proxied_client(early_proxy: Proxy) -> sy.Synnax:
    return _connect(early_proxy)


@pytest.fixture
def indexed_pair(client: sy.Synnax) -> tuple[sy.Channel, sy.Channel]:
    v = random_name()
    idx = client.channels.create(
        name=f"test_{v}_time",
        is_index=True,
        data_type=sy.DataType.TIMESTAMP,
    )
    data = client.channels.create(
        name=f"test_{v}_data",
        index=idx.key,
        data_type=sy.DataType.FLOAT64,
    )
    return idx, data


@pytest.fixture
def virtual_channel(client: sy.Synnax) -> sy.Channel:
    v = random_name()
    return client.channels.create(
        name=f"test_{v}_virtual",
        virtual=True,
        data_type=sy.DataType.FLOAT64,
    )


def pytest_addoption(parser: pytest.Parser) -> None:
    parser.addoption(
        "--run-multi-node",
        action="store_true",
        default=False,
        help="Run tests marked as multi-node",
    )


def pytest_configure(config: pytest.Config) -> None:
    config.addinivalue_line(
        "markers", "multi-node: tests that run only when a flag is provided"
    )


def pytest_collection_modifyitems(
    config: pytest.Config, items: list[pytest.Item]
) -> None:
    if config.getoption("--run-multi-node"):
        # If the flag is provided, no filtering is needed
        return
    # Else, skip tests marked as `multi-node`
    skip_multi_node = pytest.mark.skip(reason="Need --run-multi-node option to run")
    for item in items:
        if "multi_node" in item.keywords:
            item.add_marker(skip_multi_node)
