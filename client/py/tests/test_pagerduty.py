#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy


@pytest.mark.pagerduty
class TestPagerDutyAlertTask:
    """Tests for PagerDuty Alert Task configuration and validation."""

    @pytest.mark.parametrize(
        "test_data",
        [
            {
                "name": "basic_single_alert",
                "data": {
                    "routing_key": "12345678901234567890123456789012",
                    "auto_start": False,
                    "alerts": [
                        {
                            "status": "database-health",
                            "enabled": True,
                            "treat_error_as_critical": True,
                            "component": "postgres",
                            "group": "infrastructure",
                            "class": "database_error",
                        },
                    ],
                },
            },
            {
                "name": "multiple_alerts",
                "data": {
                    "routing_key": "abcdefghijklmnopqrstuvwxyz123456",
                    "auto_start": True,
                    "alerts": [
                        {
                            "status": "sensor-1",
                            "enabled": True,
                            "treat_error_as_critical": False,
                            "component": "temperature-sensor",
                            "group": "hardware",
                            "class": "sensor_anomaly",
                        },
                        {
                            "status": "sensor-2",
                            "enabled": False,
                            "treat_error_as_critical": True,
                            "component": "pressure-sensor",
                            "group": "hardware",
                            "class": "sensor_failure",
                        },
                    ],
                },
            },
            {
                "name": "minimal_alert",
                "data": {
                    "routing_key": "00000000000000000000000000000000",
                    "alerts": [
                        {
                            "status": "my-status",
                            "enabled": True,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_alert_task_config(self, test_data):
        """Test that AlertTaskConfig can parse various configurations."""
        sy.pagerduty.AlertTaskConfig.model_validate(test_data["data"])

    def test_alert_config_defaults(self):
        """Test that AlertConfig has correct defaults."""
        cfg = sy.pagerduty.AlertConfig(status="my-status")
        assert cfg.status == "my-status"
        assert cfg.treat_error_as_critical is False
        assert cfg.component == ""
        assert cfg.group == ""
        assert cfg.alert_class == ""
        assert cfg.enabled is True

    def test_alert_config_class_alias(self):
        """Test that AlertConfig serializes alert_class as 'class' in JSON."""
        cfg = sy.pagerduty.AlertConfig(
            status="my-status",
            alert_class="cpu_load",
        )
        dumped = cfg.model_dump(by_alias=True)
        assert "class" in dumped
        assert dumped["class"] == "cpu_load"
        assert "alert_class" not in dumped

    def test_alert_config_class_from_alias(self):
        """Test that AlertConfig can be created from JSON with 'class' key."""
        cfg = sy.pagerduty.AlertConfig.model_validate(
            {"status": "my-status", "class": "ping_failure", "enabled": True}
        )
        assert cfg.alert_class == "ping_failure"

    def test_alert_task_config_defaults(self):
        """Test that AlertTaskConfig has correct defaults."""
        config = sy.pagerduty.AlertTaskConfig(
            routing_key="12345678901234567890123456789012",
            alerts=[sy.pagerduty.AlertConfig(status="s", enabled=True)],
        )
        assert config.auto_start is False

    def test_routing_key_wrong_length(self):
        """Test that routing_key must be exactly 32 characters."""
        with pytest.raises(
            ValueError, match="routing_key must be exactly 32 characters"
        ):
            sy.pagerduty.AlertTaskConfig(
                routing_key="too-short",
                alerts=[sy.pagerduty.AlertConfig(status="s", enabled=True)],
            )

    def test_routing_key_too_long(self):
        """Test that routing_key rejects keys longer than 32 characters."""
        with pytest.raises(
            ValueError, match="routing_key must be exactly 32 characters"
        ):
            sy.pagerduty.AlertTaskConfig(
                routing_key="a" * 33,
                alerts=[sy.pagerduty.AlertConfig(status="s", enabled=True)],
            )

    def test_no_enabled_alerts(self):
        """Test that at least one alert must be enabled."""
        with pytest.raises(ValueError, match="at least one alert must be enabled"):
            sy.pagerduty.AlertTaskConfig(
                routing_key="12345678901234567890123456789012",
                alerts=[sy.pagerduty.AlertConfig(status="s", enabled=False)],
            )

    def test_empty_alerts_list(self):
        """Test that an empty alerts list fails validation."""
        with pytest.raises(ValueError, match="at least one alert must be enabled"):
            sy.pagerduty.AlertTaskConfig(
                routing_key="12345678901234567890123456789012",
                alerts=[],
            )

    def test_alert_task_serialization(self):
        """Test that AlertTask serializes correctly."""
        task = sy.pagerduty.AlertTask(
            name="test-alert",
            routing_key="12345678901234567890123456789012",
            auto_start=True,
            alerts=[
                sy.pagerduty.AlertConfig(
                    status="db-health",
                    treat_error_as_critical=True,
                    component="postgres",
                    group="infra",
                    alert_class="db_error",
                ),
            ],
        )
        payload = task.to_payload()
        assert payload.config["routing_key"] == "12345678901234567890123456789012"
        assert payload.config["auto_start"] is True
        assert len(payload.config["alerts"]) == 1
        alert = payload.config["alerts"][0]
        assert alert["status"] == "db-health"
        assert alert["class"] == "db_error"
        assert "alert_class" not in alert

    def test_create_and_retrieve_alert_task(self, client: sy.Synnax):
        """Test that AlertTask can be created and retrieved from the database."""
        task = sy.pagerduty.AlertTask(
            name="test-pagerduty-alert",
            routing_key="12345678901234567890123456789012",
            alerts=[
                sy.pagerduty.AlertConfig(
                    status="test-status",
                    enabled=True,
                    treat_error_as_critical=True,
                    component="test-component",
                ),
            ],
        )
        created = client.tasks.create(
            name="test-pagerduty-alert",
            type="pagerduty_alert",
            config=task.config.model_dump(by_alias=True, exclude_none=True),
        )
        tsk = sy.pagerduty.AlertTask(created)
        assert tsk.config.routing_key == task.config.routing_key
        assert tsk.config.auto_start == task.config.auto_start
        assert len(tsk.config.alerts) == len(task.config.alerts)
        for orig, retr in zip(task.config.alerts, tsk.config.alerts):
            assert retr.status == orig.status
            assert retr.enabled == orig.enabled
            assert retr.treat_error_as_critical == orig.treat_error_as_critical
            assert retr.component == orig.component
            assert retr.group == orig.group
            assert retr.alert_class == orig.alert_class

    def test_alert_task_serialization_round_trip(self, client: sy.Synnax):
        """Test that task can be serialized and deserialized correctly."""
        original = sy.pagerduty.AlertTask(
            name="test-round-trip",
            routing_key="abcdefghijklmnopqrstuvwxyz123456",
            auto_start=True,
            alerts=[
                sy.pagerduty.AlertConfig(
                    status="sensor-health",
                    enabled=True,
                    treat_error_as_critical=False,
                    component="temperature",
                    group="sensors",
                    alert_class="anomaly",
                ),
                sy.pagerduty.AlertConfig(
                    status="db-health",
                    enabled=True,
                    treat_error_as_critical=True,
                    component="postgres",
                ),
            ],
        )
        created = client.tasks.create(
            name="test-round-trip",
            type="pagerduty_alert",
            config=original.config.model_dump(by_alias=True, exclude_none=True),
        )
        retrieved = sy.pagerduty.AlertTask(created)
        assert retrieved.config.routing_key == original.config.routing_key
        assert retrieved.config.auto_start == original.config.auto_start
        assert len(retrieved.config.alerts) == len(original.config.alerts)
        for orig, retr in zip(original.config.alerts, retrieved.config.alerts):
            assert retr.status == orig.status
            assert retr.enabled == orig.enabled
            assert retr.treat_error_as_critical == orig.treat_error_as_critical
            assert retr.component == orig.component
