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
    task: int = 0
    running: bool = False
    data: dict | None = None


TaskStatus = Status[TaskStatusDetails]


class TaskPayload(Payload):
    key: int = 0
    name: str = ""
    type: str = ""
    config: str = ""
    snapshot: bool = False
    status: TaskStatus | None = None
