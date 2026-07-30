#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Integration test utilities — helpers specific to test execution.

Only integration-specific helpers belong here (fixture paths, results paths,
assertion helpers, etc.). General-purpose utilities (OS info, string helpers,
WebSocket handling, color conversion, etc.) belong in the ``x`` package
located at ``x/py/``.
"""

import os
import re
import uuid
from typing import Any

import synnax as sy

# Fixtures directory for test data (SVGs, JSONs, etc.)
FIXTURES_DIR = os.path.join(os.path.dirname(__file__), "..", "tests", "fixtures")


def get_fixture_path(filename: str) -> str:
    """Get the full path for a test fixture file.

    Args:
        filename: Name of the fixture file (e.g., "test_valve.svg")

    Returns:
        Full path to the fixture file.

    Raises:
        FileNotFoundError: If the fixture file doesn't exist.
    """
    path = os.path.join(FIXTURES_DIR, filename)
    if not os.path.exists(path):
        raise FileNotFoundError(f"Test fixture not found: {path}")
    return path


def create_time_index(client: sy.Synnax, name: str) -> sy.Channel:
    """Create (or retrieve) a timestamp index channel."""
    return client.channels.create(
        name=name,
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )


def create_virtual_channel(
    client: sy.Synnax,
    name: str,
    data_type: sy.DataType = sy.DataType.FLOAT32,
) -> sy.Channel:
    """Create (or retrieve) a virtual channel."""
    return client.channels.create(
        name=name,
        data_type=data_type,
        virtual=True,
        retrieve_if_name_exists=True,
    )


def create_virtual_channels(
    client: sy.Synnax,
    specs: list[tuple[str, sy.DataType]],
) -> list[sy.Channel]:
    """Create (or retrieve) many virtual channels in a single round-trip."""
    channels = [
        sy.Channel(name=name, data_type=data_type, virtual=True)
        for name, data_type in specs
    ]
    created: list[sy.Channel] = client.channels.create(
        channels, retrieve_if_name_exists=True
    )
    return created


def create_indexed_channel(
    client: sy.Synnax,
    name: str,
    data_type: sy.DataType,
    index_key: int,
) -> sy.Channel:
    """Create (or retrieve) a data channel indexed to a time channel."""
    return client.channels.create(
        name=name,
        data_type=data_type,
        index=index_key,
        retrieve_if_name_exists=True,
    )


def create_indexed_pair(
    client: sy.Synnax,
    name: str,
    data_type: sy.DataType = sy.DataType.FLOAT64,
) -> sy.Channel:
    """Create a time index channel and a data channel indexed to it.

    The index channel is named ``{name}_time``. Returns the data channel.
    """
    idx = create_time_index(client, f"{name}_time")
    return create_indexed_channel(client, name, data_type, idx.key)


LINK_PATTERN = re.compile(r"^synnax://cluster/([^/]+)/([^/]+)/([^/]+)$")


def assert_link_format(
    link: str, resource_type: str, resource_id: str | None = None
) -> None:
    """Assert that a synnax:// link matches the expected format.

    :param link: The link to validate.
    :param resource_type: The resource type (e.g., "lineplot", "log", "channel").
    :param resource_id: Optional specific resource ID to match. If None, validates as UUID.
    """
    match = LINK_PATTERN.match(link)
    assert match, f"Link should match synnax://cluster/<uuid>/<type>/<id>, got: {link}"

    cluster_id, actual_type, actual_id = match.groups()

    try:
        uuid.UUID(cluster_id)
    except ValueError:
        raise AssertionError(f"Cluster ID should be a valid UUID, got: {cluster_id}")

    assert actual_type == resource_type, (
        f"Resource type should be '{resource_type}', got: {actual_type}"
    )

    if resource_id is not None:
        assert actual_id == resource_id, (
            f"Resource ID should be '{resource_id}', got: {actual_id}"
        )
    else:
        try:
            uuid.UUID(actual_id)
        except ValueError:
            raise AssertionError(
                f"Resource ID should be a valid UUID, got: {actual_id}"
            )


def assert_envelope(
    exported: dict[str, Any],
    type: str,
    min_version: int,
    name: str | None = None,
) -> None:
    """Assert that a server-side export has the portable envelope shape.

    :param exported: The decoded export JSON.
    :param type: The expected envelope type (e.g., "lineplot", "schematic").
    :param min_version: Floor for the envelope version — a floor, not an exact match, so
        server-side version bumps don't break export tests.
    :param name: Optional resource name the envelope must carry.
    """
    assert exported.get("type") == type, (
        f"Envelope type should be {type!r}, got {exported.get('type')!r}"
    )
    version = exported.get("version")
    assert isinstance(version, int) and version >= min_version, (
        f"Envelope version should be an int >= {min_version}, got {version!r}"
    )
    assert "key" not in exported, (
        "Server-side export strips the resource key from the portable envelope"
    )
    if name is not None:
        assert exported.get("name") == name, (
            f"Envelope name should be {name!r}, got {exported.get('name')!r}"
        )
