#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel, Field, model_validator

from synnax import task


class AlertConfig(BaseModel):
    """Configuration for a single PagerDuty alert, mapping a Synnax status to an alert.

    :param status: The Synnax status key to watch for changes.
    :param treat_error_as_critical: If True, maps error variant to "critical" severity
        instead of "error".
    :param component: Component of the source machine (e.g., "mysql", "eth0").
    :param group: Logical grouping of components (e.g., "app-stack").
    :param alert_class: Class/type of the event (e.g., "ping failure", "cpu load").
        Serialized as "class" in JSON to match the Go server schema.
    :param enabled: Whether this alert is active.
    """

    status: str
    treat_error_as_critical: bool = False
    component: str = ""
    group: str = ""
    alert_class: str = Field(default="", alias="class")
    enabled: bool = True

    model_config = {"populate_by_name": True}


class AlertTaskConfig(task.BaseConfig):
    """Configuration for a PagerDuty alert task.

    :param routing_key: The 32-character Integration Key for a PagerDuty service
        integration or global ruleset.
    :param alerts: List of alert configurations mapping Synnax statuses to PagerDuty
        alerts.
    """

    routing_key: str
    alerts: list[AlertConfig]

    @model_validator(mode="after")
    def _validate_config(self) -> "AlertTaskConfig":
        if len(self.routing_key) != 32:
            raise ValueError("routing_key must be exactly 32 characters")
        if not any(a.enabled for a in self.alerts):
            raise ValueError("at least one alert must be enabled")
        return self


class AlertTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A PagerDuty alert task that watches Synnax statuses and sends PagerDuty events.

    When a watched status changes to an error, warning, or info variant, a PagerDuty
    trigger event is sent. When the status returns to success, a resolve event is sent.

    :param internal: An existing task for deserialization.
    :param name: A human-readable name for the task.
    :param routing_key: The 32-character PagerDuty Integration Key.
    :param auto_start: Whether to start the task automatically when configured.
    :param alerts: List of alert configurations.
    """

    TYPE = "pagerduty_alert"
    config: AlertTaskConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        name: str = "",
        routing_key: str = "",
        auto_start: bool = False,
        alerts: list[AlertConfig] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = AlertTaskConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = AlertTaskConfig(
            routing_key=routing_key,
            auto_start=auto_start,
            alerts=alerts if alerts is not None else [],
        )

    def to_payload(self) -> task.Payload:
        pld = self._internal.to_payload()
        pld.config = self.config.model_dump(by_alias=True, exclude_none=True)
        return pld
