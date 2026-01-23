#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

from pydantic import BaseModel

from synnax.task import JSONConfigMixin, StarterStopperMixin, Task, TaskProtocol

TYPE = "arc"


class ArcTaskConfig(BaseModel):
    """Configuration for an Arc task."""

    arc_key: str
    """The key of the Arc program to execute (UUID as string)."""
    auto_start: bool = False
    """Whether to start the task automatically when created."""


class ArcTask(StarterStopperMixin, JSONConfigMixin, TaskProtocol):
    """A task for executing Arc programs on a Synnax cluster."""

    TYPE = TYPE
    config: ArcTaskConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        name: str = "Arc Task",
        arc_key: UUID | str | None = None,
        auto_start: bool = False,
    ):
        """Initialize an Arc task.

        :param internal: Internal task object (used when loading from server).
        :param name: Human-readable name for the task.
        :param arc_key: The key of the Arc program to execute.
        :param auto_start: Whether to start the task automatically.
        """
        if internal is not None:
            self._internal = internal
            self.config = ArcTaskConfig.model_validate_json(internal.config)
            return
        if arc_key is None:
            raise ValueError("arc_key is required when creating a new ArcTask")
        self._internal = Task(name=name, type=self.TYPE)
        self.config = ArcTaskConfig(arc_key=str(arc_key), auto_start=auto_start)
