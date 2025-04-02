#  Copyright 2025 Synnax Labs, Inc.
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

from synnax.channel import ChannelKey
from synnax.hardware.task import JSONConfigMixin, MetaTask, StarterStopperMixin, Task
from synnax.telem import CrudeRate, Rate

TYPE = "sequence"


class Config(BaseModel):
    """Configuration for a sequence task."""

    rate: Rate
    read: list[ChannelKey]
    write: list[ChannelKey]
    script: str
    globals: dict[str, Any] = {}


ZERO_CONFIG = Config(rate=Rate(10), read=[], write=[], script="", globals={})


class StateDetails(BaseModel):
    """State details for a sequence task."""

    running: bool
    message: str


class Sequence(StarterStopperMixin, JSONConfigMixin, MetaTask):
    """A task for executing control sequences on a Synnax cluster."""

    TYPE = TYPE
    config: Config
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        name: str = "Control Sequence",
        rate: CrudeRate = 10,
        read: list[ChannelKey] = [],
        write: list[ChannelKey] = [],
        script: str = "",
        globals: dict[str, Any] = {},
    ):
        if internal is not None:
            self._internal = internal
            self.config = Config.model_validate_json(internal.config)
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = Config(
            rate=Rate(rate), read=read, write=write, script=script, globals=globals
        )
