#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import pytest

import synnax
from synnax import telem
from synnax.channel import Channel

HOST = "localhost"
PORT = 9090
USERNAME = "synnax"
PASSWORD = "seldon"


@pytest.fixture(scope="session")
def login_info() -> tuple[str, int, str, str]:
    return HOST, PORT, USERNAME, PASSWORD


@pytest.fixture(scope="session")
def client() -> synnax.Synnax:
    return synnax.Synnax(
        host=HOST,
        port=PORT,
        username=USERNAME,
        password=PASSWORD,
        secure=False,
    )


@pytest.fixture
def channel(client: synnax.Synnax) -> Channel:
    return client.channels.create(
        name=f"test-{random.randint(0, 100000)}",
        leaseholder=1,
        rate=25 * telem.Rate.HZ,
        data_type=telem.DataType.FLOAT64,
    )


@pytest.fixture
def indexed_pair(client: synnax.Synnax) -> tuple[Channel, Channel]:
    v = random.randint(0, 100000)
    idx = client.channels.create(
        name=f"test-{v}-time",
        is_index=True,
        data_type=telem.DataType.TIMESTAMP,
    )
    data = client.channels.create(
        name=f"test-{v}-data",
        index=idx.key,
        data_type=telem.DataType.FLOAT64,
    )
    return idx, data


def pytest_addoption(parser):
    parser.addoption(
        "--run-multi-node",
        action="store_true",
        default=False,
        help="Run tests marked as multi-node",
    )


def pytest_configure(config):
    config.addinivalue_line(
        "markers", "multi-node: tests that run only when a flag is provided"
    )


def pytest_collection_modifyitems(config, items):
    if config.getoption("--run-multi-node"):
        # If the flag is provided, no filtering is needed
        return
    # Else, skip tests marked as `multi-node`
    skip_multi_node = pytest.mark.skip(reason="Need --run-multi-node option to run")
    for item in items:
        if "multi_node" in item.keywords:
            item.add_marker(skip_multi_node)
