#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import Payload

from synnax.status import Status


class TaskStatusDetails(Payload):
    """
    Details about the status of a task.
    """

    task: int = 0
    """The key of the task."""
    running: bool = False
    """Whether the task is running."""
    data: dict | None = None
    """Arbitrary data about the task."""


TaskStatus = Status[TaskStatusDetails]
"""The status of a task."""


class TaskPayload(Payload):
    """A primitive task payload."""

    key: int = 0
    name: str = ""
    type: str = ""
    config: str = ""
    snapshot: bool = False
    status: TaskStatus | None = None
