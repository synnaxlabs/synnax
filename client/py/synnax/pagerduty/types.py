#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax import task
from synnax.pagerduty.types_gen import Alert, TaskConfig


class AlertTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A PagerDuty alert task that watches Synnax statuses and sends PagerDuty events.

    When a watched status changes to an error, warning, or info variant, a PagerDuty
    trigger event is sent. When the status returns to success, a resolve event is sent.

    :param internal: An existing task for deserialization.
    :param name: A human-readable name for the task.
    :param routing_key: The 32-character PagerDuty Integration Key.
    :param auto_start: Whether to start the task automatically when configured.
    :param alerts: The alerts the task evaluates.
    """

    TYPE = "pagerduty_alert"
    config: TaskConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        name: str = "",
        routing_key: str = "",
        auto_start: bool = False,
        alerts: list[Alert] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = TaskConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = TaskConfig(
            routing_key=routing_key,
            auto_start=auto_start,
            alerts=alerts if alerts is not None else [],
        )
        task.assign_keys(self.config.alerts)
