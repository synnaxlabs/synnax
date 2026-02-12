#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from pydantic import BaseModel

from synnax.ontology import ID
from synnax.status import Status

ONTOLOGY_TYPE = ID(type="task")


class StatusDetails(BaseModel):
    """
    Details about the status of a task.
    """

    task: int = 0
    """The key of the task."""
    running: bool = False
    """Whether the task is running."""
    data: dict | None = None
    """Arbitrary data about the task."""
    cmd: str | None = None


Status = Status[StatusDetails]
"""The status of a task."""


class Payload(BaseModel):
    """A primitive task payload."""

    key: int = 0
    name: str = ""
    type: str = ""
    config: str = ""
    snapshot: bool = False
    status: Status | None = None


def ontology_id(key: int) -> ID:
    """Create an ontology ID for a task.

    Args:
        key: The task key.

    Returns:
        An ontology ID dictionary with type "task" and the given key.
    """
    return ID(type=ONTOLOGY_TYPE.type, key=str(key))
