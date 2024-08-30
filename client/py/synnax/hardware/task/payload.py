#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from freighter import Payload


class TaskPayload(Payload):
    key: int = 0
    name: str = ""
    type: str = ""
    config: str = ""
    snapshot: bool = False


class TaskState(Payload):
    task: int = 0
    variant: str = ""
    key: str = ""
    details: dict

    def __init__(
        self, task: int = 0, variant: str = "", key: str = "", details: dict | str = ""
    ):
        if isinstance(details, str):
            details = json.loads(details)
        super().__init__(task=task, variant=variant, key=key, details=details)
