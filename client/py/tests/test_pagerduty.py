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
                            "key": "alert-1",
                            "status": "database-health",
                            "disabled": False,
                            "errors_critical": True,
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
                            "key": "alert-1",
                            "status": "sensor-1",
                            "disabled": False,
                            "errors_critical": False,
                            "component": "temperature-sensor",
                            "group": "hardware",
                            "class": "sensor_anomaly",
                        },
                        {
                            "key": "alert-2",
                            "status": "sensor-2",
                            "disabled": True,
                            "errors_critical": True,
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
                            "key": "alert-1",
                            "status": "my-status",
                            "disabled": False,
                        },
                    ],
                },
            },
        ],
    )
    def test_parse_alert_task_config(self, test_data):
        """Test that TaskConfig can parse various configurations."""
        sy.pagerduty.TaskConfig.model_validate(test_data["data"])

    def test_alert_defaults(self):
        """Test that Alert has correct defaults."""
        alert = sy.pagerduty.Alert(status="my-status")
        assert alert.status == "my-status"
        assert alert.errors_critical is False
        assert alert.component == ""
        assert alert.group == ""
        assert alert.class_ == ""
        assert alert.disabled is False

    def test_alert_class_alias(self):
        """Test that Alert serializes class_ as 'class' in JSON."""
        alert = sy.pagerduty.Alert(
            status="my-status",
            class_="cpu_load",
        )
        dumped = alert.model_dump(by_alias=True)
        assert "class" in dumped
        assert dumped["class"] == "cpu_load"
        assert "class_" not in dumped

    def test_alert_class_from_alias(self):
        """Test that Alert can be created from JSON with 'class' key."""
        alert = sy.pagerduty.Alert.model_validate(
            {"status": "my-status", "class": "ping_failure", "disabled": False}
        )
        assert alert.class_ == "ping_failure"

    def test_task_config_defaults(self):
        """Test that TaskConfig has correct defaults."""
        config = sy.pagerduty.TaskConfig(
            routing_key="12345678901234567890123456789012",
            alerts=[sy.pagerduty.Alert(status="s")],
        )
        assert config.auto_start is False

    def test_alert_task_auto_key_generation(self):
        """Test that the AlertTask assigns keys to alerts missing one."""
        task = sy.pagerduty.AlertTask(
            name="test",
            routing_key="12345678901234567890123456789012",
            alerts=[sy.pagerduty.Alert(status="s")],
        )
        alert = task.config.alerts[0]
        assert alert.key != ""
        assert len(alert.key) > 0

    def test_alert_task_serialization(self):
        """Test that AlertTask serializes correctly."""
        task = sy.pagerduty.AlertTask(
            name="test-alert",
            routing_key="12345678901234567890123456789012",
            auto_start=True,
            alerts=[
                sy.pagerduty.Alert(
                    status="db-health",
                    errors_critical=True,
                    component="postgres",
                    group="infra",
                    class_="db_error",
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
        assert "class_" not in alert

    def test_create_and_retrieve_alert_task(self, client: sy.Synnax):
        """Test that AlertTask can be created and retrieved from the database."""
        task = sy.pagerduty.AlertTask(
            name="test-pagerduty-alert",
            routing_key="12345678901234567890123456789012",
            alerts=[
                sy.pagerduty.Alert(
                    status="test-status",
                    disabled=False,
                    errors_critical=True,
                    component="test-component",
                ),
            ],
        )
        created = client.tasks.create(
            name="test-pagerduty-alert",
            type="pagerduty_alert",
            config=task.config,
        )
        tsk = sy.pagerduty.AlertTask(created)
        assert tsk.config.routing_key == task.config.routing_key
        assert tsk.config.auto_start == task.config.auto_start
        assert len(tsk.config.alerts) == len(task.config.alerts)
        for orig, retr in zip(task.config.alerts, tsk.config.alerts):
            assert retr.status == orig.status
            assert retr.disabled == orig.disabled
            assert retr.errors_critical == orig.errors_critical
            assert retr.component == orig.component
            assert retr.group == orig.group
            assert retr.class_ == orig.class_

    def test_alert_task_serialization_round_trip(self, client: sy.Synnax):
        """Test that task can be serialized and deserialized correctly."""
        original = sy.pagerduty.AlertTask(
            name="test-round-trip",
            routing_key="abcdefghijklmnopqrstuvwxyz123456",
            auto_start=True,
            alerts=[
                sy.pagerduty.Alert(
                    status="sensor-health",
                    disabled=False,
                    errors_critical=False,
                    component="temperature",
                    group="sensors",
                    class_="anomaly",
                ),
                sy.pagerduty.Alert(
                    status="db-health",
                    disabled=False,
                    errors_critical=True,
                    component="postgres",
                ),
            ],
        )
        created = client.tasks.create(
            name="test-round-trip",
            type="pagerduty_alert",
            config=original.config,
        )
        retrieved = sy.pagerduty.AlertTask(created)
        assert retrieved.config.routing_key == original.config.routing_key
        assert retrieved.config.auto_start == original.config.auto_start
        assert len(retrieved.config.alerts) == len(original.config.alerts)
        for orig, retr in zip(original.config.alerts, retrieved.config.alerts):
            assert retr.status == orig.status
            assert retr.disabled == orig.disabled
            assert retr.errors_critical == orig.errors_critical
            assert retr.component == orig.component
