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

import synnax as sy

# Centralized results directory for all test artifacts (screenshots, CSVs, etc.)
RESULTS_DIR = os.path.join(os.path.dirname(__file__), "..", "tests", "results")

# Fixtures directory for test data (SVGs, JSONs, etc.)
FIXTURES_DIR = os.path.join(os.path.dirname(__file__), "..", "tests", "fixtures")


def get_results_path(filename: str) -> str:
    """Get the full path for a results file, ensuring the directory exists."""
    os.makedirs(RESULTS_DIR, exist_ok=True)
    return os.path.join(RESULTS_DIR, filename)


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
