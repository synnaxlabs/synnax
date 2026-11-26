#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""HTTP invalid configuration integration tests.

Each test saves a task with an invalid setting and verifies the driver rejects
it with an error status when the task is started.
"""

from examples.http_sim import HTTPSim

import synnax as sy
from synnax import http
from tests.driver.http_task import create_read_endpoint
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import (
    assert_configure_rejected,
    assert_create_refused,
    assert_start_rejected,
    cleanup_task,
    create_channel,
    create_index,
    run_and_expect_rejection,
)


class HTTPInvalidConfig(SimulatorCase):
    """Verify the driver rejects invalid HTTP task configurations.

    Tests (run sequentially):
        1. Nonexistent device — device key that doesn't exist.
        2. No messages — an endpoint that reads no ICD messages.
        3. Nonexistent channel key — Synnax channel that doesn't exist.
        4. Invalid JSON pointer — pointer to a path that doesn't exist (runtime).
        5. Duplicate channel — two tasks using the same channel.
    """

    sim_classes = [HTTPSim]

    def setup(self) -> None:
        super().setup()
        self.device = self.client.devices.retrieve(name=self.device_name)

    def run(self) -> None:
        self.test_nonexistent_device()
        self.test_no_messages()
        self.test_nonexistent_channel_key()
        self.test_invalid_json_pointer()
        self.test_duplicate_channel()

    def test_nonexistent_device(self) -> None:
        """Configure a read task with a device key that doesn't exist."""
        self.log("Testing: Nonexistent device key")
        idx = create_index(self.client, "http_inv_dev_idx")
        task = sy.http.ReadTask(
            name="HTTP Invalid Device Test",
            device="nonexistent_device_key_12345",
            rate=10,
            endpoints=[
                create_read_endpoint(
                    self.client,
                    path="/api/v1/data",
                    bindings=[
                        (
                            sy.icd.FloatJSONField(
                                name="temperature", pointer="/temperature"
                            ),
                            create_channel(
                                self.client,
                                name="http_inv_dev_ch",
                                data_type=sy.DataType.FLOAT64,
                                index=idx.key,
                            ),
                        )
                    ],
                ),
            ],
        )
        message = assert_configure_rejected(self.client, task, "nonexistent device")
        self.log(f"  Correctly rejected (nonexistent device): {message}")

    def test_no_messages(self) -> None:
        """Start a read task whose endpoint reads no ICD messages."""
        self.log("Testing: No messages")
        task = sy.http.ReadTask(
            name="HTTP No Messages Test",
            device=self.device.key,
            rate=10,
            endpoints=[http.ReadEndpoint(path="/api/v1/data", method="GET")],
        )
        message = assert_start_rejected(self.client, task, "no messages")
        assert "reads no messages" in message, (
            f"Rejection should say the endpoint reads no messages, got {message!r}"
        )
        self.log(f"  Correctly rejected (no messages): {message}")

    def test_nonexistent_channel_key(self) -> None:
        """Save a read task with a Synnax channel key that doesn't exist. The Core
        refuses it before it reaches the driver."""
        self.log("Testing: Nonexistent Synnax channel key")
        task = sy.http.ReadTask(
            name="HTTP Invalid Channel Key Test",
            device=self.device.key,
            rate=10,
            endpoints=[
                create_read_endpoint(
                    self.client,
                    path="/api/v1/data",
                    bindings=[
                        (
                            sy.icd.FloatJSONField(
                                name="temperature", pointer="/temperature"
                            ),
                            999999999,
                        )
                    ],
                ),
            ],
        )
        message = assert_create_refused(self.client, task, "nonexistent channel key")
        assert "channel 999999999 does not exist" in message, (
            f"Refusal should name the missing channel, got {message!r}"
        )
        self.log(f"  Correctly refused (nonexistent channel key): {message}")

    def test_invalid_json_pointer(self) -> None:
        """Start a read task with a JSON pointer that doesn't exist in the response."""
        self.log("Testing: Invalid JSON pointer (runtime)")
        idx = create_index(self.client, "http_inv_ptr_idx")
        task = sy.http.ReadTask(
            name="HTTP Invalid JSON Pointer Test",
            device=self.device.key,
            rate=10,
            endpoints=[
                create_read_endpoint(
                    self.client,
                    path="/api/v1/data",
                    bindings=[
                        (
                            sy.icd.FloatJSONField(
                                name="nested",
                                pointer="/nonexistent/deeply/nested/field",
                            ),
                            create_channel(
                                self.client,
                                name="http_inv_ptr_ch",
                                data_type=sy.DataType.FLOAT64,
                                index=idx.key,
                            ),
                        )
                    ],
                ),
            ],
        )
        message = assert_start_rejected(self.client, task, "invalid JSON pointer")
        assert (
            "Field /nonexistent/deeply/nested/field not found in response from "
            "/api/v1/data" in message
        ), f"Warning should name the missing pointer and the endpoint, got {message!r}"
        self.log(f"  Correctly rejected (invalid JSON pointer): {message}")

    def test_duplicate_channel(self) -> None:
        """Configure and run two tasks that use the same channel."""
        self.log("Testing: Duplicate channel (two tasks on same channel)")
        idx = create_index(self.client, "http_dup_ch_idx")
        shared_ch_key = create_channel(
            self.client,
            name="http_dup_ch",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )

        def _make_task(name: str) -> sy.http.ReadTask:
            return sy.http.ReadTask(
                name=name,
                device=self.device.key,
                rate=10,
                endpoints=[
                    create_read_endpoint(
                        self.client,
                        path="/api/v1/data",
                        bindings=[
                            (
                                sy.icd.FloatJSONField(
                                    name="temperature", pointer="/temperature"
                                ),
                                shared_ch_key,
                            )
                        ],
                    ),
                ],
            )

        task_a = _make_task("HTTP Dup Channel Task A")
        task_b = _make_task("HTTP Dup Channel Task B")
        self.client.tasks.configure(task_a)
        self.log("  Task A configured")

        rejected = False
        try:
            with task_a.run():
                self.log("  Task A running")
                self.client.tasks.configure(task_b)
                self.log("  Task B configured (attempting run)")
                message = run_and_expect_rejection(self.client, task_b)
                if message is not None:
                    self.log(f"  Correctly rejected on run: {message}")
                    rejected = True
        finally:
            cleanup_task(self.client, task_a)
            cleanup_task(self.client, task_b)

        if not rejected:
            self.fail(
                "Driver did not reject second task using the "
                "same channel — both tasks ran simultaneously"
            )
