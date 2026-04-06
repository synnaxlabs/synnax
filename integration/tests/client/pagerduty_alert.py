#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
from random import randint

import pagerduty
import synnax as sy

from framework.test_case import TestCase

ROUTING_KEY = os.environ.get("PAGERDUTY_ROUTING_KEY")
API_KEY = os.environ.get("PAGERDUTY_API_KEY")

if ROUTING_KEY is None or API_KEY is None:
    raise RuntimeError("PAGERDUTY_ROUTING_KEY and PAGERDUTY_API_KEY must be set")


class PagerDutyAlert(TestCase):
    """Integration test that configures a PagerDuty alert task, triggers an incident
    via a Synnax status change, and verifies the incident was created in PagerDuty."""

    events: pagerduty.EventsApiV2Client
    rest: pagerduty.RestApiV2Client

    def setup(self) -> None:
        self.set_manual_timeout(60)
        self.suffix = randint(0, 1000000)
        self.status_key = f"test-pd-status-{self.suffix}"
        self.events = pagerduty.EventsApiV2Client(ROUTING_KEY)
        self.rest = pagerduty.RestApiV2Client(API_KEY)
        super().setup()

    def run(self) -> None:
        client = self.client

        # 1. Verify PagerDuty Events API connectivity
        self.log("Verifying PagerDuty Events API connectivity...")
        connectivity_key = self.events.trigger(
            f"Connectivity check {self.suffix}",
            "integration-test",
        )
        self.events.resolve(connectivity_key)
        self.log("PagerDuty Events API connectivity verified")

        # 2. Create a status in Synnax
        self.log(f"Creating status: {self.status_key}")
        client.statuses.set(
            sy.Status(
                key=self.status_key,
                name=f"Test PD Status {self.suffix}",
                variant="success",
                message="All systems operational",
            )
        )

        # 3. Find the Go driver rack (where PagerDuty factory runs)
        self.log("Finding Go driver rack...")
        go_rack = client.racks.retrieve(name="Node 1")
        self.log(f"Using Go driver rack: key={go_rack.key}, name={go_rack.name}")

        # 4. Create the PagerDuty alert task on the Go driver rack.
        # We stream sy_status_set to wait for the factory's configuration
        # acknowledgment instead of sleeping.
        self.log("Creating PagerDuty alert task...")
        task_key = (go_rack.key << 32) + 0
        streamer = client.open_streamer(["sy_status_set"])
        created = client.tasks.create(
            key=task_key,
            name=f"PD Integration Test {self.suffix}",
            type="pagerduty_alert",
            config=sy.pagerduty.AlertTaskConfig(
                routing_key=ROUTING_KEY,
                auto_start=True,
                alerts=[
                    sy.pagerduty.AlertConfig(
                        status=self.status_key,
                        enabled=True,
                        treat_error_as_critical=True,
                        component="integration-test",
                        group="ci",
                        alert_class="test_alert",
                    ),
                ],
            ).model_dump(by_alias=True, exclude_none=True),
        )
        self.log(f"Alert task created: key={created.key}")

        # Wait for the Go driver to configure and auto-start the task
        task_ontology_key = f"task:{created.key}"
        timer = sy.Timer()
        configured = False
        while timer.elapsed() < 10 * sy.TimeSpan.SECOND:
            frame = streamer.read(1)
            if frame is None or "sy_status_set" not in frame:
                continue
            for val in frame["sy_status_set"]:
                if (
                    isinstance(val, dict)
                    and val.get("key") == task_ontology_key
                    and val.get("variant") == "success"
                ):
                    self.log(f"Task acknowledged: {val.get('message')}")
                    configured = True
                    break
            if configured:
                break
        streamer.close()
        if not configured:
            self.fail("Timed out waiting for Go driver to configure the task")
            return

        # 5. Trigger an alert by setting the status to error
        error_message = f"Integration test error {self.suffix}"
        self.log(f"Setting status to ERROR: {error_message}")
        client.statuses.set(
            sy.Status(
                key=self.status_key,
                name=f"Test PD Status {self.suffix}",
                variant="error",
                message=error_message,
                description="Test incident from the integration test framework.",
            )
        )

        # 6. Poll PagerDuty for the incident (up to 10s, checking every 2s)
        self.log("Polling PagerDuty for triggered incident...")
        matching = []
        timer = sy.Timer()
        while timer.elapsed() < 10 * sy.TimeSpan.SECOND:
            incidents = self.rest.list_all(
                "incidents", params={"statuses[]": "triggered"}
            )
            matching = [
                inc
                for inc in incidents
                if error_message in inc.get("title", "")
                or self.status_key in (inc.get("incident_key") or "")
            ]
            if matching:
                break
            sy.sleep(2)

        if not matching:
            self.fail(
                f"Expected to find a PagerDuty incident for '{self.status_key}', "
                f"but none was found after 10s."
            )
            return
        self.log(f"Found {len(matching)} matching incident(s) in PagerDuty")

        # 7. Resolve the status and verify PagerDuty resolves the incident
        self.log("Setting status to SUCCESS to resolve the incident...")
        client.statuses.set(
            sy.Status(
                key=self.status_key,
                name=f"Test PD Status {self.suffix}",
                variant="success",
                message="All systems operational again",
            )
        )

        self.log("Polling PagerDuty for incident resolution...")
        incident_id = matching[0]["id"]
        resolved = False
        timer = sy.Timer()
        while timer.elapsed() < 10 * sy.TimeSpan.SECOND:
            incident = self.rest.rget(f"/incidents/{incident_id}")
            if incident.get("status") == "resolved":
                resolved = True
                break
            sy.sleep(2)

        if not resolved:
            self.log("Warning: incident was not resolved in PagerDuty within 10s")
        else:
            self.log("Incident resolved in PagerDuty")

    def teardown(self) -> None:
        self.events.resolve(self.status_key)
        self.log("Cleanup: resolved PagerDuty event")
        super().teardown()
