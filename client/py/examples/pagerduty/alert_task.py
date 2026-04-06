#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to set up a PagerDuty alert task that watches a
Synnax status and triggers a PagerDuty incident when the status changes.

Before running this example:
1. Ensure a Synnax server is running on localhost:9090
2. Replace ROUTING_KEY with your PagerDuty Events API v2 Integration Key

Usage:
    uv run python examples/pagerduty/alert_task.py
"""

import synnax as sy

ROUTING_KEY = "12345678901234567890123456789012"  # Example key, replace with your own

client = sy.Synnax()

# Create a status to watch
STATUS_KEY = "example-pagerduty-status"
client.statuses.set(
    sy.Status(
        key=STATUS_KEY,
        name="Example Service",
        variant="success",
        message="Service is healthy",
    )
)

# Create the PagerDuty alert task
alert_task = sy.pagerduty.AlertTask(
    name="Example PagerDuty Alert",
    routing_key=ROUTING_KEY,
    auto_start=False,
    alerts=[
        sy.pagerduty.AlertConfig(
            status=STATUS_KEY,
            enabled=True,
            treat_error_as_critical=True,
            component="example-service",
            group="examples",
            alert_class="service_error",
        ),
    ],
)

# Configure and start the task
client.tasks.configure(alert_task)

print("Starting PagerDuty alert task...")
print(f"Watching status: {STATUS_KEY}")
print("Press Ctrl+C to stop\n")

try:
    with alert_task.run():
        # Simulate a status change to trigger an alert
        print("Setting status to ERROR to trigger a PagerDuty incident...")
        client.statuses.set(
            sy.Status(
                key=STATUS_KEY,
                name="Example Service",
                variant="error",
                message="Service is down",
                description="Connection to database lost",
            )
        )
        sy.sleep(5)

        # Resolve by setting status back to success
        print("Setting status to SUCCESS to resolve the incident...")
        client.statuses.set(
            sy.Status(
                key=STATUS_KEY,
                name="Example Service",
                variant="success",
                message="Service recovered",
            )
        )
        sy.sleep(3)

        print("Done! Check your PagerDuty dashboard for the incident.")

except KeyboardInterrupt:
    print("\nStopped by user")
