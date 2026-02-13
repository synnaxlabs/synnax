#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from typing import Any

from pydantic import BaseModel

from synnax import channel, task
from synnax.telem import CrudeRate, Rate

TYPE = "sequence"


class Config(BaseModel):
    """Configuration for a sequence task."""

    rate: Rate
    read: list[channel.Key]
    write: list[channel.Key]
    script: str
    globals: dict[str, Any] = {}


ZERO_CONFIG = Config(rate=Rate(10), read=[], write=[], script="", globals={})


class StateDetails(BaseModel):
    """State details for a sequence task."""

    running: bool
    message: str


class Sequence(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A task for executing control sequences on a Synnax cluster."""

    TYPE = TYPE
    config: Config
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        name: str = "Control Sequence",
        rate: CrudeRate = 10,
        read: list[channel.Key] = [],
        write: list[channel.Key] = [],
        script: str = "",
        globals: dict[str, Any] = {},
    ):
        if internal is not None:
            self._internal = internal
            self.config = Config.model_validate_json(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = Config(
            rate=Rate(rate), read=read, write=write, script=script, globals=globals
        )
